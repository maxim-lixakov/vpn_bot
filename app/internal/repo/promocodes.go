package repo

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type Promocode struct {
	ID               int64
	PromocodeName    string
	PromotedBy       sql.NullInt64
	TimesUsed        int
	TimesToBeUsed    int
	PromocodeMonths  int
	AllowForOldUsers bool
	CreatedAt        time.Time
	LastUsedAt       sql.NullTime
}

type PromocodesRepo struct{ db *sql.DB }

type PromocodeWithUsage struct {
	PromocodeName string
	TimesUsed     int
	TimesToBeUsed int
	IsReferral    bool
}

type PromocodesRepoInterface interface {
	GetByName(ctx context.Context, name string) (Promocode, bool, error)
	IncrementUsage(ctx context.Context, promocodeID int64) error
	DecrementUsage(ctx context.Context, promocodeID int64) error
	GetOrCreateReferralCode(ctx context.Context, userID int64) (Promocode, error)
	GetAllWithUsage(ctx context.Context) ([]PromocodeWithUsage, error)
}

func NewPromocodesRepo(db *sql.DB) PromocodesRepoInterface {
	return &PromocodesRepo{db: db}
}

func (r *PromocodesRepo) GetByName(ctx context.Context, name string) (Promocode, bool, error) {
	name = strings.TrimSpace(name)

	row := r.db.QueryRowContext(ctx, `
		SELECT id, promocode_name, promoted_by, times_used, times_to_be_used, 
		       promocode_months, allow_for_old_users, created_at, last_used_at
		FROM promocodes
		WHERE LOWER(TRIM(promocode_name)) = LOWER(TRIM($1))
	`, name)

	var p Promocode
	err := row.Scan(
		&p.ID, &p.PromocodeName, &p.PromotedBy,
		&p.TimesUsed, &p.TimesToBeUsed, &p.PromocodeMonths,
		&p.AllowForOldUsers, &p.CreatedAt, &p.LastUsedAt,
	)

	if err == sql.ErrNoRows {
		return Promocode{}, false, nil
	}
	if err != nil {
		return Promocode{}, false, err
	}
	return p, true, nil
}

func (r *PromocodesRepo) IncrementUsage(ctx context.Context, promocodeID int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE promocodes
		SET times_used = times_used + 1,
		    last_used_at = now()
		WHERE id = $1
	`, promocodeID)
	return err
}

func (r *PromocodesRepo) DecrementUsage(ctx context.Context, promocodeID int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE promocodes
		SET times_used = GREATEST(0, times_used - 1)
		WHERE id = $1
	`, promocodeID)
	return err
}

func (r *PromocodesRepo) GetAllWithUsage(ctx context.Context) ([]PromocodeWithUsage, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT promocode_name, times_used, times_to_be_used, promoted_by IS NOT NULL AS is_referral
		FROM promocodes
		WHERE times_used > 0
		ORDER BY times_used DESC, promocode_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PromocodeWithUsage
	for rows.Next() {
		var p PromocodeWithUsage
		if err := rows.Scan(&p.PromocodeName, &p.TimesUsed, &p.TimesToBeUsed, &p.IsReferral); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// generateReferralCodeHash генерирует стабильный 10-символьный хеш на основе userID
func generateReferralCodeHash(userID int64) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("referral_%d", userID)))
	return hex.EncodeToString(hash[:])[:10]
}

// GetOrCreateReferralCode получает или создаёт реферальный промокод для пользователя
func (r *PromocodesRepo) GetOrCreateReferralCode(ctx context.Context, userID int64) (Promocode, error) {
	hash := generateReferralCodeHash(userID)
	codeName := fmt.Sprintf("referral_%d_%s", userID, hash)

	// Пытаемся найти существующий промокод
	promo, found, err := r.GetByName(ctx, codeName)
	if err != nil {
		return Promocode{}, err
	}
	if found {
		return promo, nil
	}

	// Создаём новый промокод
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO promocodes(
			promocode_name, promoted_by, times_used, times_to_be_used, promocode_months, allow_for_old_users, created_at
		)
		VALUES ($1, $2, 0, 50, 1, false, now())
		RETURNING id, promocode_name, promoted_by, times_used, times_to_be_used, 
		       promocode_months, allow_for_old_users, created_at, last_used_at
	`, codeName, userID)

	var p Promocode
	err = row.Scan(
		&p.ID, &p.PromocodeName, &p.PromotedBy,
		&p.TimesUsed, &p.TimesToBeUsed, &p.PromocodeMonths,
		&p.AllowForOldUsers, &p.CreatedAt, &p.LastUsedAt,
	)
	if err != nil {
		return Promocode{}, err
	}

	return p, nil
}
