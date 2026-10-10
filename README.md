# trip-service

HTTP API для создания, получения и завершения поездок. Данные хранятся в PostgreSQL.
Контракт — `contracts/openapi/trip-service.openapi.yaml`.

## Запуск

```bash
# 1. Окружение курса (PostgreSQL)
cp environment.toml .   # если ещё нет в корне
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect        # сгенерирует .env с DATABASE_URL

# 2. Зависимости и миграции
go mod download
make migrate

# 3. Сервис
make run
```

Проверка:

```bash
curl -sS localhost:8080/health
curl -sS localhost:8080/ready
```

Полезные цели Makefile: `generate`, `migrate`, `migrate-down`, `run`.

## Переменные окружения

См. `.env.example`. Значения подставляет `tripgoctl connect`; в репозиторий коммитится только пример.

| Переменная | Назначение |
|---|---|
| `HTTP_ADDR` | Адрес HTTP-сервера, например `:8080` |
| `LOG_LEVEL` | Уровень логов (`info`, `debug`, …) |
| `SHUTDOWN_TIMEOUT` | Бюджет graceful shutdown, например `10s` |
| `DATABASE_URL` | DSN PostgreSQL |
| `DATABASE_MAX_CONNS` | Максимум соединений в пуле |
| `DATABASE_MIN_CONNS` | Минимум соединений в пуле |
| `DATABASE_MAX_CONN_LIFETIME` | Время жизни соединения, например `30m` |
| `DATABASE_CONNECT_TIMEOUT` | Таймаут подключения / ping на старте |
| `DATABASE_QUERY_TIMEOUT` | Таймаут операций с БД (в т.ч. транзакции) |


## Решения

### Уровень изоляции транзакций

По умолчанию — **Read Committed** (`pgx.ReadCommitted`).

Почему достаточно:

- запрет двух активных поездок у водителя держит **partial unique index**
  `ON trips (driver_id) WHERE status = 'active'`, а не блокировки на уровне
  изоляции; при гонке второй `INSERT` получает `23505` → `409 driver_busy`;
- завершение поездки атомарно в одном `UPDATE ... WHERE id = $1 AND status = 'active'`:
  второй параллельный `finish` не перезапишет `finished_at`.

Serializable / Repeatable Read здесь ничего принципиально не добавляют. Read Committed — стандарт + совпадает с тем, как защищены инварианты.

### Менеджер транзакций

Интерфейс:

```go
Do(ctx context.Context, fn func(ctx context.Context) error) error
```

- `Do` открывает транзакцию, кладёт её в `context` и вызывает `fn`;
- `nil` → `COMMIT`, ошибка или panic → `ROLLBACK`;
- вложенный `Do` переиспользует ту же транзакцию;
- репозиторий берёт исполнителя из контекста (`tx` или пул), транзакция
  **не** передаётся аргументом метода;
- бизнес-код не импортирует `pgx.Tx`.

Создание поездки выполняется в одном `Do`: `INSERT` в `trips` и
`INSERT` в `trip_status_history`. Если вторая вставка падает, откатывается и первая.

### Запрет двух активных поездок

В миграции:

```sql
CREATE UNIQUE INDEX idx_unique_active_driver
ON trips (driver_id)
WHERE status = 'active';
```

Ограничение в БД, поэтому его нельзя обойти гонкой в приложении: из N
параллельных `POST /trips` на одного водителя ровно один получит `201`,
остальные — `409` с `code = driver_busy` (маппинг PostgreSQL `23505`).

### Повторное и конкурентное завершение поездки

`POST /trips/{id}/finish` делает:

```sql
UPDATE trips
SET status = 'completed', finished_at = NOW(), ...
WHERE id = $1 AND status = 'active'
RETURNING ...
```

- первая успешная операция обновляет строку и пишет запись в
  `trip_status_history` в той же транзакции;
- повторный или параллельный `finish`: условие `status = 'active'` не
  выполняется → `UPDATE` не затрагивает строк → `ErrNoRows`;
- дальше сервис читает поездку: нет строки → `404 trip_not_found`,
  есть (уже `completed`) → `409 trip_completed`;
- `finished_at` второй раз не перезаписывается: в `SET` попадает только
  победивший `UPDATE`.

Проверка статуса в Go без условия в `UPDATE` / без `FOR UPDATE` здесь не
используется — между «прочитал» и «записал» не остаётся окна для гонки.
