# trip-service

HTTP API для создания, получения и завершения поездок. Данные хранятся в PostgreSQL.
Контракт — `contracts/openapi/trip-service.openapi.yaml`.

## Запуск

```bash
cp environment.toml .   # если ещё нет в корне
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect

go mod download
make migrate
make run
```

Проверка:

```bash
curl -sS localhost:8080/health
curl -sS -X POST localhost:8080/api/v1/trips -H 'content-type: application/json' -d @trip.json
```

## Makefile

| Цель | Что делает |
|---|---|
| `make run` | запуск сервиса |
| `make migrate` | накатить миграции |
| `make test` | unit-тесты с `-race` и покрытием |
| `make test-integration` | интеграционные тесты репозитория (нужен `DATABASE_URL`) |
| `make lint` | golangci-lint |
| `make generate` | OpenAPI + mocks |

## Переменные окружения

См. `.env.example`. Значения подставляет `tripgoctl connect`.

| Переменная | Назначение |
|---|---|
| `HTTP_ADDR` | адрес HTTP-сервера |
| `LOG_LEVEL` | уровень логов |
| `SHUTDOWN_TIMEOUT` | бюджет graceful shutdown |
| `DATABASE_URL` | DSN PostgreSQL |
| `DATABASE_MAX_CONNS` | максимум соединений |
| `DATABASE_MIN_CONNS` | минимум соединений |
| `DATABASE_MAX_CONN_LIFETIME` | время жизни соединения |
| `DATABASE_CONNECT_TIMEOUT` | таймаут подключения |
| `DATABASE_QUERY_TIMEOUT` | таймаут операций с БД |
| `OTEL_SERVICE_NAME` | имя сервиса в телеметрии |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | адрес OTLP-коллектора |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | протокол OTLP |
| `OTEL_TRACES_SAMPLER` | сэмплер трейсов |
| `OTEL_RESOURCE_ATTRIBUTES` | доп. атрибуты ресурса |

## Архитектура

Слои:

```
cmd/service          → читает конфиг, зовёт container.New, ждёт сигнал
internal/container   → единственное место, где создаются pgxpool, репозиторий, usecase, HTTP
internal/http        → транспорт: хендлеры, middleware, маппинг ошибок
internal/usecase     → сценарии (create/get/finish)
internal/domain/trip → правила поездки и доменные ошибки
internal/repository  → PostgreSQL
internal/infra       → пул БД, транзакции, OpenTelemetry + slog
```

Зависимости направлены внутрь: HTTP и repository знают про usecase/domain, usecase знает про domain, domain никого не импортирует.

Интерфейсы объявлены в `contract.go` пакета-потребителя (usecase и http), а не рядом с реализацией. Так usecase описывает, что ему нужно, и его можно вызвать из другого транспорта без переписывания.

Отдельный пакет `model` не заводили: между слоями ходят типы из `domain/trip`. Это те же структуры данных, без дублирования.

### Ошибки

Доменные ошибки — в `internal/domain/trip`. Перевод в HTTP `Problem` — только в `internal/http/domainerror`. Хендлеры не разбирают `errors.Is` сами.

### Observability

Порядок middleware: Recover → Tracing → Logging → Metrics → роутер.

- логи: `slog` JSON в stdout, в записи есть `trace_id` и `span_id`
- трейсы: HTTP через otelhttp, БД через otelpgx
- метрики: `http.server.request.duration`, `http.server.active_requests`, `db.client.operation.duration`, `tripgo.trips.created`, `tripgo.trips.completed`

## Решения (из ЛР1, сохранены)

### Уровень изоляции транзакций

Read Committed. Запрет двух активных поездок держит partial unique index, завершение — `UPDATE ... WHERE status = 'active'`.

### Менеджер транзакций

`Do` открывает транзакцию и кладёт её в context. Репозиторий берёт executor из контекста. Создание поездки — один `Do` на `trips` + `trip_status_history`.

### Запрет двух активных поездок

Unique index `ON trips (driver_id) WHERE status = 'active'`. Конфликт → `23505` → `ErrDriverBusy` → `409`.

## Ответы на вопросы защиты

### 1. Где объявлены интерфейсы зависимостей и почему там; что изменится во 2-м транспорте (ЛР4)

Интерфейсы лежат в `contract.go` у **потребителя**, а не у реализации:

- usecase: `internal/usecase/trip/create/contract.go` — `tripRepository`, `txManager`, `logger`, `metrics`
- то же для `get` и `finish`
- HTTP: `internal/http/trip/*/contract.go` — интерфейс `usecase`, который хендлер вызывает

Почему так: usecase сам говорит, что ему нужно («сохрани поездку», «открой транзакцию»). Репозиторий только реализует эти методы. Реализаций может быть несколько, а usecase от них не зависит.

Когда в ЛР4 появится gRPC (второй транспорт):

- **не трогаем** domain и usecase — сценарии те же;
- **добавляем** пакет вроде `internal/grpc/...` с хендлерами, которые зовут те же usecase;
- **добавляем** маппинг доменных ошибок в gRPC-коды (рядом с HTTP `domainerror`);
- **меняем** `container`: создаём gRPC-сервер и передаём ему уже собранные usecase;
- HTTP остаётся как есть.

### 2. Тест и какая регрессия; почему unit, а не интеграционный

Пример: `TestExecute_DriverBusy` в `internal/usecase/trip/create/usecase_test.go`.

Что проверяет: при `Save` → `ErrDriverBusy` usecase возвращает эту ошибку наверх и **не** создаёт поездку (результат `nil`).

Какую регрессию ловит: если кто-то проглотит ошибку репозитория, подменит её или продолжит выполнение после `driver_busy`, тест упадёт. Без него можно случайно вернуть `500` вместо доменной ошибки или отдать «успешную» поездку при занятом водителе.

Почему unit: репозиторий и `TxManager` — моки (`gomock`), БД не нужна. Проверяем только логику usecase. Интеграционный тест был бы для SQL и уникального индекса в PostgreSQL (`repo_integration_test.go`).

### 3. Трейс в Jaeger, спаны и `trace_id` в логе

Как смотреть: `tripgoctl connect` → URL Jaeger → сервис `trip-service` → любой `POST /api/v1/trips`.

Что в трейсе обычно видно:

- корневой спан HTTP: `POST /api/v1/trips` (имя = method + шаблон маршрута);
- внутри — спаны БД от `otelpgx`: `pool.acquire`, `BEGIN`, `INSERT`, `COMMIT` (или `SELECT` на get/finish).

Как `trace_id` попадает в лог:

1. tracing middleware (`otelhttp`) создаёт корневой спан и кладёт его в `context` запроса;
2. дальше этот `context` идёт в хендлер → usecase → репозиторий;
3. наш `slog`-handler (`internal/infra/telemetry`) при каждой записи читает спан из `context` через `trace.SpanFromContext` и пишет поля `trace_id` и `span_id`.

Поэтому в логе `request handled` / `trip created` тот же `trace_id`, что у трейса в Jaeger — по нему их и связывают.
