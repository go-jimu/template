package eventbus_test

import (
	"testing"

	"github.com/go-jimu/template/internal/pkg/eventbus"
)

func TestConfigValidate(t *testing.T) {
	valid := eventbus.Config{
		BufferSize: 1, Workers: 1, DelayClose: "0s", HandlerTimeout: "1s",
	}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}

	invalid := valid
	invalid.HandlerTimeout = "0s"
	if err := invalid.Validate(); err == nil {
		t.Fatal("Validate() error = nil for zero handler timeout")
	}
}
