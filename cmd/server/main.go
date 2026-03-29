//go:generate swag init -g cmd/server/main.go --dir ./internal,./cmd
package main

import (
	"context"
	"log/slog"
	"os"

	appPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/app"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/infra/config"
)

func main() {
	slog.Info("app starting")

	configImpl, err := config.LoadConfig(os.Getenv("CONFIG_PATH"))
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	app, err := appPkg.NewApp(configImpl)
	if err != nil {
		slog.Error("failed to create app", "err", err)
		os.Exit(1)
	}

	if err = app.ListenAndServe(context.Background()); err != nil {
		slog.Error("app stopped", "err", err)
		os.Exit(1)
	}
}
