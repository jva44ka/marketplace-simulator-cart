# marketplace-simulator-cart

[🇷🇺 Русский](README.md) · 🇬🇧 English

Shopping cart microservice in the "Marketplace Simulator" educational project.

## Stack

- **Go** — implementation language
- **HTTP** — transport layer (`net/http`, Go ServeMux)
- **PostgreSQL** — cart storage (pgx/v5, pgxpool)
- **gRPC** — client for calling the product service
- **etcd** — dynamic configuration store (hot-reload without restarts)
- **goose** — database migrations
- **Prometheus** — metrics
- **OpenTelemetry** — distributed traces (OTLP → Tempo)
- **Swagger** — API documentation (swaggo)

## Architecture

```
cmd/
  server/            — HTTP server entry point
internal/
  app/
    handlers/        — HTTP handlers (one directory = one endpoint)
    middlewares/     — HTTP middleware (request timer)
    interceptors/    — gRPC interceptors (timer, retry)
    validation/      — incoming request validation
  model/             — domain models and errors
  usecases/          — use cases
    add_product.go          — add item to cart
    checkout.go             — place an order (saga)
    get_cart.go             — get cart contents
    remove_product.go       — remove a single item
    remove_all_products.go  — clear the cart
    transactor.go           — interfaces (Transactor, repositories, client, metrics)
  service/
    outbox/          — build outbox records for reservation confirmations
  jobs/
    reservation_confirmation_outbox — async delivery of ConfirmReservation to product
    outbox_monitor                  — collect outbox and connection pool metrics
  infra/
    config/          — YAML config loading + ConfigStore (atomic hot-reload)
    etcd/            — etcd client, config read/seed, watcher
    circuitbreaker/  — circuit breaker for the gRPC client (gobreaker, atomic swap)
    database/
      repository/    — cart_items, products, outbox repositories
    external_services/
      products/      — gRPC client for the product service
    metrics/         — Prometheus metrics
    tracing/         — OpenTelemetry initialisation
migrations/          — SQL migrations (goose)
swagger/             — generated Swagger documentation
```

## Checkout flow

1. Fetch all cart items for the user from the database
2. Call `Reserve` on the product service — reserve each item
3. Build outbox records for reservation confirmations
4. In a single transaction: delete the cart + create outbox records
5. On transaction failure — call `ReleaseReservation` to roll back reservations
6. **Outbox job** asynchronously calls `ConfirmReservation` for each record

> **At-least-once guarantee**: the outbox job may call `ConfirmReservation` again after a restart or retry. Both `ConfirmReservation` and `ReleaseReservation` on the product side are **idempotent** — a repeated call with already-processed IDs returns success without modifying stock.

## API

Base URL: `http://localhost:5002` (in docker-compose)

| Method | Path                               | Description                                    |
|--------|------------------------------------|------------------------------------------------|
| GET    | `/user/{user_id}/cart`             | Get the user's cart contents                   |
| POST   | `/user/{user_id}/cart/{sku}`       | Add an item to the cart                        |
| DELETE | `/user/{user_id}/cart/{sku}`       | Remove an item from the cart                   |
| DELETE | `/user/{user_id}/cart`             | Clear the cart entirely                        |
| POST   | `/user/{user_id}/cart/checkout`    | Place an order                                 |
| GET    | `/metrics`                         | Prometheus metrics                             |
| GET    | `/swagger/`                        | Swagger UI                                     |

> `user_id` — user UUID, `sku` — numeric product identifier.

### Examples

**Add an item:**
```
POST http://localhost:5002/user/550e8400-e29b-41d4-a716-446655440000/cart/1
Content-Type: application/json

{"count": 2}
```

**Get cart:**
```
GET http://localhost:5002/user/550e8400-e29b-41d4-a716-446655440000/cart
```
```json
{
  "cart_items": [
    {"id": 1, "sku": 1, "name": "Face cream", "price": 100.0, "count": 2}
  ],
  "total_price": 200.0
}
```

**Place an order:**
```
POST http://localhost:5002/user/550e8400-e29b-41d4-a716-446655440000/cart/checkout
```
```json
{"total_price": 200.0}
```

## Configuration

