package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Agent struct {
	ID          string           `json:"id"`
	Name        string          `json:"name"`
	DisplayName string          `json:"display_name,omitempty"`
	Description string          `json:"description,omitempty"`
	AuthorID    string          `json:"author_id"`
	Version     string          `json:"version"`
	Category    string          `json:"category,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	Manifest    json.RawMessage `json:"manifest,omitempty"`
	IsPublic    bool            `json:"is_public"`
	IsHosted    bool            `json:"is_hosted"`
	PriceCents  *int            `json:"price_cents,omitempty"`
	CreatedAt   string          `json:"created_at"`
}

type AgentVersion struct {
	ID          string `json:"id"`
	AgentID     string `json:"agent_id"`
	Version     string `json:"version"`
	Changelog   string `json:"changelog,omitempty"`
	ArtifactURL string `json:"artifact_url,omitempty"`
	Checksum    string `json:"checksum,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func ListAgents(ctx context.Context, query, category string, limit, offset int) ([]Agent, error) {
	sql := `SELECT id::text, name, COALESCE(display_name,''), COALESCE(description,''),
	        author_id::text, version, COALESCE(category,''), tags, manifest,
	        is_public, is_hosted, price_cents, created_at::text
	        FROM agents WHERE is_public = true`
	var args []interface{}

	if query != "" {
		sql += ` AND (LOWER(name) LIKE $1 OR LOWER(description) LIKE $1)`
		args = append(args, "%"+strings.ToLower(query)+"%")
	}
	if category != "" {
		paramIdx := len(args) + 1
		sql += fmt.Sprintf(` AND category = $%d`, paramIdx)
		args = append(args, category)
	}

	paramIdx := len(args) + 1
	sql += fmt.Sprintf(` ORDER BY download_count DESC LIMIT $%d OFFSET $%d`, paramIdx, paramIdx+1)
	args = append(args, limit, offset)

	rows, err := Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var a Agent
		var tags *string
		if err := rows.Scan(&a.ID, &a.Name, &a.DisplayName, &a.Description,
			&a.AuthorID, &a.Version, &a.Category, &tags, &a.Manifest,
			&a.IsPublic, &a.IsHosted, &a.PriceCents, &a.CreatedAt); err != nil {
			return nil, err
		}
		if tags != nil {
			a.Tags = splitTags(*tags)
		}
		agents = append(agents, a)
	}
	return agents, nil
}

func GetAgentByName(ctx context.Context, name string) (*Agent, error) {
	var a Agent
	var tags *string
	err := Pool.QueryRow(ctx,
		`SELECT id::text, name, COALESCE(display_name,''), COALESCE(description,''),
		        author_id::text, version, COALESCE(category,''), tags, manifest,
		        is_public, is_hosted, price_cents, created_at::text
		 FROM agents WHERE name = $1`, name,
	).Scan(&a.ID, &a.Name, &a.DisplayName, &a.Description,
		&a.AuthorID, &a.Version, &a.Category, &tags, &a.Manifest,
		&a.IsPublic, &a.IsHosted, &a.PriceCents, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	if tags != nil {
		a.Tags = splitTags(*tags)
	}
	return &a, nil
}

func GetAgentByID(ctx context.Context, id string) (*Agent, error) {
	var a Agent
	var tags *string
	err := Pool.QueryRow(ctx,
		`SELECT id::text, name, COALESCE(display_name,''), COALESCE(description,''),
		        author_id::text, version, COALESCE(category,''), tags, manifest,
		        is_public, is_hosted, price_cents, created_at::text
		 FROM agents WHERE id = $1`, id,
	).Scan(&a.ID, &a.Name, &a.DisplayName, &a.Description,
		&a.AuthorID, &a.Version, &a.Category, &tags, &a.Manifest,
		&a.IsPublic, &a.IsHosted, &a.PriceCents, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	if tags != nil {
		a.Tags = splitTags(*tags)
	}
	return &a, nil
}

func CreateAgent(ctx context.Context, name, authorID, version, displayName, description, category string, tags []string, manifest json.RawMessage, isHosted bool, priceCents *int) (*Agent, error) {
	id := uuid.New().String()
	var tagsStr *string
	if len(tags) > 0 {
		s := strings.Join(tags, ",")
		tagsStr = &s
	}
	var row struct{ ID string }
	err := Pool.QueryRow(ctx,
		`INSERT INTO agents (id, name, display_name, description, author_id, version, category, tags, manifest, is_public, is_hosted, price_cents)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,false,$10,$11)
		 RETURNING id::text`,
		id, name, displayName, description, authorID, version, category, tagsStr, manifest, isHosted, priceCents,
	).Scan(&row.ID)
	if err != nil {
		return nil, err
	}
	return &Agent{
		ID:          id,
		Name:        name,
		DisplayName: displayName,
		Description: description,
		AuthorID:    authorID,
		Version:     version,
		Category:    category,
		Tags:        tags,
		Manifest:    manifest,
		IsPublic:    false,
		IsHosted:    isHosted,
		PriceCents:  priceCents,
	}, nil
}

func CreateAgentVersion(ctx context.Context, agentID, version, changelog, artifactURL, checksum string) (*AgentVersion, error) {
	id := uuid.New().String()
	_, err := Pool.Exec(ctx,
		`INSERT INTO agent_versions (id, agent_id, version, changelog, artifact_url, checksum)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (agent_id, version) DO UPDATE SET changelog=$4, artifact_url=$5, checksum=$6`,
		id, agentID, version, changelog, artifactURL, checksum,
	)
	if err != nil {
		return nil, err
	}
	return &AgentVersion{
		ID:          id,
		AgentID:     agentID,
		Version:     version,
		Changelog:   changelog,
		ArtifactURL: artifactURL,
		Checksum:    checksum,
	}, nil
}

func ListAgentVersions(ctx context.Context, agentID string) ([]AgentVersion, error) {
	rows, err := Pool.Query(ctx,
		`SELECT id::text, agent_id::text, version, COALESCE(changelog,''), COALESCE(artifact_url,''), COALESCE(checksum,''), created_at::text
		 FROM agent_versions WHERE agent_id = $1 ORDER BY created_at DESC`,
		agentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []AgentVersion
	for rows.Next() {
		var v AgentVersion
		if err := rows.Scan(&v.ID, &v.AgentID, &v.Version, &v.Changelog, &v.ArtifactURL, &v.Checksum, &v.CreatedAt); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, nil
}

func GetAgentVersion(ctx context.Context, agentID, version string) (*AgentVersion, error) {
	var v AgentVersion
	err := Pool.QueryRow(ctx,
		`SELECT id::text, agent_id::text, version, COALESCE(changelog,''), COALESCE(artifact_url,''), COALESCE(checksum,''), created_at::text
		 FROM agent_versions WHERE agent_id = $1 AND version = $2`,
		agentID, version,
	).Scan(&v.ID, &v.AgentID, &v.Version, &v.Changelog, &v.ArtifactURL, &v.Checksum, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}
