package middleware

import (
	"encoding/base64"
	"errors"
	"strings"
)

func ParseAuthHeader(authHeader string) (*ParsedAuthHeader, error) {
	if len(authHeader) < 1 {
		return nil, errors.New("auth header is empty")
	}

	headerParts := strings.Split(authHeader, " ")
	if len(headerParts) != 2 && headerParts[0] != "Basic" {
		return nil, errors.New("invalid auth header format")
	}

	data, err := base64.StdEncoding.DecodeString(headerParts[1])
	if err != nil {
		return nil, err
	}

	authParts := strings.Split(string(data), ":")
	if len(authParts) != 3 {
		return nil, errors.New("invalid auth header format")
	}

	return &ParsedAuthHeader{
		authParts[0],
		authParts[1],
		authParts[2],
	}, nil
}
