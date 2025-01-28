package models

import "github.com/google/uuid"

type HeaderReplacement struct {
	ID          uuid.UUID
	ConfigID    uuid.UUID
	HeaderName  string
	HeaderValue string
}
