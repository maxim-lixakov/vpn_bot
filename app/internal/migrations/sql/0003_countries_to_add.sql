CREATE TABLE IF NOT EXISTS countries_to_add (
                                                id BIGSERIAL PRIMARY KEY,
                                                user_id BIGINT NOT NULL REFERENCES users(id),
                                                request_text TEXT NOT NULL,
                                                created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_countries_to_add_user_created
    ON countries_to_add(user_id, created_at DESC);
