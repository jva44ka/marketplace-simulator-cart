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
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/interceptors"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/middlewares"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/validation"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/config"
	databasePkg "github.com/jva44ka/ozon-simulator-go-cart/internal/infra/database"
	productsClientPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/infra/external_services/products"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/metrics"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/jobs"
	cartItemPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/service/cart_item"
	outboxServicePkg "github.com/jva44ka/ozon-simulator-go-cart/internal/service/outbox"
	_ "github.com/jva44ka/ozon-simulator-go-cart/swagger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type App struct {
	config           *config.Config
	server           http.Server
	outboxJob        *jobs.ReservationConfirmationOutboxJob
	outboxMonitorJob *jobs.OutboxMonitorJob
}

func NewApp(cfg *config.Config) (*App, error) {
	app := &App{config: cfg}

	var err error
	app.server.Handler, app.outboxJob, app.outboxMonitorJob, err = bootstrapHandler(cfg)
	if err != nil {
		return nil, fmt.Errorf("bootstrapHandler: %w", err)
	}

	return app, nil
}

func (app *App) ListenAndServe(ctx context.Context) error {
	address := fmt.Sprintf("%s:%s", app.config.Server.Host, app.config.Server.Port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	errGroup, ctx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		slog.Info("starting reservation confirmation job")
		app.outboxJob.Run(ctx)
		return nil
	})

	errGroup.Go(func() error {
		slog.Info("starting outbox monitor job")
		app.outboxMonitorJob.Run(ctx)
		return nil
	})

	errGroup.Go(func() error {
		return app.server.Serve(listener)
	})

	errGroup.Go(func() error {
		<-ctx.Done()
		return app.server.Shutdown(context.Background())
	})

	return errGroup.Wait()
}

func bootstrapHandler(config *config.Config) (http.Handler, *jobs.ReservationConfirmationOutboxJob, *jobs.OutboxMonitorJob, error) {
	productClient, err := productsClientPkg.NewProductClient(
		config.Products.Host,
		config.Products.Port,
		config.Products.AuthToken,
		config.Products.Timeout,
		grpc.WithUnaryInterceptor(interceptors.NewTimerInterceptor()),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("productsClientPkg.NewProductClient: %w", err)
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
		return nil, nil, nil, fmt.Errorf("pgxpool.New: %w", err)
	}

	dbMetrics := metrics.NewDbMetrics()
	db := databasePkg.NewDBManager(pool, dbMetrics)
	recordBuilder := outboxServicePkg.NewReservationConfirmationRecordBuilder()
	cartService := cartItemPkg.NewCartItemService(db, productClient, recordBuilder)
	validator := validation.Validator{}

	outboxJobInterval, err := time.ParseDuration(config.Jobs.ReservationConfirmationOutbox.JobInterval)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse reservation-confirmation.job-interval: %w", err)
	}

	outboxMonitorJobInterval, err := time.ParseDuration(config.Jobs.ReservationConfirmationOutboxMonitor.JobInterval)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse outbox-monitor.job-interval: %w", err)
	}

	outboxMetrics := metrics.NewOutboxMetrics()
	outboxMonitorMetrics := metrics.NewOutboxMonitorMetrics()

	outboxJob := jobs.NewReservationConfirmationOutboxJob(
		pool,
		db.OutboxPgxRepo(),
		productClient,
		outboxMetrics,
		config.Jobs.ReservationConfirmationOutbox.Enabled,
		outboxJobInterval,
		config.Jobs.ReservationConfirmationOutbox.BatchSize,
		config.Jobs.ReservationConfirmationOutbox.MaxRetries,
	)

	outboxMonitorJob := jobs.NewOutboxMonitorJob(
		db.OutboxPgxRepo(),
		outboxMonitorMetrics,
		config.Jobs.ReservationConfirmationOutboxMonitor.Enabled,
		outboxMonitorJobInterval,
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

	return middlewares.NewTimerMiddleware(mx, metrics.NewRequestMetrics()), outboxJob, outboxMonitorJob, nil
}
