//go:generate swag init -g cmd/server/main.go --dir ./internal,./cmd
package main

import (
	"fmt"
	"os"

	appPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/app"
)

func main() {
	fmt.Println("app starting")

	app, err := appPkg.NewApp(os.Getenv("CONFIG_PATH"))
	if err != nil {
		panic(err)
	}

	if err = app.ListenAndServe(); err != nil {
		panic(err)
	}
}
