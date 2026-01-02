package repo

import (
	"context"
	"database/sql"
	"time"
)

type Payment struct {
	ID                      int64
	SubscriptionID          int64
	UserID                  int64
	Provider                string
	AmountMinor             int64
	Currency                string
	PaidAt                  time.Time
	TelegramPaymentChargeID sql.NullString
	ProviderPaymentChargeID sql.NullString
	Months                  int
	CreatedAt               time.Time
}

type PaymentsRepo struct{ db *sql.DB }

type PaymentsRepoInterface interface {
	Insert(ctx context.Context, args InsertPaymentArgs) (int64, error)
}

type InsertPaymentArgs struct {
	SubscriptionID          int64
	UserID                  int64
	Provider                string
	AmountMinor             int64
	Currency                string
	PaidAt                  time.Time
	TelegramPaymentChargeID sql.NullString
	ProviderPaymentChargeID sql.NullString
	Months                  int
}

func NewPaymentsRepo(db *sql.DB) PaymentsRepoInterface {
	return &PaymentsRepo{db: db}
}

func (r *PaymentsRepo) Insert(ctx context.Context, args InsertPaymentArgs) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO payments(
			subscription_id, user_id, provider, amount_minor, currency,
			paid_at, telegram_payment_charge_id, provider_payment_charge_id, months
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`,
		args.SubscriptionID,
		args.UserID,
		args.Provider,
		args.AmountMinor,
		args.Currency,
		args.PaidAt,
		args.TelegramPaymentChargeID,
		args.ProviderPaymentChargeID,
		args.Months,
	).Scan(&id)
	return id, err
}
