//go:generate swag init -g cmd/server/main.go --dir ./internal,./cmd
package main

import (
	"context"
	"log/slog"
	"os"

	appPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/app"
)

func main() {
	slog.Info("app starting")

	app, err := appPkg.NewApp(os.Getenv("CONFIG_PATH"))
	if err != nil {
		slog.Error("failed to create app", "err", err)
		os.Exit(1)
	}

	if err = app.ListenAndServe(context.Background()); err != nil {
		slog.Error("app stopped", "err", err)
		os.Exit(1)
	}
}
