package application_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/domain"
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

func (d *dispatcherFake) Dispatch(ctx context.Context, domainEvent event.Event) error {
	return d.DispatchAll(ctx, []event.Event{domainEvent})
}

func (d *dispatcherFake) DispatchAll(_ context.Context, events []event.Event) error {
	if d.order != nil {
		*d.order = append(*d.order, "dispatch")
	}
	d.events = append(d.events, events...)
	return d.dispatchErr
}

func (*dispatcherFake) Close(context.Context) error { return nil }

func TestCreateUserPersistsBeforeDispatch(t *testing.T) {
	order := make([]string, 0, 2)
	dispatcher := &dispatcherFake{order: &order}
	app := application.NewApplication(&repositoryFake{order: &order}, dispatcher)

	result, err := app.CreateUser(context.Background(), application.CreateUser{
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
}

func TestCreateUserDoesNotDispatchWhenSaveFails(t *testing.T) {
	saveErr := errors.New("database unavailable")
	dispatcher := &dispatcherFake{}
	app := application.NewApplication(&repositoryFake{saveErr: saveErr}, dispatcher)

	_, err := app.CreateUser(context.Background(), application.CreateUser{
		Name: "Alice", Password: "current-pass", Email: "alice@example.com",
	})
	if !errors.Is(err, saveErr) {
		t.Fatalf("CreateUser() error = %v, want %v", err, saveErr)
	}
	if len(dispatcher.events) != 0 {
		t.Fatalf("dispatched %d events after failed save", len(dispatcher.events))
	}
}

func TestCreateUserSucceedsWhenDispatchFails(t *testing.T) {
	dispatcher := &dispatcherFake{dispatchErr: event.ErrDispatcherClosed}
	app := application.NewApplication(&repositoryFake{}, dispatcher)
	var logs bytes.Buffer
	ctx := sloghelper.NewContext(
		context.Background(),
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)

	if _, err := app.CreateUser(ctx, application.CreateUser{
		Name: "Alice", Password: "current-pass", Email: "alice@example.com",
	}); err != nil {
		t.Fatalf("CreateUser() error = %v, want nil after the User is saved", err)
	}
	for _, field := range []string{`"level":"WARN"`, `"operation":"user.events.dispatch"`, `"user_id":`} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("dispatch log %q does not contain %s", logs.String(), field)
		}
	}
}

func TestChangePasswordSavesChangedUser(t *testing.T) {
	user, err := domain.NewUser("Alice", "current-pass", "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	repository := &repositoryFake{user: user}
	app := application.NewApplication(repository, &dispatcherFake{})

	err = app.ChangePassword(context.Background(), application.ChangePassword{
		UserID: user.ID, CurrentPassword: "current-pass", NewPassword: "next-pass-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if repository.saveCalls != 1 {
		t.Fatalf("save calls = %d, want 1", repository.saveCalls)
	}
}

func TestChangePasswordDoesNotSaveRejectedChange(t *testing.T) {
	user, err := domain.NewUser("Alice", "current-pass", "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	repository := &repositoryFake{user: user}
	app := application.NewApplication(repository, &dispatcherFake{})

	err = app.ChangePassword(context.Background(), application.ChangePassword{
		UserID: user.ID, CurrentPassword: "wrong-pass", NewPassword: "next-pass-1",
	})
	if err == nil {
		t.Fatal("ChangePassword() error = nil, want rejection")
	}
	if repository.saveCalls != 0 {
		t.Fatalf("save calls = %d, want 0", repository.saveCalls)
	}
}