The config file path is set via the `CONFIG_PATH` environment variable.

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
    half-open-requests: 3   # requests allowed in half-open state
    interval: 10s           # window for resetting counters in closed state
    timeout: 5s             # time in open state before transitioning to half-open
    threshold: 0.6          # error ratio to trip the circuit (0.0–1.0)
    min-requests-to-trip: 10 # minimum requests before checking threshold
  retry:
    enabled: true
    max-attempts: 3         # including the first attempt
    initial-backoff: 100ms
    max-backoff: 1s
    multiplier: 2.0         # exponential backoff
    jitter-factor: 0.2      # random deviation ±20%

database:
  user: cart
  password: cart
  host: cart-db
  port: 5432
  name: marketplace-simulator-cart

tracing:
  enabled: true
  otlp-endpoint: tempo:4317

etcd:
  endpoints:
    - etcd:2379
  dial-timeout: 5s
  config-key: /config/cart   # etcd key where the config is stored

jobs:
  reservation-confirmation-outbox:
    enabled: true
    idle-interval: 10ms   # pause when queue is empty
    active-interval: 0s   # pause when previous tick had records (0 = immediately)
    batch-size: 100
    max-retries: 5
  reservation-confirmation-outbox-monitor:
    enabled: true
    job-interval: 10s
```

## Dynamic configuration (etcd)

On startup the service reads config from YAML, then connects to etcd:
- if the key exists — loads config from etcd on top of YAML defaults;
- if the key is absent — writes the YAML config to etcd (first start).

It then starts a `Watch` on the key — any change in etcd is applied in real time.

If etcd is unavailable on startup — the service continues with YAML config (graceful degradation).

| Parameter | Updated without restart | Mechanism |
|---|---|---|
| `products.circuit-breaker.*` | ✅ | atomic replacement of `gobreaker.CircuitBreaker` in callback |
| `products.retry.*` | ✅ | retry interceptor reads `cfgStore.Load()` on every call |
| `products.timeout` | ✅ | read from `cfgStore.Load()` on every call |
| `jobs.*.enabled` / `job-interval` / `batch-size` / `max-retries` | ✅ | jobs read `cfgStore.Load()` on every tick |
| `database.*` | ⚠️ requires restart | warning log on change |
| `server.*` | ⚠️ requires restart | warning log on change |
| `tracing.*` | ⚠️ requires restart | warning log on change |

### Change config via etcdctl

```bash
# View current config
docker exec etcd etcdctl get /config/cart

# Tighten circuit breaker
docker exec etcd etcdctl put /config/cart "$(
  docker exec etcd etcdctl get /config/cart --print-value-only \
  | sed 's/threshold: 0.6/threshold: 0.3/'
)"
```

Or via **etcd UI** → [http://localhost:8091](http://localhost:8091).

## Prometheus metrics

| Metric | Type | Description |
|--------|------|-------------|
| `requests_total{service,method,code}` | Counter | HTTP requests by route pattern and status code |
| `request_duration_seconds{service,method}` | Histogram | HTTP request processing time |
| `db_requests_total{service,method,status}` | Counter | Database requests |
| `db_request_duration_seconds{service,method}` | Histogram | Database request duration |
| `db_pool_acquired_conns{service}` | Gauge | Acquired pool connections |
| `db_pool_idle_conns{service}` | Gauge | Idle pool connections |
| `db_pool_total_conns{service}` | Gauge | Total pool connections |
| `db_pool_max_conns{service}` | Gauge | Maximum connections (MaxConns) |
| `db_pool_avg_acquire_duration_seconds{service}` | Gauge | Average connection wait time |
| `outbox_records_pending{service}` | Gauge | Outbox records in queue |
| `outbox_records_dead_letter{service}` | Gauge | Outbox records in dead letter |
| `outbox_records_processed_total{service,status}` | Counter | Processed outbox records |
| `active_carts{service}` | Gauge | Users with non-empty carts |
| `cart_items_total{service}` | Gauge | Total number of items across all carts |
| `checkouts_total{service,status,reason}` | Counter | Checkout attempts (success / failed with reason) |
| `checkout_value_total{service}` | Counter | Total revenue from successful orders |

## Running locally

### Dependencies

- Go 1.24+
- PostgreSQL
- Running `marketplace-simulator-product`
- [goose](https://github.com/pressly/goose)

### Migrations

```bash
make up-migrations
```

### Server

```bash
CONFIG_PATH=configs/values_local.yaml go run ./cmd/server
```

## Docker

```bash
make docker-build-latest   # service image
make docker-build-migrator # migrator image
make docker-push-latest
make docker-push-migrator
```

## Code generation

```bash
make generate-swagger   # Swagger documentation
make proto-generate     # gRPC client from proto
```
