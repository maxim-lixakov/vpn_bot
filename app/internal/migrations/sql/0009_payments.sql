-- Payments table for tracking all payments
CREATE TABLE IF NOT EXISTS payments (
    id BIGSERIAL PRIMARY KEY,
    subscription_id BIGINT NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL, -- telegram
    amount_minor BIGINT NOT NULL,
    currency TEXT NOT NULL,
    paid_at TIMESTAMPTZ NOT NULL,
    telegram_payment_charge_id TEXT,
    provider_payment_charge_id TEXT,
    months INT NOT NULL DEFAULT 1, -- количество месяцев подписки
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS payments_subscription_id_idx ON payments(subscription_id);
CREATE INDEX IF NOT EXISTS payments_user_id_idx ON payments(user_id);
CREATE INDEX IF NOT EXISTS payments_paid_at_idx ON payments(paid_at);

