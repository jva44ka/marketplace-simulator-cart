//go:generate swag init -g cmd/server/main.go --dir ./internal,./cmd
package main

import (
	"context"
	"log/slog"
	"os"

	appPkg "github.com/jva44ka/marketplace-simulator-cart/internal/app"
	"github.com/jva44ka/marketplace-simulator-cart/internal/infra/config"
	"github.com/jva44ka/marketplace-simulator-cart/internal/infra/tracing"
)

func main() {
	slog.Info("app starting")

	ctx := context.Background()

	configImpl, err := config.LoadConfig(os.Getenv("CONFIG_PATH"))
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	if configImpl.Tracing.Enabled {
		shutdown, err := tracing.InitTracer(ctx, "cart", configImpl.Tracing.OtlpEndpoint)
		if err != nil {
			slog.Error("failed to init tracer", "err", err)
			os.Exit(1)
		}
		defer shutdown(ctx)
	}

	app, err := appPkg.NewApp(configImpl)
	if err != nil {
		slog.Error("failed to create app", "err", err)
		os.Exit(1)
	}

	if err = app.ListenAndServe(ctx); err != nil {
		slog.Error("app stopped", "err", err)
		os.Exit(1)
	}
}
