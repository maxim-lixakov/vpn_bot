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

type UsersRepoInterface interface {
	UpsertByTelegram(ctx context.Context, u User) (User, error)
	GetByTelegramID(ctx context.Context, tgUserID int64) (User, bool, error)
	GetByID(ctx context.Context, userID int64) (User, bool, error)
	GetAllUsers(ctx context.Context) ([]User, error)
	GetUsersWithActiveSubscriptions(ctx context.Context, now time.Time) ([]User, error)
	GetUsersWithoutSubscriptions(ctx context.Context) ([]User, error)
	CountAll(ctx context.Context) (int, error)
}

func NewUsersRepo(db *sql.DB) UsersRepoInterface { return &UsersRepo{db: db} }

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

func (r *UsersRepo) GetByID(ctx context.Context, userID int64) (User, bool, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tg_user_id, username, first_name, last_name, language_code, phone, created_at, last_activity_at
		FROM users WHERE id=$1
	`, userID)

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

// GetAllUsers возвращает всех пользователей
func (r *UsersRepo) GetAllUsers(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tg_user_id, username, first_name, last_name, language_code, phone, created_at, last_activity_at
		FROM users
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.TgUserID, &u.Username, &u.FirstName, &u.LastName, &u.LanguageCode, &u.Phone, &u.CreatedAt, &u.LastActivityAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// GetUsersWithActiveSubscriptions возвращает пользователей с хотя бы одной активной подпиской
func (r *UsersRepo) GetUsersWithActiveSubscriptions(ctx context.Context, now time.Time) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT u.id, u.tg_user_id, u.username, u.first_name, u.last_name, u.language_code, u.phone, u.created_at, u.last_activity_at
		FROM users u
		INNER JOIN subscriptions s ON u.id = s.user_id
		WHERE s.status = 'paid' AND s.kind = 'vpn' AND s.active_until > $1
		ORDER BY u.id
	`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.TgUserID, &u.Username, &u.FirstName, &u.LastName, &u.LanguageCode, &u.Phone, &u.CreatedAt, &u.LastActivityAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// GetUsersWithoutSubscriptions возвращает пользователей без подписок
func (r *UsersRepo) GetUsersWithoutSubscriptions(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.tg_user_id, u.username, u.first_name, u.last_name, u.language_code, u.phone, u.created_at, u.last_activity_at
		FROM users u
		LEFT JOIN subscriptions s ON u.id = s.user_id AND s.status = 'paid' AND s.kind = 'vpn'
		WHERE s.id IS NULL
		ORDER BY u.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.TgUserID, &u.Username, &u.FirstName, &u.LastName, &u.LanguageCode, &u.Phone, &u.CreatedAt, &u.LastActivityAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CountAll возвращает общее количество пользователей
func (r *UsersRepo) CountAll(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}
