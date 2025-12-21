package app

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jva44ka/ozon-simulator-go-cart/docs"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/handlers/add_products_to_cart_handler"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/handlers/clean_cart_handler"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/handlers/get_cart_items_by_user_id_handler"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/app/handlers/remove_products_from_cart_handler"
	httpSwagger "github.com/swaggo/http-swagger"

	cartItemsRepositoryPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/domain/cart_items/repository"
	cartItemsServicePkg "github.com/jva44ka/ozon-simulator-go-cart/internal/domain/cart_items/service"
	productsClientPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/domain/products/client"
	productsRepositoryPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/domain/products/repository"
	productsServicePkg "github.com/jva44ka/ozon-simulator-go-cart/internal/domain/products/service"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/config"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/http/middlewares"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/http/round_trippers"
)

type App struct {
	config *config.Config
	server http.Server
}

func NewApp(configPath string) (*App, error) {
	configImpl, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("config.LoadConfig: %w", err)
	}

	app := &App{
		config: configImpl,
	}

	app.server.Handler, err = boostrapHandler(configImpl)
	if err != nil {
		return nil, fmt.Errorf("boostrapHandler: %w", err)
	}

	return app, nil
}

func (app *App) ListenAndServe() error {
	address := fmt.Sprintf("%s:%s", app.config.Server.Host, app.config.Server.Port)

	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	return app.server.Serve(l)
}

func boostrapHandler(config *config.Config) (http.Handler, error) {
	tr := http.DefaultTransport
	tr = round_trippers.NewTimerRoundTipper(tr)

	client := http.Client{Transport: tr}

	productClient := productsClientPkg.NewProductClient(
		client,
		config.Products.Token,
		fmt.Sprintf("%s://%s:%s", config.Products.Schema, config.Products.Host, config.Products.Port),
	)

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

	productRepository := productsRepositoryPkg.NewPgxProductRepository(pool)
	productService := productsServicePkg.NewProductService(productRepository)

	cartRepository := cartItemsRepositoryPkg.NewPgxCartItemRepository(pool)
	cartService := cartItemsServicePkg.NewCartService(cartRepository, productClient, productRepository)

	mx := http.NewServeMux()
	mx.Handle("GET /user/{user_id}/cart", get_cart_items_by_user_id_handler.NewGetCartItemsByUserIdHandler(cartService, productService))
	mx.Handle("POST /user/{user_id}/cart/{sku_id}", add_products_to_cart_handler.NewAddProductsToCartHandler(cartService))
	mx.Handle("DELETE /user/{user_id}/cart/{sku_id}", remove_products_from_cart_handler.NewRemoveProductsFromCartHandler(cartService))
	mx.Handle("DELETE /user/{user_id}/cart", clean_cart_handler.NewCleanCartHandler(cartService))
	mx.Handle("/swagger/", httpSwagger.WrapHandler)

	middleware := middlewares.NewTimerMiddleware(mx)

	return middleware, nil
}
