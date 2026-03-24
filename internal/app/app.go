package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/handlers/add_products_to_cart_handler"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/handlers/checkout_handler"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/handlers/clean_cart_handler"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/handlers/get_cart_items_by_user_id_handler"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/handlers/remove_products_from_cart_handler"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/validation"
	_ "github.com/jva44ka/ozon-simulator-go-cart/swagger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"

	cartItemsRepositoryPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/domain/cart_items/repository"
	cartItemsServicePkg "github.com/jva44ka/ozon-simulator-go-cart/internal/domain/cart_items/service"
	productsClientPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/domain/products/client"
	productsRepositoryPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/domain/products/repository"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/config"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/http/middlewares"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/http/round_trippers"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/kafka"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/metrics"
)

type App struct {
	config   *config.Config
	server   http.Server
	consumer *kafka.Consumer
}

func NewApp(configPath string) (*App, error) {
	configImpl, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("config.LoadConfig: %w", err)
	}

	app := &App{config: configImpl}

	app.server.Handler, app.consumer, err = bootstrapHandler(configImpl)
	if err != nil {
		return nil, fmt.Errorf("bootstrapHandler: %w", err)
	}

	return app, nil
}

func (app *App) ListenAndServe(ctx context.Context) error {
	go func() {
		slog.Info("starting reservation expiry consumer")
		app.consumer.Run(ctx)
	}()

	address := fmt.Sprintf("%s:%s", app.config.Server.Host, app.config.Server.Port)

	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	return app.server.Serve(l)
}

func bootstrapHandler(config *config.Config) (http.Handler, *kafka.Consumer, error) {
	_ = round_trippers.NewTimerRoundTipper(http.DefaultTransport)

	productClient, err := productsClientPkg.NewProductClient(
		config.Products.Host,
		config.Products.Port,
		config.Products.AuthToken,
		config.Products.Timeout,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("productsClientPkg.NewProductClient: %w", err)
	}

	pool, err := pgxpool.New(context.Background(), fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		config.Database.User,
		config.Database.Password,
		config.Database.Host,
		config.Database.Port,
		config.Database.Name,
	))
	if err != nil {
		return nil, nil, fmt.Errorf("pgxpool.New: %w", err)
	}

	reservationTTL, err := time.ParseDuration(config.Reservation.TTL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse reservation.ttl: %w", err)
	}

	dbMetrics := metrics.NewDbMetrics()
	productRepository := productsRepositoryPkg.NewPgxProductRepository(pool, dbMetrics)
	cartRepository := cartItemsRepositoryPkg.NewPgxCartItemRepository(pool, dbMetrics)
	cartService := cartItemsServicePkg.NewCartService(cartRepository, productClient, productRepository, reservationTTL)
	validator := validation.Validator{}

	consumer := kafka.NewConsumer(
		config.Kafka.Brokers,
		config.Kafka.ReservationExpiredTopic,
		config.Kafka.ConsumerGroup,
		cartRepository,
	)

	mx := http.NewServeMux()

	mx.Handle("GET /user/{user_id}/cart", get_cart_items_by_user_id_handler.NewGetCartItemsByUserIdHandler(
		cartService, validator))
	mx.Handle("POST /user/{user_id}/cart/{sku}", add_products_to_cart_handler.NewAddProductsToCartHandler(
		cartService, validator))
	mx.Handle("DELETE /user/{user_id}/cart/{sku}", remove_products_from_cart_handler.NewRemoveProductsFromCartHandler(
		cartService, validator))
	mx.Handle("DELETE /user/{user_id}/cart", clean_cart_handler.NewCleanCartHandler(
		cartService, validator))
	mx.Handle("POST /user/{user_id}/cart/checkout", checkout_handler.NewCheckoutHandler(
		cartService, validator))
	mx.Handle("/swagger/", httpSwagger.WrapHandler)
	mx.Handle("/metrics", promhttp.Handler())

	return middlewares.NewTimerMiddleware(mx, metrics.NewRequestMetrics()), consumer, nil
}
