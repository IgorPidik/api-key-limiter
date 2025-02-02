package middleware

import (
	"api-key-limiter/handlers"
	"context"
	"log"
	"net/http"
)

type ParsedAuthHeader struct {
	ConfigID  string
	ProjectID string
	AccessKey string
}

type AuthMiddleware struct {
	projectHandler *handlers.ProjectHandler
}

func NewAuthMiddleware(projectHandler *handlers.ProjectHandler) *AuthMiddleware {
	return &AuthMiddleware{projectHandler}
}

func (a *AuthMiddleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader, parsingErr := ParseAuthHeader(r.Header.Get("Proxy-Authorization"))
		if parsingErr != nil {
			http.Error(w, "Invalid auth header", http.StatusBadRequest)
			return
		}

		validationErr := a.projectHandler.ValidateProjectIdAndAccessKey(authHeader.ProjectID, authHeader.AccessKey)
		if validationErr != nil {
			log.Printf("failed to validate request auth: %v\n", validationErr)
			if validationErr == handlers.ErrInvalidProjectIdAndAccessKeyCombination {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// pass project & config IDs to handlers
		ctxWithProjectID := context.WithValue(r.Context(), "ProjectID", authHeader.ProjectID)
		ctxWithConfigID := context.WithValue(ctxWithProjectID, "ConfigID", authHeader.ConfigID)
		next.ServeHTTP(w, r.WithContext(ctxWithConfigID))
	})
}
