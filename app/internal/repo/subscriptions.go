package repo

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type Subscription struct {
	ID                      int64
	UserID                  int64
	Kind                    string
	CountryCode             sql.NullString
	AccessKeyID             sql.NullInt64 // <-- NEW
	Status                  string
	Provider                string
	AmountMinor             int64
	Currency                string
	PaidAt                  time.Time
	ActiveUntil             time.Time
	TelegramPaymentChargeID sql.NullString
	ProviderPaymentChargeID sql.NullString
	CreatedAt               time.Time
}

type SubscriptionsRepo struct{ db *sql.DB }

type SubscriptionsRepoInterface interface {
	GetActiveUntilFor(ctx context.Context, userID int64, kind string, country sql.NullString, now time.Time) (time.Time, bool, error)
	ListByUser(ctx context.Context, userID int64) ([]Subscription, error)
	MarkPaid(ctx context.Context, args MarkPaidArgs) (activeUntil time.Time, err error)
	AttachAccessKeyToLatestPaid(ctx context.Context, userID int64, kind string, country sql.NullString, accessKeyID int64) error
}

func NewSubscriptionsRepo(db *sql.DB) SubscriptionsRepoInterface { return &SubscriptionsRepo{db: db} }

func (r *SubscriptionsRepo) GetActiveUntilFor(ctx context.Context, userID int64, kind string, country sql.NullString, now time.Time) (time.Time, bool, error) {
	q := `
		SELECT active_until
		FROM subscriptions
		WHERE user_id=$1 AND status='paid' AND kind=$2
		  AND (
		        ($3::text IS NULL AND country_code IS NULL) OR
		        (country_code = $3::text)
		      )
		ORDER BY active_until DESC
		LIMIT 1
	`

	var until time.Time
	var argCountry any = nil
	if country.Valid {
		argCountry = strings.TrimSpace(strings.ToLower(country.String))
	}

	kind = strings.TrimSpace(strings.ToLower(kind))

	err := r.db.QueryRowContext(ctx, q, userID, kind, argCountry).Scan(&until)
	if err == sql.ErrNoRows {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	return until, until.After(now), nil
}

func (r *SubscriptionsRepo) ListByUser(ctx context.Context, userID int64) ([]Subscription, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, user_id, kind, country_code, access_key_id,
			status, provider, amount_minor, currency, paid_at, active_until,
			telegram_payment_charge_id, provider_payment_charge_id, created_at
		FROM subscriptions
		WHERE user_id=$1
		ORDER BY paid_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Kind, &s.CountryCode, &s.AccessKeyID,
			&s.Status, &s.Provider,
			&s.AmountMinor, &s.Currency, &s.PaidAt, &s.ActiveUntil,
			&s.TelegramPaymentChargeID, &s.ProviderPaymentChargeID, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

type MarkPaidArgs struct {
	UserID                  int64
	Kind                    string
	CountryCode             sql.NullString
	AccessKeyID             sql.NullInt64 // <-- NEW (можно передать если ключ уже есть)
	Provider                string
	AmountMinor             int64
	Currency                string
	TelegramPaymentChargeID sql.NullString
	ProviderPaymentChargeID sql.NullString
	PaidAt                  time.Time
}

func (r *SubscriptionsRepo) MarkPaid(ctx context.Context, args MarkPaidArgs) (time.Time, error) {
	now := args.PaidAt
	if now.IsZero() {
		now = time.Now().UTC()
	}

	args.Kind = strings.TrimSpace(strings.ToLower(args.Kind))
	if args.CountryCode.Valid {
		args.CountryCode.String = strings.TrimSpace(strings.ToLower(args.CountryCode.String))
	}

	latestUntil, _, err := r.GetActiveUntilFor(ctx, args.UserID, args.Kind, args.CountryCode, now)
	if err != nil {
		return time.Time{}, err
	}

	base := now
	if latestUntil.After(now) {
		base = latestUntil
	}

	activeUntil := base
	if args.Kind == "vpn" {
		activeUntil = base.AddDate(0, 1, 0)
	} else {
		activeUntil = now
	}

	var cc any = nil
	if args.CountryCode.Valid {
		cc = args.CountryCode.String
	}

	var ak any = nil
	if args.AccessKeyID.Valid {
		ak = args.AccessKeyID.Int64
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO subscriptions(
			user_id, kind, country_code, access_key_id,
			status, provider, amount_minor, currency, paid_at, active_until,
			telegram_payment_charge_id, provider_payment_charge_id
		)
		VALUES ($1,$2,$3,$4,'paid',$5,$6,$7,$8,$9,$10,$11)
	`,
		args.UserID, args.Kind, cc, ak,
		args.Provider, args.AmountMinor, args.Currency, now, activeUntil,
		nullStringToAny(args.TelegramPaymentChargeID), nullStringToAny(args.ProviderPaymentChargeID),
	)

	return activeUntil, err
}

func (r *SubscriptionsRepo) AttachAccessKeyToLatestPaid(ctx context.Context, userID int64, kind string, country sql.NullString, accessKeyID int64) error {
	kind = strings.TrimSpace(strings.ToLower(kind))
	var cc any = nil
	if country.Valid {
		cc = strings.TrimSpace(strings.ToLower(country.String))
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE subscriptions
		SET access_key_id = $4
		WHERE id = (
			SELECT id
			FROM subscriptions
			WHERE user_id=$1 AND status='paid' AND kind=$2
			  AND (
			        ($3::text IS NULL AND country_code IS NULL) OR
			        (country_code = $3::text)
			      )
			ORDER BY paid_at DESC, id DESC
			LIMIT 1
		)
	`, userID, kind, cc, accessKeyID)
	return err
}

func nullStringToAny(ns sql.NullString) any {
	if !ns.Valid {
		return nil
	}
	return ns.String
}
