package eventbus_test

import (
	"testing"

	"github.com/go-jimu/template/internal/pkg/eventbus"
)

func validConfig() eventbus.Config {
	return eventbus.Config{
		BufferSize: 1, Workers: 1, HandlerTimeout: "1s",
		Outbox: eventbus.OutboxConfig{
			Lease: "30s", PollInterval: "1s", Workers: 1, BatchSize: 1,
			MinBackoff: "1s", MaxBackoff: "1m", Retention: "24h",
		},
	}
}

func TestConfigValidate(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Fatal(err)
	}

	for name, mutate := range map[string]func(*eventbus.Config){
		"zero handler timeout":      func(c *eventbus.Config) { c.HandlerTimeout = "0s" },
		"missing outbox lease":      func(c *eventbus.Config) { c.Outbox.Lease = "" },
		"zero outbox poll":          func(c *eventbus.Config) { c.Outbox.PollInterval = "0s" },
		"zero outbox workers":       func(c *eventbus.Config) { c.Outbox.Workers = 0 },
		"zero outbox batch":         func(c *eventbus.Config) { c.Outbox.BatchSize = 0 },
		"max backoff below min":     func(c *eventbus.Config) { c.Outbox.MaxBackoff = "500ms" },
		"negative outbox retention": func(c *eventbus.Config) { c.Outbox.Retention = "-1h" },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := validConfig()
			mutate(&invalid)
			if err := invalid.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}
