package db

import (
	"context"

	"github.com/google/uuid"
)

type User struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	GitHubID  *string `json:"github_id,omitempty"`
	Name      *string `json:"name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

func GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := Pool.QueryRow(ctx,
		`SELECT id::text, email, github_id, name, avatar_url FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.GitHubID, &u.Name, &u.AvatarURL)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := Pool.QueryRow(ctx,
		`SELECT id::text, email, github_id, name, avatar_url FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.GitHubID, &u.Name, &u.AvatarURL)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func CreateUser(ctx context.Context, email, name string) (*User, error) {
	id := uuid.New().String()
	_, err := Pool.Exec(ctx,
		`INSERT INTO users (id, email, name) VALUES ($1, $2, $3)`,
		id, email, name,
	)
	if err != nil {
		return nil, err
	}
	return &User{ID: id, Email: email, Name: &name}, nil
}

func GetUserByGitHubID(ctx context.Context, githubID string) (*User, error) {
	var u User
	err := Pool.QueryRow(ctx,
		`SELECT id::text, email, github_id, name, avatar_url FROM users WHERE github_id = $1`,
		githubID,
	).Scan(&u.ID, &u.Email, &u.GitHubID, &u.Name, &u.AvatarURL)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
