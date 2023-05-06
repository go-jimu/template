package pkg

import (
	"github.com/go-jimu/template/internal/pkg/eventbus"
	"github.com/go-jimu/template/internal/pkg/log"
	"go.uber.org/fx"
)

var Module = fx.Module("internal.pkg",
	fx.Invoke(log.NewLog),
	fx.Provide(eventbus.New),
)
