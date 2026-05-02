# marketplace-simulator-cart

Микросервис корзины покупок — часть учебного проекта «Симулятор маркетплейса».

REST API на Go, PostgreSQL, Outbox pattern для надёжной доставки подтверждений резервирований в сервис товаров. Circuit breaker + retry для gRPC-клиента. Prometheus-метрики, OpenTelemetry-трейсы.

→ [Подробная документация](docs/README.md)

Запуск в составе полного стека: [marketplace-simulator](https://github.com/jva44ka/marketplace-simulator)
