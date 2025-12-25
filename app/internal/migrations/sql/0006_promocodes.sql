-- Promocodes table
CREATE TABLE IF NOT EXISTS promocodes (
    id BIGSERIAL PRIMARY KEY,
    promocode_name TEXT NOT NULL UNIQUE,
    promoted_by BIGINT REFERENCES users(id) ON DELETE SET NULL, -- NULL если создан админом
    times_used INT NOT NULL DEFAULT 0,
    times_to_be_used INT NOT NULL, -- сколько раз может быть использован (0 = безлимит)
    promocode_months INT NOT NULL DEFAULT 1, -- сколько месяцев добавить к подписке
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_promocodes_name ON promocodes(promocode_name);
CREATE INDEX IF NOT EXISTS idx_promocodes_promoted_by ON promocodes(promoted_by);

-- Promocode usages (история использований)
CREATE TABLE IF NOT EXISTS promocode_usages (
    id BIGSERIAL PRIMARY KEY,
    promocode_id BIGINT NOT NULL REFERENCES promocodes(id) ON DELETE CASCADE,
    used_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    used_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_promocode_usages_promocode ON promocode_usages(promocode_id);
CREATE INDEX IF NOT EXISTS idx_promocode_usages_user ON promocode_usages(used_by);
CREATE INDEX IF NOT EXISTS idx_promocode_usages_used_at ON promocode_usages(used_at DESC);

