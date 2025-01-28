package handlers

import (
	"api-key-limiter/models"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
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
		SELECT id, project_id, limit_requests_count, limit_duration
		FROM configs
		WHERE id = $1 AND project_id = $2 
	`

	var config models.Config
	if err := p.db.QueryRow(query, configID, projectID).Scan(
		&config.ID, &config.ProjectID, &config.LimitNumberOfRequests, &config.LimitPer,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrConfigDoesNotExist
		}

		return nil, fmt.Errorf("failed to query config data: %w", err)
	}

	headers, headersErr := p.ListHeaderReplacements(config.ID)
	if headersErr != nil {
		return nil, headersErr
	}

	config.HeaderReplacements = headers
	return &config, nil
}

func (p *ProjectHandler) ListHeaderReplacements(configID uuid.UUID) ([]models.HeaderReplacement, error) {
	query := `
		SELECT id, config_id, header_name, header_value
		FROM header_replacements
		WHERE config_id = $1
	`
	rows, err := p.db.Query(query, configID)
	if err != nil {
		return nil, fmt.Errorf("failed to query header replacements: %v", err)
	}
	defer rows.Close()

	var replacements []models.HeaderReplacement
	for rows.Next() {
		var replacement models.HeaderReplacement
		if err := rows.Scan(
			&replacement.ID, &replacement.ConfigID, &replacement.HeaderName, &replacement.HeaderValue,
		); err != nil {
			return nil, fmt.Errorf("failed to scan config row: %v", err)
		}

		replacements = append(replacements, replacement)
	}

	return replacements, nil
}
