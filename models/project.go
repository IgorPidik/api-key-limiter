package models

import "github.com/google/uuid"

type Project struct {
	ID          uuid.UUID
	Name        string
	Description string
	UserID      uuid.UUID
	AccessKey   string
	Configs     []Config
}
