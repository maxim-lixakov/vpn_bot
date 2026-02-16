package repo

import (
	"context"
	"database/sql"
	"time"
)

type PromocodeUsage struct {
	ID          int64
	PromocodeID int64
	UsedBy      int64
	UsedAt      time.Time
}

type PromocodeUsagesRepo struct{ db *sql.DB }

type ReferralUsageDetail struct {
	PromocodeName    string
	ReceiverTgUserID int64
	ReceiverUsername sql.NullString
	ReferrerTgUserID int64
	ReferrerUsername sql.NullString
	UsedAt           time.Time
}

type PromocodeUsagesRepoInterface interface {
	Insert(ctx context.Context, promocodeID, userID int64) error
	HasUserUsed(ctx context.Context, promocodeID, userID int64) (bool, error)
	Delete(ctx context.Context, promocodeID, userID int64) error
	GetLastUsedPromocodeID(ctx context.Context, userID int64) (int64, bool, error)
	GetReferralUsagesInPeriod(ctx context.Context, from, to time.Time) ([]ReferralUsageDetail, error)
}

func NewPromocodeUsagesRepo(db *sql.DB) PromocodeUsagesRepoInterface {
	return &PromocodeUsagesRepo{db: db}
}

func (r *PromocodeUsagesRepo) Insert(ctx context.Context, promocodeID, userID int64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO promocode_usages(promocode_id, used_by, used_at)
		VALUES ($1, $2, now())
	`, promocodeID, userID)
	return err
}

func (r *PromocodeUsagesRepo) HasUserUsed(ctx context.Context, promocodeID, userID int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM promocode_usages
		WHERE promocode_id = $1 AND used_by = $2
	`, promocodeID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PromocodeUsagesRepo) Delete(ctx context.Context, promocodeID, userID int64) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM promocode_usages
		WHERE promocode_id = $1 AND used_by = $2
	`, promocodeID, userID)
	return err
}

func (r *PromocodeUsagesRepo) GetReferralUsagesInPeriod(ctx context.Context, from, to time.Time) ([]ReferralUsageDetail, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			p.promocode_name,
			receiver.tg_user_id,
			receiver.username,
			referrer.tg_user_id,
			referrer.username,
			pu.used_at
		FROM promocode_usages pu
		JOIN promocodes p ON pu.promocode_id = p.id
		JOIN users receiver ON pu.used_by = receiver.id
		JOIN users referrer ON p.promoted_by = referrer.id
		WHERE pu.used_at >= $1 AND pu.used_at < $2
		  AND p.promoted_by IS NOT NULL
		ORDER BY pu.used_at DESC
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ReferralUsageDetail
	for rows.Next() {
		var d ReferralUsageDetail
		if err := rows.Scan(
			&d.PromocodeName,
			&d.ReceiverTgUserID, &d.ReceiverUsername,
			&d.ReferrerTgUserID, &d.ReferrerUsername,
			&d.UsedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *PromocodeUsagesRepo) GetLastUsedPromocodeID(ctx context.Context, userID int64) (int64, bool, error) {
	var promocodeID int64
	err := r.db.QueryRowContext(ctx, `
		SELECT promocode_id
		FROM promocode_usages
		WHERE used_by = $1
		ORDER BY used_at DESC
		LIMIT 1
	`, userID).Scan(&promocodeID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return promocodeID, true, nil
}
