# ozon-simulator-go-cart

Микросервис корзины покупок — сервис в рамках учебного проекта «Симулятор Ozon».

## Стек

- **Go** — язык реализации
- **HTTP** — транспортный слой (стандартная библиотека `net/http`)
- **PostgreSQL** — хранилище корзин
- **goose** — миграции БД
- **Prometheus** — сбор метрик
- **Swagger** — документация API (swaggo)
- **gRPC** — клиент для обращения к сервису товаров
- **Kafka** — получение событий об истечении резервирований (segmentio/kafka-go)

## Архитектура

```
cmd/
  server/          — точка входа HTTP-сервера
  consumer/        — точка входа Kafka-консьюмера
internal/
  app/
    handlers/      — HTTP-обработчики (один каталог = один endpoint)
    middlewares/   — HTTP-middleware (таймер)
    round_trippers/ — round-tripper для замера времени исходящих запросов
    validation/    — общая валидация запросов
  model/           — доменные модели и ошибки
  service/
    cart_item/     — бизнес-логика корзины
  infra/
    config/        — загрузка конфигурации из YAML
    database/
      repository/  — работа с таблицами корзины и товаров в PostgreSQL
    external_services/
      products/    — gRPC-клиент сервиса товаров
    kafka/         — Kafka-консьюмер событий reservation-expired
    metrics/       — Prometheus-метрики (запросы, БД)
pkg/
  http/            — общие HTTP-утилиты (response writer, error response)
migrations/        — SQL-миграции (goose)
swagger/           — сгенерированная Swagger-документация
```

## Резервирование товаров

Резервирование происходит только в момент оформления заказа:

1. **Добавление товара** — обращается к сервису товаров (`GetBySku`) для получения данных о товаре (цена, название) и проверки наличия на складе. Удаление позиции и очистка корзины — только с локальной БД, без обращения к складу.
2. **Чекаут** — резервирует все позиции корзины (`Reserve`), очищает корзину, затем асинхронно подтверждает резервирование (`ConfirmReservation`). Если очистить корзину не удалось — резервирование сразу освобождается (`ReleaseReservation`).
3. **Подтверждение резервирования** — выполняется в фоновой горутине. TODO: заменить на outbox для гарантированной доставки.
4. **Истечение резервирования** — сервис товаров публикует событие в топик `reservation-expired-topic`. Kafka-консьюмер принимает событие и логирует его. TODO: реализовать обработку (возврат товара в корзину или уведомление пользователя).

## API

Базовый URL: `http://localhost:5010` (локально)

| Метод  | Путь                               | Описание                                          |
|--------|------------------------------------|---------------------------------------------------|
| GET    | `/user/{user_id}/cart`             | Получить содержимое корзины пользователя          |
| POST   | `/user/{user_id}/cart/{sku}`       | Добавить товар в корзину                          |
| DELETE | `/user/{user_id}/cart/{sku}`       | Удалить позицию из корзины                        |
| DELETE | `/user/{user_id}/cart`             | Очистить корзину полностью                        |
| POST   | `/user/{user_id}/cart/checkout`    | Оформить заказ (подтвердить резервирования)       |
| GET    | `/metrics`                         | Prometheus-метрики                                |
| GET    | `/swagger/`                        | Swagger UI                                        |

> `user_id` — UUID пользователя, `sku` — числовой идентификатор товара.

### Примеры

**Добавить товар в корзину:**
```
POST http://localhost:5010/user/550e8400-e29b-41d4-a716-446655440000/cart/1
Content-Type: application/json

{"count": 2}
```

**Получить корзину:**
```
GET http://localhost:5010/user/550e8400-e29b-41d4-a716-446655440000/cart
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
POST http://localhost:5010/user/550e8400-e29b-41d4-a716-446655440000/cart/checkout
```
```json
{"total_price": 200.0}
```

## Взаимодействие с сервисом товаров

Сервис корзины обращается к `ozon-simulator-go-products` по gRPC для:
- получения данных о товаре при добавлении в корзину (`GetBySku`)
- резервирования товаров на складе при чекауте (`Reserve`)
- освобождения резервирования при ошибке чекаута (`ReleaseReservation`)
- подтверждения резервирования после очистки корзины (`ConfirmReservation`)

## Конфигурация

Путь до файла конфигурации задаётся переменной окружения `CONFIG_PATH`.

```yaml
server:
  host: localhost
  port: 5010

products:
  schema: http
  host: localhost
  port: 8082
  auth-token: testToken
  timeout: 30s

database:
  user: postgres
  password: 1234
  host: localhost
  port: 5432
  name: ozon_simulator_go_cart

jobs:
  reservation-expired-consumer:
    enabled: true

kafka:
  brokers:
    - localhost:9092
  topics:
    - name: reservation-expired-topic
      consumer-group: cart-service
```

## Запуск локально

### Зависимости

- Go 1.24+
- PostgreSQL
- Kafka
- [goose](https://github.com/pressly/goose)
- Запущенный `ozon-simulator-go-products`

### Миграции

```bash
make up-migrations
```

> По умолчанию подключается к `postgresql://postgres:1234@127.0.0.1:5432/ozon_simulator_go_cart`

### HTTP-сервер

```bash
CONFIG_PATH=configs/values_local.yaml go run ./cmd/server
```

### Kafka-консьюмер

```bash
CONFIG_PATH=configs/values_local.yaml go run ./cmd/consumer
```

> Консьюмер запускается только если в конфиге `jobs.reservation-expired-consumer.enabled: true`.

## Docker

Образ содержит два бинарных файла: `server` и `consumer`. По умолчанию запускается `server`.

```bash
# Собрать образ сервиса (содержит server и consumer)
make docker-build-latest

# Собрать образ мигратора
make docker-build-migrator

# Опубликовать
make docker-push-latest
make docker-push-migrator
```

## Метрики Prometheus

| Метрика                              | Тип       | Описание                                        |
|--------------------------------------|-----------|-------------------------------------------------|
| `cart_http_requests_total`           | Counter   | Общее количество HTTP-запросов (method, code)   |
| `cart_http_request_duration_seconds` | Histogram | Время обработки HTTP-запроса                    |
| `cart_db_requests_total`             | Counter   | Запросы к БД (method, status)                   |

Доступны по адресу `GET /metrics`.

## Генерация Swagger

```bash
make generate-swagger
```

## Генерация gRPC-клиента из proto

```bash
make proto-generate
```