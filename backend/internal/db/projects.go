package db

import (
	"context"
	"time"
)

type Project struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	RepoURL   *string   `json:"repo_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func ListProjects(ctx context.Context, userID string) ([]Project, error) {
	rows, err := Pool.Query(ctx,
		`SELECT id::text, user_id::text, name, repo_url, created_at
		 FROM projects WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.RepoURL, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func CreateProject(ctx context.Context, userID, name string, repoURL *string) (*Project, error) {
	var p Project
	err := Pool.QueryRow(ctx,
		`INSERT INTO projects (user_id, name, repo_url)
		 VALUES ($1, $2, $3)
		 RETURNING id::text, user_id::text, name, repo_url, created_at`,
		userID, name, repoURL,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.RepoURL, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetProject(ctx context.Context, projectID string) (*Project, error) {
	var p Project
	err := Pool.QueryRow(ctx,
		`SELECT id::text, user_id::text, name, repo_url, created_at
		 FROM projects WHERE id = $1`,
		projectID,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.RepoURL, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
