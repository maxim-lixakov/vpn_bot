package repo

import (
	"context"
	"database/sql"
	"time"
)

type User struct {
	ID             int64
	TgUserID       int64
	Username       sql.NullString
	FirstName      sql.NullString
	LastName       sql.NullString
	LanguageCode   sql.NullString
	Phone          sql.NullString
	CreatedAt      time.Time
	LastActivityAt time.Time
}

type UsersRepo struct{ db *sql.DB }

func NewUsersRepo(db *sql.DB) *UsersRepo { return &UsersRepo{db: db} }

func (r *UsersRepo) UpsertByTelegram(ctx context.Context, u User) (User, error) {
	// update last_activity every time
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO users (tg_user_id, username, first_name, last_name, language_code, phone, last_activity_at)
		VALUES ($1,$2,$3,$4,$5,$6, now())
		ON CONFLICT (tg_user_id) DO UPDATE SET
		  username = EXCLUDED.username,
		  first_name = EXCLUDED.first_name,
		  last_name = EXCLUDED.last_name,
		  language_code = EXCLUDED.language_code,
		  phone = COALESCE(users.phone, EXCLUDED.phone),
		  last_activity_at = now()
		RETURNING id, tg_user_id, username, first_name, last_name, language_code, phone, created_at, last_activity_at
	`,
		u.TgUserID, u.Username, u.FirstName, u.LastName, u.LanguageCode, u.Phone,
	)

	var out User
	if err := row.Scan(&out.ID, &out.TgUserID, &out.Username, &out.FirstName, &out.LastName, &out.LanguageCode, &out.Phone, &out.CreatedAt, &out.LastActivityAt); err != nil {
		return User{}, err
	}
	return out, nil
}

func (r *UsersRepo) GetByTelegramID(ctx context.Context, tgUserID int64) (User, bool, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tg_user_id, username, first_name, last_name, language_code, phone, created_at, last_activity_at
		FROM users WHERE tg_user_id=$1
	`, tgUserID)

	var out User
	err := row.Scan(&out.ID, &out.TgUserID, &out.Username, &out.FirstName, &out.LastName, &out.LanguageCode, &out.Phone, &out.CreatedAt, &out.LastActivityAt)
	if err == sql.ErrNoRows {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	return out, true, nil
}
