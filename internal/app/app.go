package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jva44ka/marketplace-simulator-cart/internal/app/handlers/add_products_to_cart_handler"
	"github.com/jva44ka/marketplace-simulator-cart/internal/app/handlers/checkout_handler"
	"github.com/jva44ka/marketplace-simulator-cart/internal/app/handlers/clean_cart_handler"
	"github.com/jva44ka/marketplace-simulator-cart/internal/app/handlers/get_cart_items_by_user_id_handler"
	"github.com/jva44ka/marketplace-simulator-cart/internal/app/handlers/remove_products_from_cart_handler"
	"github.com/jva44ka/marketplace-simulator-cart/internal/app/interceptors"
	"github.com/jva44ka/marketplace-simulator-cart/internal/app/middlewares"
	"github.com/jva44ka/marketplace-simulator-cart/internal/app/validation"
	"github.com/jva44ka/marketplace-simulator-cart/internal/infra/circuitbreaker"
	"github.com/jva44ka/marketplace-simulator-cart/internal/infra/config"
	repoPkg "github.com/jva44ka/marketplace-simulator-cart/internal/infra/database/repository"
	transactorPkg "github.com/jva44ka/marketplace-simulator-cart/internal/infra/database/transactor"
	etcdPkg "github.com/jva44ka/marketplace-simulator-cart/internal/infra/etcd"
	productsClientPkg "github.com/jva44ka/marketplace-simulator-cart/internal/infra/external_services/products"
	"github.com/jva44ka/marketplace-simulator-cart/internal/infra/metrics"
	"github.com/jva44ka/marketplace-simulator-cart/internal/infra/tracing"
	"github.com/jva44ka/marketplace-simulator-cart/internal/jobs"
	cartItemPkg "github.com/jva44ka/marketplace-simulator-cart/internal/service/cart_item"
	outboxServicePkg "github.com/jva44ka/marketplace-simulator-cart/internal/service/outbox"
	_ "github.com/jva44ka/marketplace-simulator-cart/swagger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type App struct {
	cfg                *config.Config // начальный yaml-конфиг (для boot-параметров)
	cfgStore           *config.ConfigStore
	server             http.Server
	outboxJob          *jobs.ReservationConfirmationOutboxJob
	metricCollectorJob *jobs.MetricCollectorJob
	etcdClient         *clientv3.Client // nil если etcd не настроен
	etcdConfigKey      string
	applyDynamicConfig func(*config.Config)
}

func NewApp(cfg *config.Config) (*App, error) {
	// --- ConfigStore: начинаем с yaml-конфига ---
	cfgStore := config.NewConfigStore(cfg)

	// --- etcd: подключение и первоначальная загрузка ---
	var etcdClient *clientv3.Client
	if cfg.Etcd.Enabled {
		var err error
		etcdClient, err = etcdPkg.NewClient(cfg.Etcd)
		if err != nil {
			slog.Warn("etcd: failed to connect, using yaml defaults", "err", err)
			etcdClient = nil
		} else {
			initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if etcdCfg, found, err := etcdPkg.ReadFromEtcd(initCtx, etcdClient, cfg.Etcd.ConfigKey); err != nil {
				slog.Warn("etcd: failed to read config, using yaml defaults", "err", err)
			} else if found {
				cfgStore.Store(etcdCfg)
				slog.Info("etcd: loaded config from etcd")
			} else {
				// Первый старт: seeding
				if err := etcdPkg.SeedIfAbsent(initCtx, etcdClient, cfg.Etcd.ConfigKey, cfg); err != nil {
					slog.Warn("etcd: failed to seed config", "err", err)
				}
			}
		}
	}

	// Строим компоненты приложения
	handler, outboxJob, metricCollectorJob, cbExecutor, err := bootstrapHandler(cfgStore)
	if err != nil {
		return nil, fmt.Errorf("bootstrapHandler: %w", err)
	}

	// --- applyDynamicConfig: вызывается при каждом изменении в etcd ---
	applyDynamicConfig := func(newCfg *config.Config) {
		if cbExecutor != nil && newCfg.Products.CircuitBreaker.Enabled {
			cbExecutor.Update(newCfg.Products.CircuitBreaker)
		}
		slog.Info("dynamic config applied",
			"circuit-breaker.threshold", newCfg.Products.CircuitBreaker.Threshold,
			"retry.max-attempts", newCfg.Products.Retry.MaxAttempts,
		)
	}

	etcdConfigKey := ""
	if cfg.Etcd.Enabled {
		etcdConfigKey = cfg.Etcd.ConfigKey
	}

	app := &App{
		cfg:                cfg,
		cfgStore:           cfgStore,
		server:             http.Server{Handler: handler},
		outboxJob:          outboxJob,
		metricCollectorJob: metricCollectorJob,
		etcdClient:         etcdClient,
		etcdConfigKey:      etcdConfigKey,
		applyDynamicConfig: applyDynamicConfig,
	}

	return app, nil
}

