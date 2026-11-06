//go:build integration

package infrastructure

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/go-jimu/components/ddd/event/outbox"
	"github.com/go-jimu/components/ddd/message"
	jimuxorm "github.com/go-jimu/contrib/transaction/xorm"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/application/eventhandler"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/go-jimu/template/internal/pkg/database"
	"github.com/go-jimu/template/internal/pkg/eventbus"
	"xorm.io/xorm"
)

type publishedMessages chan message.Message

func (p publishedMessages) Publish(_ context.Context, m message.Message) error {
	p <- m
	return nil
}

// Intent: registration commits the User and its outbox record together, the
// Relay publishes UserRegisteredV1 under the record's stable message ID, and a
// rolled back registration leaves neither.
func TestMySQLRegistrationThroughOutbox(t *testing.T) {
	engine := newMySQLEngine(t)
	db, err := jimuxorm.New(engine)
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := eventbus.Config{
		BufferSize: 1, Workers: 1, HandlerTimeout: "1s",
		Outbox: eventbus.OutboxConfig{
			Owner: "integration", Lease: "30s", PollInterval: "100ms", Workers: 1, BatchSize: 8,
			MinBackoff: "100ms", MaxBackoff: "1s", Retention: "24h",
		},
	}
	bestEffort, _, err := eventbus.NewBestEffort(config, logger)
	if err != nil {
		t.Fatal(err)
	}
	registry := outbox.NewRegistry()
	if err = outbox.Register[domain.UserCreated](registry); err != nil {
		t.Fatal(err)
	}
	relay, dispatcher, err := eventbus.NewOutbox(config, db, registry, bestEffort, logger)
	if err != nil {
		t.Fatal(err)
	}
	published := make(publishedMessages, 1)
	handler := eventhandler.NewUserCreatedHandler(published)
	if err = relay.Subscribe(handler); err != nil {
		t.Fatal(err)
	}
	transactor := database.NewTransactor(db)
	repository := NewUserRepository(db)
	app := application.NewApplication(transactor, repository, dispatcher)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = relay.Run(ctx) }()
	t.Cleanup(func() {
		closeCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = relay.Close(closeCtx)
	})

	t.Run("committed registration is published once under the record ID", func(t *testing.T) {
		created, err := app.CreateUser(ctx, application.CreateUser{
			Name: "Outbox", Password: "current-pass", Email: "outbox@example.com",
		})
		if err != nil {
			t.Fatal(err)
		}

		var sent message.Message
		select {
		case sent = <-published:
		case <-time.After(10 * time.Second):
			t.Fatal("UserRegisteredV1 was not published")
		}
		var recordID string
		if _, err = engine.SQL("SELECT id FROM outbox_records WHERE handler = ?", handler.Name()).Get(&recordID); err != nil {
			t.Fatal(err)
		}
		if sent.ID() != recordID+"/0" || sent.Key() != created.ID {
			t.Fatalf("message ID/key = %q/%q, want %q/%q", sent.ID(), sent.Key(), recordID+"/0", created.ID)
		}
		waitForStatus(t, engine, recordID, outbox.StatusDone)
		select {
		case again := <-published:
			t.Fatalf("published again: %q", again.ID())
		default:
		}
	})

	t.Run("rolled back registration leaves no User and no record", func(t *testing.T) {
		user, err := domain.NewUser("Rollback", "current-pass", "rollback@example.com")
		if err != nil {
			t.Fatal(err)
		}
		abort := errors.New("abort registration")
		err = transactor.Within(ctx, func(ctx context.Context) error {
			if err := repository.Save(ctx, user); err != nil {
				return err
			}
			if err := dispatcher.DispatchAll(ctx, user.Events.Drain()); err != nil {
				return err
			}
			return abort
		})
		if !errors.Is(err, abort) {
			t.Fatalf("Within() error = %v, want %v", err, abort)
		}
		if _, err = repository.Get(ctx, user.ID); !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("Get() error = %v, want %v", err, domain.ErrUserNotFound)
		}
		var records int
		if _, err = engine.SQL("SELECT COUNT(*) FROM outbox_records").Get(&records); err != nil {
			t.Fatal(err)
		}
		if records != 1 {
			t.Fatalf("outbox records = %d, want only the committed registration's", records)
		}
	})
}

func waitForStatus(t *testing.T, engine *xorm.Engine, recordID string, want outbox.Status) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		var status int
		if _, err := engine.SQL("SELECT status FROM outbox_records WHERE id = ?", recordID).Get(&status); err != nil {
			t.Fatal(err)
		}
		if outbox.Status(status) == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("record status = %d, want %d", status, want)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
