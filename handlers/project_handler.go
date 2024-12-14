package handlers

import (
	"api-key-limiter/models"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

var ErrInvalidProjectIdAndAccessKeyCombination = errors.New("failed to validate given project id and access key")
var ErrConfigDoesNotExist = errors.New("config with the given id does not exist")

type ProjectHandler struct {
	db *sql.DB
}

func NewProjectHandler(db *sql.DB) *ProjectHandler {
	return &ProjectHandler{db}
}

func (p *ProjectHandler) ValidateProjectIdAndAccessKey(projectID string, accessKey string) error {
	query := `
		SELECT id 
		FROM projects
		WHERE id = $1 AND access_key = $2 
	`

	var project models.Project
	if err := p.db.QueryRow(query, projectID, accessKey).Scan(&project.ID); err != nil {
		if err == sql.ErrNoRows {
			return ErrInvalidProjectIdAndAccessKeyCombination
		}

		return fmt.Errorf("failed to query project data: %w", err)
	}

	return nil
}

func (p *ProjectHandler) GetConfig(projectID string, configID string) (*models.Config, error) {
	query := `
		SELECT id, project_id, header_name, header_value
		FROM configs
		WHERE id = $1 AND project_id = $2 
	`

	var config models.Config
	if err := p.db.QueryRow(query, configID, projectID).Scan(
		&config.ID, &config.ProjectID, &config.HeaderName, &config.HeaderValue,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrConfigDoesNotExist
		}

		return nil, fmt.Errorf("failed to query config data: %w", err)
	}

	return &config, nil
}
