package models

import (
	"github.com/google/uuid"
)

type Config struct {
	ID                    uuid.UUID
	ProjectID             uuid.UUID
	Name                  string
	LimitNumberOfRequests int
	LimitPer              string
	HeaderReplacements    []HeaderReplacement
}
