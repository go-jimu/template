package main

import (
	"os"
	"testing"
	"time"

	"github.com/go-jimu/components/config/loader"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/notification"
	"github.com/go-jimu/template/internal/business/user"
	"github.com/go-jimu/template/internal/pkg"
	"github.com/go-jimu/template/internal/pkg/connectrpc"
	"github.com/go-jimu/template/internal/pkg/database"
	"github.com/go-jimu/template/internal/pkg/eventbus"
	"github.com/go-jimu/template/internal/pkg/messagebus"
	"go.uber.org/fx"
)

// This regression test protects the components v0.10 default -> defaults rename.
func TestDefaultConfigLoadsExpectedValues(t *testing.T) {
	t.Setenv("CONNECT_ADDR", ":8080")
	t.Setenv("MYSQL_HOST", "localhost")
	t.Setenv("MYSQL_PORT", "3306")
	t.Setenv("MYSQL_DATABASE", "jimu")
	t.Setenv("KAFKA_BROKER", "localhost:9092")
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(".."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(workingDirectory); err != nil {
			t.Fatal(err)
		}
	})

	var option Option
	if err = loader.Load(
		&option,
		loader.WithConfigurationDirectory("./configs", "defaults"),
	); err != nil {
		t.Fatal(err)
	}
	if option.Connect.Addr != ":8080" {
		t.Fatalf("Connect.Addr = %q, want %q", option.Connect.Addr, ":8080")
	}
	if option.MySQL.Host != "localhost" || option.MySQL.Port != 3306 {
		t.Fatalf(
			"MySQL endpoint = %s:%d, want localhost:3306",
			option.MySQL.Host,
			option.MySQL.Port,
		)
	}
	if option.EventBus.BufferSize != 1024 {
		t.Fatalf("EventBus.BufferSize = %d, want 1024", option.EventBus.BufferSize)
	}
	if len(option.Kafka.Brokers) != 1 ||
		option.Kafka.Brokers[0] != "localhost:9092" {
		t.Fatalf("Kafka.Brokers = %v, want [localhost:9092]", option.Kafka.Brokers)
	}
}

func TestApplicationDependencyGraphIsComplete(t *testing.T) {
	option := Option{
		Logger:   sloghelper.Options{},
		MySQL:    database.Option{},
		EventBus: eventbus.Config{},
		Kafka:    messagebus.Option{},
		Connect:  connectrpc.Option{},
	}
	if err := fx.ValidateApp(
		fx.Supply(option),
		fx.Provide(sloghelper.NewLog),
		fx.Provide(eventbus.NewDispatcher),
		pkg.Module,
		user.Module,
		notification.Module,
		fx.StartTimeout(15*time.Second),
		fx.StopTimeout(30*time.Second),
		fx.NopLogger,
	); err != nil {
		t.Fatal(err)
	}
}
