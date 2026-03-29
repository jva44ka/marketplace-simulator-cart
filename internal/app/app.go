package app

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/handlers/add_products_to_cart_handler"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/handlers/checkout_handler"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/handlers/clean_cart_handler"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/handlers/get_cart_items_by_user_id_handler"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/handlers/remove_products_from_cart_handler"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/middlewares"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/round_trippers"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/validation"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/config"
	productsRepositoryPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/infra/database/repository"
	productsClientPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/infra/external_services/products"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/metrics"
	cartItemsServicePkg "github.com/jva44ka/ozon-simulator-go-cart/internal/service/cart_item"
	_ "github.com/jva44ka/ozon-simulator-go-cart/swagger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

type App struct {
	config *config.Config
	server http.Server
}

func NewApp(cfg *config.Config) (*App, error) {
	app := &App{config: cfg}

	var err error
	app.server.Handler, err = bootstrapHandler(cfg)
	if err != nil {
		return nil, fmt.Errorf("bootstrapHandler: %w", err)
	}

	return app, nil
}

func (app *App) ListenAndServe(_ context.Context) error {
	address := fmt.Sprintf("%s:%s", app.config.Server.Host, app.config.Server.Port)

	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	return app.server.Serve(l)
}

func bootstrapHandler(config *config.Config) (http.Handler, error) {
	_ = round_trippers.NewTimerRoundTipper(http.DefaultTransport)

	productClient, err := productsClientPkg.NewProductClient(
		config.Products.Host,
		config.Products.Port,
		config.Products.AuthToken,
		config.Products.Timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("productsClientPkg.NewProductClient: %w", err)
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
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}

	dbMetrics := metrics.NewDbMetrics()
	productRepository := productsRepositoryPkg.NewPgxProductRepository(pool, dbMetrics)
	cartRepository := productsRepositoryPkg.NewPgxCartItemRepository(pool, dbMetrics)
	cartService := cartItemsServicePkg.NewCartItemService(cartRepository, productClient, productRepository)
	validator := validation.Validator{}

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

	return middlewares.NewTimerMiddleware(mx, metrics.NewRequestMetrics()), nil
}