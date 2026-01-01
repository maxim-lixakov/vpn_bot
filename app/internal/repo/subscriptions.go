package repo

import (
	"context"
	"database/sql"
	"fmt"
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

type ExpiredSubscriptionWithKey struct {
	SubscriptionID int64
	UserID         int64
	CountryCode    sql.NullString
	AccessKeyID    int64
	OutlineKeyID   string
	ActiveUntil    time.Time
}

type SubscriptionsRepoInterface interface {
	GetActiveUntilFor(ctx context.Context, userID int64, kind string, country sql.NullString, now time.Time) (time.Time, bool, error)
	HasAnyActiveSubscription(ctx context.Context, userID int64, kind string, now time.Time) (bool, error)
	HasEverHadSubscription(ctx context.Context, userID int64, kind string) (bool, error)
	ListByUser(ctx context.Context, userID int64) ([]Subscription, error)
	MarkPaid(ctx context.Context, args MarkPaidArgs) (activeUntil time.Time, err error)
	AttachAccessKeyToLatestPaid(ctx context.Context, userID int64, kind string, country sql.NullString, accessKeyID int64) error
	GetLatestPaidByKind(ctx context.Context, userID int64, kind string) (int64, bool, error)
	UpdateCountryCodeForPromocode(ctx context.Context, userID int64, country string) error
	DeletePromocodeSubscription(ctx context.Context, userID int64) error
	ExtendActiveSubscriptionByMonth(ctx context.Context, userID int64, kind string) (oldUntil, newUntil time.Time, err error)
	GetExpiredSubscriptionsWithActiveKeys(ctx context.Context, now time.Time) ([]ExpiredSubscriptionWithKey, error)
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

// HasAnyActiveSubscription проверяет, есть ли у пользователя хотя бы одна активная подписка указанного вида (любая страна)
func (r *SubscriptionsRepo) HasAnyActiveSubscription(ctx context.Context, userID int64, kind string, now time.Time) (bool, error) {
	kind = strings.TrimSpace(strings.ToLower(kind))

	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM subscriptions
		WHERE user_id=$1 AND status='paid' AND kind=$2 AND active_until > $3
	`, userID, kind, now).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasEverHadSubscription проверяет, была ли у пользователя когда-либо подписка указанного вида (независимо от статуса и даты)
func (r *SubscriptionsRepo) HasEverHadSubscription(ctx context.Context, userID int64, kind string) (bool, error) {
	kind = strings.TrimSpace(strings.ToLower(kind))

	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM subscriptions
		WHERE user_id=$1 AND status='paid' AND kind=$2
	`, userID, kind).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
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
	Months                  int // количество месяцев (0 = использовать дефолт по kind)
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
		months := args.Months
		if months <= 0 {
			months = 1 // дефолт для VPN
		}
		activeUntil = base.AddDate(0, months, 0)
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

func (r *SubscriptionsRepo) GetLatestPaidByKind(ctx context.Context, userID int64, kind string) (int64, bool, error) {
	kind = strings.TrimSpace(strings.ToLower(kind))

	var subID int64
	err := r.db.QueryRowContext(ctx, `
		SELECT id
		FROM subscriptions
		WHERE user_id=$1 AND status='paid' AND kind=$2
		ORDER BY paid_at DESC, id DESC
		LIMIT 1
	`, userID, kind).Scan(&subID)

	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return subID, true, nil
}

// UpdateCountryCodeForPromocode обновляет country_code в последней подписке от промокода (без country_code)
func (r *SubscriptionsRepo) UpdateCountryCodeForPromocode(ctx context.Context, userID int64, country string) error {
	country = strings.TrimSpace(strings.ToLower(country))

	_, err := r.db.ExecContext(ctx, `
		UPDATE subscriptions
		SET country_code = $2
		WHERE id = (
			SELECT id
			FROM subscriptions
			WHERE user_id=$1 
			  AND status='paid' 
			  AND kind='vpn'
			  AND country_code IS NULL
			  AND provider_payment_charge_id = 'promocode'
			ORDER BY paid_at DESC, id DESC
			LIMIT 1
		)
	`, userID, country)
	return err
}

// DeletePromocodeSubscription удаляет последнюю подписку, созданную промокодом
// Ищет подписку с provider_payment_charge_id = 'promocode', даже если country_code уже установлен
func (r *SubscriptionsRepo) DeletePromocodeSubscription(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM subscriptions
		WHERE id = (
			SELECT id
			FROM subscriptions
			WHERE user_id=$1 
			  AND status='paid' 
			  AND kind='vpn'
			  AND provider_payment_charge_id = 'promocode'
			ORDER BY paid_at DESC, id DESC
			LIMIT 1
		)
	`, userID)
	return err
}

// ExtendActiveSubscriptionByMonth продлевает самую активную подписку пользователя на +1 месяц
// Возвращает старое и новое значение active_until
func (r *SubscriptionsRepo) ExtendActiveSubscriptionByMonth(ctx context.Context, userID int64, kind string) (oldUntil, newUntil time.Time, err error) {
	kind = strings.TrimSpace(strings.ToLower(kind))
	now := time.Now().UTC()

	// Сначала получаем старое значение active_until
	var subID int64
	err = r.db.QueryRowContext(ctx, `
		SELECT id, active_until
		FROM subscriptions
		WHERE user_id=$1 
		  AND status='paid' 
		  AND kind=$2
		  AND active_until > $3
		ORDER BY active_until DESC
		LIMIT 1
	`, userID, kind, now).Scan(&subID, &oldUntil)
	if err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, time.Time{}, fmt.Errorf("no active subscription found")
		}
		return time.Time{}, time.Time{}, err
	}

	// Обновляем подписку
	_, err = r.db.ExecContext(ctx, `
		UPDATE subscriptions
		SET active_until = active_until + INTERVAL '1 month'
		WHERE id = $1
	`, subID)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	newUntil = oldUntil.AddDate(0, 1, 0)
	return oldUntil, newUntil, nil
}

// GetExpiredSubscriptionsWithActiveKeys возвращает список истекших подписок с активными (не отозванными) ключами
func (r *SubscriptionsRepo) GetExpiredSubscriptionsWithActiveKeys(ctx context.Context, now time.Time) ([]ExpiredSubscriptionWithKey, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			s.id,
			s.user_id,
			s.country_code,
			s.access_key_id,
			ak.outline_key_id,
			s.active_until
		FROM subscriptions s
		INNER JOIN access_keys ak ON s.access_key_id = ak.id
		WHERE s.status = 'paid'
		  AND s.kind = 'vpn'
		  AND s.active_until < $1
		  AND s.access_key_id IS NOT NULL
		  AND ak.revoked_at IS NULL
		ORDER BY s.active_until ASC
	`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ExpiredSubscriptionWithKey
	for rows.Next() {
		var item ExpiredSubscriptionWithKey
		var countryCode sql.NullString
		err := rows.Scan(
			&item.SubscriptionID,
			&item.UserID,
			&countryCode,
			&item.AccessKeyID,
			&item.OutlineKeyID,
			&item.ActiveUntil,
		)
		if err != nil {
			return nil, err
		}
		item.CountryCode = countryCode
		result = append(result, item)
	}
	return result, rows.Err()
}

func nullStringToAny(ns sql.NullString) any {
	if !ns.Valid {
		return nil
	}
	return ns.String
}
