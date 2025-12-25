-- 1) add column
ALTER TABLE countries_to_add
    ADD COLUMN IF NOT EXISTS subscription_id BIGINT;

-- 2) FK (если подписку удалят - удалим и связанные записи)
DO $$
    BEGIN
        -- Удаляем старое ограничение если оно существует с SET NULL
        IF EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'countries_to_add_subscription_id_fkey'
        ) THEN
            ALTER TABLE countries_to_add
                DROP CONSTRAINT countries_to_add_subscription_id_fkey;
        END IF;
        
        -- Создаём новое с CASCADE
        ALTER TABLE countries_to_add
            ADD CONSTRAINT countries_to_add_subscription_id_fkey
                FOREIGN KEY (subscription_id) REFERENCES subscriptions(id)
                    ON DELETE CASCADE;
    END $$;

-- 3) индекс под джойны/проверки
CREATE INDEX IF NOT EXISTS idx_countries_to_add_subscription_id
    ON countries_to_add(subscription_id);

