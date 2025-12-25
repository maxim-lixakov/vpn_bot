package repo

import (
	"context"
	"database/sql"
	"time"
)

type CountryToAdd struct {
	ID             int64
	UserID         int64
	SubscriptionID sql.NullInt64
	Text           string
	CreatedAt      time.Time
}

type CountriesToAddRepo struct{ db *sql.DB }

type CountriesToAddRepoInterface interface {
	Insert(ctx context.Context, userID int64, subscriptionID sql.NullInt64, text string) error
}

func NewCountriesToAddRepo(db *sql.DB) CountriesToAddRepoInterface {
	return &CountriesToAddRepo{db: db}
}

func (r *CountriesToAddRepo) Insert(ctx context.Context, userID int64, subscriptionID sql.NullInt64, text string) error {
	var subID any = nil
	if subscriptionID.Valid {
		subID = subscriptionID.Int64
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO countries_to_add(user_id, subscription_id, request_text)
		VALUES ($1,$2,$3)
	`, userID, subID, text)
	return err
}
