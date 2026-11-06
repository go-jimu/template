package main

import (
	"testing"

	"github.com/go-jimu/components/config/loader"
	"go.uber.org/fx"
)

// Intent: the shipped defaults load and validate, and the Fx graph wires
// every constructor, without connecting to MySQL or Kafka.
func TestDefaultsWireTheApplication(t *testing.T) {
	var option Option
	if err := loader.Load(&option, loader.WithConfigurationDirectory("../../configs/server", "defaults")); err != nil {
		t.Fatal(err)
	}
	if err := option.EventBus.Validate(); err != nil {
		t.Fatalf("eventbus defaults: %v", err)
	}
	if err := fx.ValidateApp(application(option)); err != nil {
		t.Fatal(err)
	}
}
