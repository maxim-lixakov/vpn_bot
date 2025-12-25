-- 1) add column
ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS access_key_id BIGINT;

-- 2) FK (если ключ удалят - просто отвяжем)
DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'subscriptions_access_key_id_fkey'
        ) THEN
            ALTER TABLE subscriptions
                ADD CONSTRAINT subscriptions_access_key_id_fkey
                    FOREIGN KEY (access_key_id) REFERENCES access_keys(id)
                        ON DELETE SET NULL;
        END IF;
    END $$;

-- 3) индекс под джойны/проверки
CREATE INDEX IF NOT EXISTS idx_subscriptions_access_key_active_until
    ON subscriptions(access_key_id, active_until DESC);

-- 4) backfill для продовых данных:
-- пытаемся привязать vpn-подписки к активному ключу по user_id + country_code
UPDATE subscriptions s
SET access_key_id = ak.id
FROM access_keys ak
WHERE s.access_key_id IS NULL
  AND s.kind = 'vpn'
  AND s.user_id = ak.user_id
  AND lower(trim(s.country_code)) = lower(trim(ak.country_code))
  AND ak.revoked_at IS NULL;

-- 5) (опционально) если активного ключа нет, но есть любой ключ в истории - привяжем к последнему
UPDATE subscriptions s
SET access_key_id = (
    SELECT ak.id
    FROM access_keys ak
    WHERE ak.user_id = s.user_id
      AND lower(trim(ak.country_code)) = lower(trim(s.country_code))
    ORDER BY ak.created_at DESC
    LIMIT 1
)
WHERE s.access_key_id IS NULL
  AND s.kind = 'vpn'
  AND s.country_code IS NOT NULL
  AND EXISTS (
      SELECT 1
      FROM access_keys ak
      WHERE ak.user_id = s.user_id
        AND lower(trim(ak.country_code)) = lower(trim(s.country_code))
  );
