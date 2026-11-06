package main

import (
	"log/slog"
	"os"

	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/app"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

func main() {
	option, err := app.LoadOption()
	if err != nil {
		slog.Error("failed to load configuration", sloghelper.Error(oops.Wrap(err)))
		os.Exit(1)
	}
	application := fx.New(app.New(option))
	slog.Info("starting application")
	application.Run()
}
