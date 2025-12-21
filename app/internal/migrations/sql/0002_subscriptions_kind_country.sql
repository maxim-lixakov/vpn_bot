ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'vpn',
    ADD COLUMN IF NOT EXISTS country_code TEXT;

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_kind_country_active_until
    ON subscriptions(user_id, kind, country_code, active_until DESC);
