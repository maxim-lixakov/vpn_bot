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

type PromocodeUsagesRepoInterface interface {
	Insert(ctx context.Context, promocodeID, userID int64) error
	HasUserUsed(ctx context.Context, promocodeID, userID int64) (bool, error)
	Delete(ctx context.Context, promocodeID, userID int64) error
	GetLastUsedPromocodeID(ctx context.Context, userID int64) (int64, bool, error)
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
