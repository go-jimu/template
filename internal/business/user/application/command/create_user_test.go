package command_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/user/application/command"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/samber/oops"
)

type repositoryFake struct {
	user      *domain.User
	getErr    error
	saveErr   error
	saveCalls int
	order     *[]string
}

func (r *repositoryFake) Get(context.Context, string) (*domain.User, error) {
	return r.user, r.getErr
}

func (r *repositoryFake) Save(context.Context, *domain.User) error {
	r.saveCalls++
	if r.order != nil {
		*r.order = append(*r.order, "save")
	}
	return r.saveErr
}

type dispatcherFake struct {
	dispatchErr error
	events      []event.Event
	order       *[]string
}

func (d *dispatcherFake) Dispatch(domainEvent event.Event) error {
	return d.DispatchAll([]event.Event{domainEvent})
}

func (d *dispatcherFake) DispatchAll(events []event.Event) error {
	if d.order != nil {
		*d.order = append(*d.order, "dispatch")
	}
	d.events = append(d.events, events...)
	return d.dispatchErr
}

func (*dispatcherFake) Close(context.Context) error { return nil }

func TestCreateUserPersistsBeforeDispatch(t *testing.T) {
	order := make([]string, 0, 2)
	repository := &repositoryFake{order: &order}
	dispatcher := &dispatcherFake{order: &order}
	handler := command.NewCreateUserHandler(repository, dispatcher)
	var logs bytes.Buffer
	ctx := sloghelper.NewContext(
		context.Background(),
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)

	result, err := handler.Handle(ctx, command.CreateUser{
		Name: "Alice", Password: "current-pass", Email: "alice@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ID == "" || result.Email != "alice@example.com" {
		t.Fatalf("result = %#v", result)
	}
	if len(order) != 2 || order[0] != "save" || order[1] != "dispatch" {
		t.Fatalf("operation order = %v, want [save dispatch]", order)
	}
	if len(dispatcher.events) != 1 {
		t.Fatalf("dispatched event count = %d, want 1", len(dispatcher.events))
	}
	for _, field := range []string{
		`"operation":"user.create"`,
		`"outcome":"success"`,
		`"duration_ms":`,
		`"user_id":`,
	} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("completion log %q does not contain %s", logs.String(), field)
		}
	}
}

func TestCreateUserDoesNotDispatchWhenSaveFails(t *testing.T) {
	saveErr := errors.New("database unavailable")
	repository := &repositoryFake{saveErr: saveErr}
	dispatcher := &dispatcherFake{}
	handler := command.NewCreateUserHandler(repository, dispatcher)

	_, err := handler.Handle(context.Background(), command.CreateUser{
		Name: "Alice", Password: "current-pass", Email: "alice@example.com",
	})
	if !errors.Is(err, saveErr) {
		t.Fatalf("Handle() error = %v, want %v", err, saveErr)
	}
	if len(dispatcher.events) != 0 {
		t.Fatalf("dispatched %d events after failed save", len(dispatcher.events))
	}
}

func TestCreateUserReportsSuccessWhenBestEffortDispatchIsClosed(t *testing.T) {
	repository := &repositoryFake{}
	dispatcher := &dispatcherFake{dispatchErr: event.ErrDispatcherClosed}
	handler := command.NewCreateUserHandler(repository, dispatcher)
	var logs bytes.Buffer
	ctx := sloghelper.NewContext(
		context.Background(),
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)

	result, err := handler.Handle(ctx, command.CreateUser{
		Name: "Alice", Password: "current-pass", Email: "alice@example.com",
	})
	if err != nil || result.ID == "" {
		t.Fatalf("Handle() = %#v, %v; want persisted success", result, err)
	}
	if strings.Count(logs.String(), "\n") != 1 ||
		!strings.Contains(logs.String(), `"event_dispatch_outcome":"failed"`) ||
		!strings.Contains(logs.String(), `"outcome":"success"`) {
		t.Fatalf("completion log = %q", logs.String())
	}
}

func TestChangePasswordSavesChangedAggregate(t *testing.T) {
	user, err := domain.NewUser("Alice", "current-pass", "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	user.Events.Drain()
	repository := &repositoryFake{user: user}
	handler := command.NewChangePasswordHandler(repository)

	if err = handler.Handle(context.Background(), command.ChangePassword{
		UserID:          user.ID,
		CurrentPassword: "current-pass",
		NewPassword:     "replacement-pass",
	}); err != nil {
		t.Fatal(err)
	}
	if repository.saveCalls != 1 {
		t.Fatalf("Save() calls = %d, want 1", repository.saveCalls)
	}
}

func TestChangePasswordDoesNotSaveRejectedChange(t *testing.T) {
	user, err := domain.NewUser("Alice", "current-pass", "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	user.Events.Drain()
	repository := &repositoryFake{user: user}
	handler := command.NewChangePasswordHandler(repository)
	var logs bytes.Buffer
	ctx := sloghelper.NewContext(
		context.Background(),
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)

	err = handler.Handle(ctx, command.ChangePassword{
		UserID:          user.ID,
		CurrentPassword: "wrong-pass",
		NewPassword:     "replacement-pass",
	})
	if !errors.Is(err, domain.ErrPasswordMismatch) {
		t.Fatalf("Handle() error = %v, want %v", err, domain.ErrPasswordMismatch)
	}
	if repository.saveCalls != 0 {
		t.Fatalf("Save() calls = %d, want 0", repository.saveCalls)
	}
	enhanced, ok := oops.AsOops(err)
	if !ok || enhanced.Context()["user_id"] != user.ID {
		t.Fatalf("error context = %#v, want user_id %q", enhanced.Context(), user.ID)
	}
	for _, field := range []string{
		`"operation":"user.change_password"`,
		`"outcome":"failed"`,
		`"duration_ms":`,
	} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("completion log %q does not contain %s", logs.String(), field)
		}
	}
	if strings.Contains(logs.String(), `"error":`) {
		t.Fatalf("inner Application log contains error details: %q", logs.String())
	}
}
