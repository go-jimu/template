package pkg

import (
	"github.com/go-jimu/template/internal/pkg/eventbus"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"internal.pkg",
	fx.Provide(eventbus.New),
)
