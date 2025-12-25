package repo

import (
	"context"
	"database/sql"
	"time"
)

type AccessKey struct {
	ID           int64
	UserID       int64
	Country      string
	OutlineKeyID string
	AccessURL    string
	CreatedAt    time.Time
	RevokedAt    sql.NullTime
}

type AccessKeysRepo struct{ db *sql.DB }

type AccessKeysRepoInterface interface {
	GetActive(ctx context.Context, userID int64, country string) (AccessKey, bool, error)
	Insert(ctx context.Context, userID int64, country, outlineKeyID, accessURL string) (int64, error)
	Revoke(ctx context.Context, id int64, at time.Time) error
}

func NewAccessKeysRepo(db *sql.DB) AccessKeysRepoInterface { return &AccessKeysRepo{db: db} }

func (r *AccessKeysRepo) GetActive(ctx context.Context, userID int64, country string) (AccessKey, bool, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, country_code, outline_key_id, access_url, created_at, revoked_at
		FROM access_keys
		WHERE user_id=$1 AND country_code=$2 AND revoked_at IS NULL
		LIMIT 1
	`, userID, country)

	var k AccessKey
	err := row.Scan(&k.ID, &k.UserID, &k.Country, &k.OutlineKeyID, &k.AccessURL, &k.CreatedAt, &k.RevokedAt)
	if err == sql.ErrNoRows {
		return AccessKey{}, false, nil
	}
	if err != nil {
		return AccessKey{}, false, err
	}
	return k, true, nil
}

func (r *AccessKeysRepo) Insert(ctx context.Context, userID int64, country, outlineKeyID, accessURL string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO access_keys(user_id, country_code, outline_key_id, access_url)
		VALUES ($1,$2,$3,$4)
		RETURNING id
	`, userID, country, outlineKeyID, accessURL).Scan(&id)
	return id, err
}

func (r *AccessKeysRepo) Revoke(ctx context.Context, id int64, at time.Time) error {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE access_keys
		SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`, id, at)
	return err
}
