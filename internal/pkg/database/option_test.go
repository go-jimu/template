package database_test

import (
	"testing"

	"github.com/go-jimu/template/internal/pkg/database"
)

func TestOptionValidate(t *testing.T) {
	valid := database.Option{
		Host: "localhost", Port: 3306, User: "root", Database: "jimu",
		MaxOpenConns: 12, MaxIdleConns: 12,
		MaxIdleTime: "5m", MaxLifetime: "30m",
	}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}

	invalid := valid
	invalid.MaxIdleConns = invalid.MaxOpenConns + 1
	if err := invalid.Validate(); err == nil {
		t.Fatal("Validate() error = nil for invalid connection limits")
	}
}
