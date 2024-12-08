package handlers

import (
	"api-key-limiter/models"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type ProjectHandler struct {
	db *sql.DB
}

func NewProjectHandler(db *sql.DB) *ProjectHandler {
	return &ProjectHandler{db}
}

func (p *ProjectHandler) GetConfigs(projectID string) ([]models.Config, error) {
	query := `
		SELECT id, project_id, header_name, header_value
		FROM configs
		WHERE project_id = $1
	`

	rows, err := p.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("query configs failed: %w", err)
	}
	defer rows.Close()

	var configs []models.Config
	for rows.Next() {
		var config models.Config
		if err := rows.Scan(&config.ID, &config.ProjectID, &config.HeaderName, &config.HeaderValue); err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}
		configs = append(configs, config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return configs, nil
}
