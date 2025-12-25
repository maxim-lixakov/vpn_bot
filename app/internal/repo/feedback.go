package repo

import (
	"context"
	"database/sql"
	"time"
)

type Feedback struct {
	ID        int64
	UserID    int64
	Text      string
	CreatedAt time.Time
}

type FeedbackRepo struct{ db *sql.DB }

type FeedbackRepoInterface interface {
	Insert(ctx context.Context, userID int64, text string) error
}

func NewFeedbackRepo(db *sql.DB) FeedbackRepoInterface {
	return &FeedbackRepo{db: db}
}

func (r *FeedbackRepo) Insert(ctx context.Context, userID int64, text string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO feedback(user_id, text, created_at)
		VALUES ($1, $2, now())
	`, userID, text)
	return err
}
