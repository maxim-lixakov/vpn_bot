-- Users
CREATE TABLE IF NOT EXISTS users (
                                     id BIGSERIAL PRIMARY KEY,
                                     tg_user_id BIGINT NOT NULL UNIQUE,
                                     username TEXT,
                                     first_name TEXT,
                                     last_name TEXT,
                                     language_code TEXT,
                                     phone TEXT,
                                     created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                     last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- User states (one row per user)
CREATE TABLE IF NOT EXISTS user_states (
                                           user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
                                           state TEXT NOT NULL,
                                           selected_country TEXT,
                                           updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS user_states_state_idx ON user_states(state);

-- Subscriptions (history)
CREATE TABLE IF NOT EXISTS subscriptions (
                                             id BIGSERIAL PRIMARY KEY,
                                             user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                             status TEXT NOT NULL,                  -- paid/refunded/etc
                                             provider TEXT NOT NULL,                -- telegram
                                             amount_minor BIGINT NOT NULL,
                                             currency TEXT NOT NULL,
                                             paid_at TIMESTAMPTZ NOT NULL,
                                             active_until TIMESTAMPTZ NOT NULL,
                                             telegram_payment_charge_id TEXT,
                                             provider_payment_charge_id TEXT,
                                             created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS subscriptions_user_until_idx ON subscriptions(user_id, active_until);

-- Access keys (store issued keys)
CREATE TABLE IF NOT EXISTS access_keys (
                                           id BIGSERIAL PRIMARY KEY,
                                           user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                           country_code TEXT NOT NULL,
                                           outline_key_id TEXT NOT NULL,
                                           access_url TEXT NOT NULL,
                                           created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                           revoked_at TIMESTAMPTZ
);

-- only one active key per user+country
CREATE UNIQUE INDEX IF NOT EXISTS access_keys_active_uq
    ON access_keys(user_id, country_code)
    WHERE revoked_at IS NULL;