func (app *App) ListenAndServe(ctx context.Context) error {
	address := fmt.Sprintf("%s:%s", app.cfg.Server.Host, app.cfg.Server.Port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	errGroup, ctx := errgroup.WithContext(ctx)

	// etcd watcher
	if app.etcdClient != nil {
		errGroup.Go(func() error {
			etcdPkg.Watch(ctx, app.etcdClient, app.etcdConfigKey, app.cfgStore, app.applyDynamicConfig)
			return nil
		})
	}

	errGroup.Go(func() error {
		slog.Info("starting reservation confirmation job")
		app.outboxJob.Run(ctx)
		return nil
	})

	errGroup.Go(func() error {
		slog.Info("starting metric collector job")
		app.metricCollectorJob.Run(ctx)
		return nil
	})

	errGroup.Go(func() error {
		return app.server.Serve(listener)
	})

	errGroup.Go(func() error {
		<-ctx.Done()
		var etcdCloseErr error
		if app.etcdClient != nil {
			etcdCloseErr = app.etcdClient.Close()
		}
		shutdownErr := app.server.Shutdown(context.Background())
		if etcdCloseErr != nil {
			return etcdCloseErr
		}
		return shutdownErr
	})

	return errGroup.Wait()
}

// bootstrapResult содержит результат инициализации компонентов приложения.
type bootstrapResult struct {
	handler            http.Handler
	outboxJob          *jobs.ReservationConfirmationOutboxJob
	metricCollectorJob *jobs.MetricCollectorJob
	cbExecutor         *circuitbreaker.Executor // nil если CB отключён
}

func bootstrapHandler(cfgStore *config.ConfigStore) (http.Handler, *jobs.ReservationConfirmationOutboxJob, *jobs.MetricCollectorJob, *circuitbreaker.Executor, error) {
	cfg := cfgStore.Load()

	unaryInterceptors := []grpc.UnaryClientInterceptor{}

	if cfg.Products.Retry.Enabled {
		unaryInterceptors = append(unaryInterceptors, interceptors.NewRetryInterceptor(cfgStore))
	}

	unaryInterceptors = append(unaryInterceptors, interceptors.NewTimerInterceptor())

	var cbExecutor *circuitbreaker.Executor
	if cfg.Products.CircuitBreaker.Enabled {
		var err error
		cbExecutor, err = circuitbreaker.NewExecutor(cfg.Products.CircuitBreaker, "product-client")
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("circuitbreaker.NewExecutor: %w", err)
		}
		unaryInterceptors = append(unaryInterceptors, cbExecutor.UnaryClientInterceptor())
	}

	productDialOpts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(unaryInterceptors...),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}

	productClient, err := productsClientPkg.NewProductClient(
		cfg.Products.Host,
		cfg.Products.Port,
		cfg.Products.AuthToken,
		cfg.Products.Timeout,
		productDialOpts...,
	)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("productsClientPkg.NewProductClient: %w", err)
	}

	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("pgxpool.ParseConfig: %w", err)
	}
	poolConfig.ConnConfig.Tracer = tracing.NewPgxTracer()
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("pgxpool.NewWithConfig: %w", err)
	}

	dbMetrics := metrics.NewDbMetrics()
	cartItemRepo := repoPkg.NewPgxCartItemRepository(pool, dbMetrics)
	productRepo := repoPkg.NewPgxProductRepository(pool, dbMetrics)
	outboxRepo := repoPkg.NewReservationConfirmationOutboxRepository(pool)
	transactor := transactorPkg.NewCartServiceTransactor(pool, dbMetrics)

	recordBuilder := outboxServicePkg.NewReservationConfirmationRecordBuilder()
	businessMetrics := metrics.NewBusinessMetrics()
	cartService := cartItemPkg.NewCartItemService(
		transactor, cartItemRepo, productRepo, outboxRepo,
		productClient, recordBuilder, businessMetrics,
	)
	validator := validation.Validator{}

	outboxMetrics := metrics.NewOutboxMetrics()
	metricCollectorMetrics := metrics.NewMetricCollectorMetrics()

	outboxJob := jobs.NewReservationConfirmationOutboxJob(
		outboxRepo,
		productClient,
		outboxMetrics,
		cfgStore,
	)

	metricCollectorJob := jobs.NewMetricCollectorJob(
		outboxRepo,
		cartItemRepo,
		pool,
		metricCollectorMetrics,
		cfgStore,
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

	timerHandler := middlewares.NewTimerMiddleware(mx, metrics.NewRequestMetrics())
	tracedHandler := otelhttp.NewHandler(timerHandler, "cart-http")

	return tracedHandler, outboxJob, metricCollectorJob, cbExecutor, nil
}
