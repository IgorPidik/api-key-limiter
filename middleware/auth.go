package middleware

import (
	"api-key-limiter/handlers"
	"context"
	"fmt"
	"net/http"
)

type ParsedAuthHeader struct {
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

		fmt.Printf("project id: %s\n", "test")
		fmt.Printf("project header: %v\n", r.Header)
		authHeader, parsingErr := ParseAuthHeader(r.Header.Get("Proxy-Authorization"))
		if parsingErr != nil {
			http.Error(w, "Invalid auth header", http.StatusBadRequest)
			return
		}

		validationErr := a.projectHandler.ValidateProjectIdAndAccessKey(authHeader.ProjectID, authHeader.AccessKey)
		if validationErr != nil {
			fmt.Printf("failed to validate request auth: %w\n", validationErr)
			if validationErr == handlers.ErrInvalidProjectIdAndAccessKeyCombination {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		fmt.Printf("project id: %s\n", authHeader.ProjectID)

		// pass project ID to handlers
		ctx := context.WithValue(r.Context(), "ProjectID", authHeader.ProjectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
