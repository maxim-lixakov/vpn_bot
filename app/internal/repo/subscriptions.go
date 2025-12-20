package repo

import (
	"context"
	"database/sql"
	"time"
)

type SubscriptionsRepo struct{ db *sql.DB }

func NewSubscriptionsRepo(db *sql.DB) *SubscriptionsRepo { return &SubscriptionsRepo{db: db} }

func (r *SubscriptionsRepo) GetActiveUntil(ctx context.Context, userID int64, now time.Time) (time.Time, bool, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT active_until
		FROM subscriptions
		WHERE user_id=$1 AND status='paid'
		ORDER BY active_until DESC
		LIMIT 1
	`, userID)

	var until time.Time
	err := row.Scan(&until)
	if err == sql.ErrNoRows {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}

	return until, until.After(now), nil
}

type MarkPaidArgs struct {
	UserID                  int64
	Provider                string
	AmountMinor             int64
	Currency                string
	TelegramPaymentChargeID sql.NullString
	ProviderPaymentChargeID sql.NullString
	PaidAt                  time.Time
}

func (r *SubscriptionsRepo) MarkPaid(ctx context.Context, args MarkPaidArgs) (activeUntil time.Time, err error) {
	now := args.PaidAt

	latestUntil, _, err := r.GetActiveUntil(ctx, args.UserID, now)
	if err != nil {
		return time.Time{}, err
	}

	base := now
	if latestUntil.After(now) {
		base = latestUntil
	}

	activeUntil = base.AddDate(0, 1, 0)

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO subscriptions(
			user_id, status, provider, amount_minor, currency, paid_at, active_until,
			telegram_payment_charge_id, provider_payment_charge_id
		)
		VALUES ($1,'paid',$2,$3,$4,$5,$6,$7,$8)
	`, args.UserID, args.Provider, args.AmountMinor, args.Currency, args.PaidAt, activeUntil, args.TelegramPaymentChargeID, args.ProviderPaymentChargeID)

	return activeUntil, err
}
