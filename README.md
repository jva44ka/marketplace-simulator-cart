# marketplace-simulator-cart

Микросервис корзины покупок в рамках учебного проекта «Симулятор маркетплейса».

## Стек

- **Go** — язык реализации
- **HTTP** — транспортный слой (`net/http`, Go ServeMux)
- **PostgreSQL** — хранилище корзин (pgx/v5, pgxpool)
- **gRPC** — клиент для обращения к сервису товаров
- **goose** — миграции БД
- **Prometheus** — метрики
- **OpenTelemetry** — распределённые трейсы (OTLP → Tempo)
- **Swagger** — документация API (swaggo)

## Архитектура

```
cmd/
  server/            — точка входа HTTP-сервера
internal/
  app/
    handlers/        — HTTP-обработчики (один каталог = один endpoint)
    middlewares/     — HTTP-middleware (таймер запросов)
    interceptors/    — gRPC-интерцепторы (таймер, ретрай)
    validation/      — валидация входящих запросов
  model/             — доменные модели и ошибки
  service/
    cart_item/       — бизнес-логика корзины (добавление, удаление, чекаут)
    outbox/          — построение outbox-записей подтверждения резервирований
  jobs/
    reservation_confirmation_outbox — асинхронная доставка ConfirmReservation в product
    outbox_monitor                  — сбор метрик outbox и пула соединений
  infra/
    config/          — загрузка конфигурации из YAML
    circuitbreaker/  — circuit breaker для gRPC-клиента (gobreaker)
    database/
      repository/    — репозитории cart_items, products, outbox
    external_services/
      products/      — gRPC-клиент сервиса товаров
    metrics/         — Prometheus-метрики
    tracing/         — инициализация OpenTelemetry
migrations/          — SQL-миграции (goose)
swagger/             — сгенерированная Swagger-документация
```

## Оформление заказа (checkout)

1. Получает все позиции корзины пользователя из БД
2. Вызывает `Reserve` на product service — резервирует каждый товар
3. Строит outbox-записи для подтверждения резервирований
4. В одной транзакции: удаляет корзину + создаёт outbox-записи
5. При ошибке транзакции — вызывает `ReleaseReservation` для возврата резервирований
6. **Outbox job** асинхронно вызывает `ConfirmReservation` для каждой записи

## API

Базовый URL: `http://localhost:5002` (в docker-compose)

| Метод  | Путь                               | Описание                                          |
|--------|------------------------------------|---------------------------------------------------|
| GET    | `/user/{user_id}/cart`             | Получить содержимое корзины пользователя          |
| POST   | `/user/{user_id}/cart/{sku}`       | Добавить товар в корзину                          |
| DELETE | `/user/{user_id}/cart/{sku}`       | Удалить позицию из корзины                        |
| DELETE | `/user/{user_id}/cart`             | Очистить корзину полностью                        |
| POST   | `/user/{user_id}/cart/checkout`    | Оформить заказ                                    |
| GET    | `/metrics`                         | Prometheus-метрики                                |
| GET    | `/swagger/`                        | Swagger UI                                        |

> `user_id` — UUID пользователя, `sku` — числовой идентификатор товара.

### Примеры

**Добавить товар:**
```
POST http://localhost:5002/user/550e8400-e29b-41d4-a716-446655440000/cart/1
Content-Type: application/json

{"count": 2}
```

**Получить корзину:**
```
GET http://localhost:5002/user/550e8400-e29b-41d4-a716-446655440000/cart
```
```json
{
  "cart_items": [
    {"id": 1, "sku": 1, "name": "Крем для лица", "price": 100.0, "count": 2}
  ],
  "total_price": 200.0
}
```

**Оформить заказ:**
```
POST http://localhost:5002/user/550e8400-e29b-41d4-a716-446655440000/cart/checkout
```
```json
{"total_price": 200.0}
```

## Конфигурация

Путь до файла задаётся переменной окружения `CONFIG_PATH`.

```yaml
server:
  host:
  port: 5000

products:
  host: product
  port: 8002
  auth-token: admin
  timeout: 30s
  circuit-breaker:
    enabled: true
    half-open-requests: 3   # запросов в half-open состоянии
    interval: 10s           # окно сброса счётчиков в closed
    timeout: 5s             # время в open перед переходом в half-open
    threshold: 0.6          # доля ошибок для открытия цепи (0.0–1.0)
    min-requests-to-trip: 10 # минимум запросов перед проверкой threshold
  retry:
    enabled: true
    max-attempts: 3         # включая первую попытку
    initial-backoff: 100ms
    max-backoff: 1s
    multiplier: 2.0         # exponential backoff
    jitter-factor: 0.2      # случайное отклонение ±20%

database:
  user: cart
  password: cart
  host: cart-db
  port: 5432
  name: marketplace-simulator-cart

tracing:
  enabled: true
  otlp-endpoint: tempo:4317

jobs:
  reservation-confirmation-outbox:
    enabled: true
    idle-interval: 10ms   # пауза когда очередь пуста
    active-interval: 0s   # пауза когда в прошлом тике были записи (0 = сразу)
    batch-size: 100
    max-retries: 5
  reservation-confirmation-outbox-monitor:
    enabled: true
    job-interval: 10s
```

## Метрики Prometheus

| Метрика | Тип | Описание |
|---------|-----|----------|
| `requests_total{service,method,code}` | Counter | HTTP-запросы по route pattern и статус-коду |
| `request_duration_seconds{service,method}` | Histogram | Время обработки HTTP-запроса |
| `db_requests_total{service,method,status}` | Counter | Запросы к БД |
| `db_request_duration_seconds{service,method}` | Histogram | Длительность запросов к БД |
| `db_pool_acquired_conns{service}` | Gauge | Занятые соединения пула |
| `db_pool_idle_conns{service}` | Gauge | Свободные соединения пула |
| `db_pool_total_conns{service}` | Gauge | Всего соединений в пуле |
| `db_pool_max_conns{service}` | Gauge | Максимум соединений (MaxConns) |
| `db_pool_avg_acquire_duration_seconds{service}` | Gauge | Среднее время ожидания соединения |
| `outbox_records_pending{service}` | Gauge | Записи outbox в очереди |
| `outbox_records_dead_letter{service}` | Gauge | Записи outbox в dead letter |
| `outbox_records_processed_total{service,status}` | Counter | Обработанные outbox-записи |
| `active_carts{service}` | Gauge | Пользователи с непустой корзиной |
| `cart_items_total{service}` | Gauge | Суммарное количество позиций в корзинах |
| `checkouts_total{service,status,reason}` | Counter | Попытки чекаута (success / failed с причиной) |
| `checkout_value_total{service}` | Counter | Суммарная выручка успешных заказов |

## Запуск локально

### Зависимости

- Go 1.24+
- PostgreSQL
- Запущенный `marketplace-simulator-product`
- [goose](https://github.com/pressly/goose)

### Миграции

```bash
make up-migrations
```

### Сервер

```bash
CONFIG_PATH=configs/values_local.yaml go run ./cmd/server
```

## Docker

```bash
make docker-build-latest   # образ сервиса
make docker-build-migrator # образ мигратора
make docker-push-latest
make docker-push-migrator
```

## Генерация кода

```bash
make generate-swagger   # Swagger-документация
make proto-generate     # gRPC-клиент из proto
```
