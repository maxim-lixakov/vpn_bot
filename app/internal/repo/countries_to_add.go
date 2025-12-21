package repo

import (
	"context"
	"database/sql"
	"time"
)

type CountryToAdd struct {
	ID        int64
	UserID    int64
	Text      string
	CreatedAt time.Time
}

type CountriesToAddRepo struct{ db *sql.DB }

type CountriesToAddRepoInterface interface {
	Insert(ctx context.Context, userID int64, text string) error
}

func NewCountriesToAddRepo(db *sql.DB) CountriesToAddRepoInterface {
	return &CountriesToAddRepo{db: db}
}

func (r *CountriesToAddRepo) Insert(ctx context.Context, userID int64, text string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO countries_to_add(user_id, request_text)
		VALUES ($1,$2)
	`, userID, text)
	return err
}
