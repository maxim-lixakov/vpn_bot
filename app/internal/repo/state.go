package repo

import (
	"context"
	"database/sql"
	"time"
)

type UserState struct {
	UserID          int64
	State           string
	SelectedCountry sql.NullString
	UpdatedAt       time.Time
}

type StateRepo struct{ db *sql.DB }

type StateRepoInterface interface {
	EnsureDefault(ctx context.Context, userID int64, defaultState string) (UserState, error)
	Get(ctx context.Context, userID int64) (UserState, error)
	Set(ctx context.Context, userID int64, state string, selectedCountry sql.NullString) (UserState, error)
}

func NewStateRepo(db *sql.DB) StateRepoInterface { return &StateRepo{db: db} }

func (r *StateRepo) EnsureDefault(ctx context.Context, userID int64, defaultState string) (UserState, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO user_states(user_id, router, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (user_id) DO NOTHING
		RETURNING user_id, router, selected_country, updated_at
	`, userID, defaultState)

	var st UserState
	err := row.Scan(&st.UserID, &st.State, &st.SelectedCountry, &st.UpdatedAt)
	if err == nil {
		return st, nil
	}

	// already exists -> load
	return r.Get(ctx, userID)
}

func (r *StateRepo) Get(ctx context.Context, userID int64) (UserState, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT user_id, router, selected_country, updated_at
		FROM user_states WHERE user_id=$1
	`, userID)

	var st UserState
	if err := row.Scan(&st.UserID, &st.State, &st.SelectedCountry, &st.UpdatedAt); err != nil {
		return UserState{}, err
	}
	return st, nil
}

func (r *StateRepo) Set(ctx context.Context, userID int64, state string, selectedCountry sql.NullString) (UserState, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE user_states
		SET router=$2, selected_country=$3, updated_at=now()
		WHERE user_id=$1
		RETURNING user_id, router, selected_country, updated_at
	`, userID, state, selectedCountry)

	var st UserState
	if err := row.Scan(&st.UserID, &st.State, &st.SelectedCountry, &st.UpdatedAt); err != nil {
		return UserState{}, err
	}
	return st, nil
}
