-- Добавляем флаг allow_for_old_users в таблицу promocodes
-- По умолчанию false - промокод только для новых пользователей
ALTER TABLE promocodes
    ADD COLUMN IF NOT EXISTS allow_for_old_users BOOLEAN NOT NULL DEFAULT false;

-- Создаём индекс для быстрого поиска (если нужно)
CREATE INDEX IF NOT EXISTS idx_promocodes_allow_for_old_users
    ON promocodes(allow_for_old_users);

