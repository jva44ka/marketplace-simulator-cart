package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/config"
	productsRepositoryPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/infra/database/repository"
	productsClientPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/infra/external_services/products"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/kafka"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/metrics"
	cartItemsServicePkg "github.com/jva44ka/ozon-simulator-go-cart/internal/service/cart_item"
)

func main() {
	slog.Info("consumer starting")

	configImpl, err := config.LoadConfig(os.Getenv("CONFIG_PATH"))
	if err != nil {
		slog.Error("Failed to load config", "err", err)
		os.Exit(1)
	}

	if configImpl.Jobs.ReservationExpiredConsumer.Enabled == false {
		slog.Info("Consumer job is turn off. Shutting down")
		os.Exit(1)
	}

	consumer, err := createConsumer(configImpl)
	if err != nil {
		slog.Error("Failed to create consumer", "err", err)
		os.Exit(1)
	}
	defer consumer.Close()

	slog.Info("consumer started")
	consumer.Run(context.Background())
}

func createConsumer(configImpl *config.Config) (*kafka.Consumer, error) {
	productClient, err := productsClientPkg.NewProductClient(
		configImpl.Products.Host,
		configImpl.Products.Port,
		configImpl.Products.AuthToken,
		configImpl.Products.Timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to create product client: %w", err)
	}

	pool, err := pgxpool.New(context.Background(), fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		configImpl.Database.User,
		configImpl.Database.Password,
		configImpl.Database.Host,
		configImpl.Database.Port,
		configImpl.Database.Name,
	))
	if err != nil {
		return nil, fmt.Errorf("Failed to create db pool: %w", err)
	}

	dbMetrics := metrics.NewDbMetrics()
	productRepository := productsRepositoryPkg.NewPgxProductRepository(pool, dbMetrics)
	cartRepository := productsRepositoryPkg.NewPgxCartItemRepository(pool, dbMetrics)
	cartService := cartItemsServicePkg.NewCartItemService(cartRepository, productClient, productRepository)

	reservationTopicConfig, err := configImpl.Kafka.GetReservationExpiredTopicConfig()
	if err != nil {
		return nil, fmt.Errorf("Failed to get reservation-expired-topic config: %w", err)
	}

	consumer := kafka.NewConsumer(
		configImpl.Kafka.Brokers,
		reservationTopicConfig.Name,
		reservationTopicConfig.ConsumerGroup,
	)
	return consumer, nil
}
