# ozon-simulator-go-cart

Микросервис корзины покупок — учебный проект в рамках курса Route 256 «Продвинутая разработка микросервисов на Go».

## Стек

- **Go** — язык реализации
- **HTTP** — транспортный слой (стандартная библиотека `net/http`)
- **PostgreSQL** — хранилище корзин
- **goose** — миграции БД
- **Prometheus** — сбор метрик
- **Swagger** — документация API (swaggo)
- **gRPC** — клиент для обращения к сервису товаров

## Архитектура

```
cmd/server/          — точка входа
internal/
  app/
    handlers/        — HTTP-обработчики (один каталог = один endpoint)
  domain/
    model/           — доменные модели и ошибки
    cart_items/
      repository/    — работа с таблицей корзины в PostgreSQL
      service/       — бизнес-логика корзины
    products/
      client/        — gRPC-клиент сервиса товаров
      repository/    — кеш товаров в PostgreSQL
  infra/
    config/          — загрузка конфигурации из YAML
    http/            — middleware и round-tripper для замера времени
    metrics/         — Prometheus-метрики (запросы, БД)
migrations/          — SQL-миграции (goose)
swagger/             — сгенерированная Swagger-документация
```

## API

Базовый URL: `http://localhost:8080` (локально)

| Метод  | Путь                               | Описание                                      |
|--------|------------------------------------|-----------------------------------------------|
| GET    | `/user/{user_id}/cart`             | Получить содержимое корзины пользователя      |
| POST   | `/user/{user_id}/cart/{sku}`       | Добавить товар в корзину                      |
| DELETE | `/user/{user_id}/cart/{sku}`       | Удалить позицию из корзины                    |
| DELETE | `/user/{user_id}/cart`             | Очистить корзину полностью                    |
| POST   | `/user/{user_id}/cart/checkout`    | Оформить заказ (списать товары со склада)     |
| GET    | `/metrics`                         | Prometheus-метрики                            |
| GET    | `/swagger/`                        | Swagger UI                                    |

> `user_id` — UUID пользователя, `sku` — числовой идентификатор товара.

### Примеры

**Добавить товар в корзину:**
```
POST http://localhost:8080/user/550e8400-e29b-41d4-a716-446655440000/cart/1
Content-Type: application/json

{"count": 2}
```

**Получить корзину:**
```
GET http://localhost:8080/user/550e8400-e29b-41d4-a716-446655440000/cart
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
POST http://localhost:8080/user/550e8400-e29b-41d4-a716-446655440000/cart/checkout
```
```json
{"total_price": 200.0}
```

## Взаимодействие с сервисом товаров

Сервис корзины обращается к `ozon-simulator-go-products` по gRPC для:
- проверки существования товара и получения его данных при добавлении в корзину
- списания количества товаров со склада при оформлении заказа (`DecreaseProductCount`)

## Конфигурация

Путь до файла конфигурации задаётся переменной окружения `CONFIG_PATH`.

```yaml
server:
  host:
  port: 8080

products:
  schema: http
  host: localhost
  port: 8082
  auth-token: admin
  timeout: 30s

database:
  user: postgres
  password: 1234
  host: localhost
  port: 5432
  name: ozon_simulator_go_cart
```

## Запуск локально

### Зависимости

- Go 1.24+
- PostgreSQL
- [goose](https://github.com/pressly/goose)
- Запущенный `ozon-simulator-go-products`

### Миграции

```bash
make up-migrations
```

> По умолчанию подключается к `postgresql://postgres:1234@127.0.0.1:5432/ozon_simulator_go_cart`

### Сервер

```bash
CONFIG_PATH=configs/values_local.yaml go run ./cmd/server/main.go
```

## Docker

```bash
# Собрать образ сервиса
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