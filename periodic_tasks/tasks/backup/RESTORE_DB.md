# Восстановление базы данных из дампа

## Способ 1: Через docker compose (рекомендуется)

Если контейнер postgres запущен через docker compose:

```bash
# Убедитесь, что контейнер запущен
docker compose up -d postgres

# Восстановите базу данных
docker compose exec -T postgres psql -U vpn -d vpn < vpn_backup.sql
```

Или с переменной окружения для пароля:

```bash
PGPASSWORD=vpn docker compose exec -T postgres psql -U vpn -d vpn < vpn_backup.sql
```

