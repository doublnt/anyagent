package db

import (
	"context"
	"time"
)

type Memory struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Kind      string    `json:"kind"`
	Content   string    `json:"content"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

func ListMemories(ctx context.Context, projectID string, limit, offset int) ([]Memory, error) {
	rows, err := Pool.Query(ctx,
		`SELECT id::text, project_id::text, kind, content, source, created_at
		 FROM memories WHERE project_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		projectID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var m Memory
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Kind, &m.Content, &m.Source, &m.CreatedAt); err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}
	return memories, nil
}

func CreateMemory(ctx context.Context, projectID, kind, content, source string) (*Memory, error) {
	var m Memory
	err := Pool.QueryRow(ctx,
		`INSERT INTO memories (project_id, kind, content, source)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id::text, project_id::text, kind, content, source, created_at`,
		projectID, kind, content, source,
	).Scan(&m.ID, &m.ProjectID, &m.Kind, &m.Content, &m.Source, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func SearchMemories(ctx context.Context, projectID, query string, limit int) ([]Memory, error) {
	// Full-text search (vector search requires pgvector extension)
	rows, err := Pool.Query(ctx,
		`SELECT id::text, project_id::text, kind, content, source, created_at
		 FROM memories
		 WHERE project_id = $1 AND content ILIKE '%' || $2 || '%'
		 ORDER BY created_at DESC LIMIT $3`,
		projectID, query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var m Memory
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Kind, &m.Content, &m.Source, &m.CreatedAt); err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}
	return memories, nil
}

func DeleteMemory(ctx context.Context, memoryID string) error {
	_, err := Pool.Exec(ctx, `DELETE FROM memories WHERE id = $1`, memoryID)
	return err
}
