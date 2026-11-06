package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/transaction"
	"github.com/go-jimu/components/transaction/transactiontest"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/domain"
)

type repositoryFake struct {
	user      *domain.User
	getErr    error
	saveErr   error
	saveCalls int
	order     *[]string
	// inTransaction records whether each Save ran inside a transaction.
	inTransaction []bool
}

func (r *repositoryFake) Get(context.Context, string) (*domain.User, error) {
	return r.user, r.getErr
}

func (r *repositoryFake) Save(ctx context.Context, _ *domain.User) error {
	r.saveCalls++
	if r.order != nil {
		*r.order = append(*r.order, "save")
	}
	_, err := transaction.Value(ctx, saveProbe{}, func() bool { return true })
	r.inTransaction = append(r.inTransaction, err == nil)
	return r.saveErr
}

type saveProbe struct{}

type dispatcherFake struct {
	dispatchErr   error
	events        []event.Event
	order         *[]string
	inTransaction []bool
}

func (d *dispatcherFake) Dispatch(ctx context.Context, domainEvent event.Event) error {
	return d.DispatchAll(ctx, []event.Event{domainEvent})
}

func (d *dispatcherFake) DispatchAll(ctx context.Context, events []event.Event) error {
	_, err := transaction.Value(ctx, saveProbe{}, func() bool { return true })
	d.inTransaction = append(d.inTransaction, err == nil)
	if d.order != nil {
		*d.order = append(*d.order, "dispatch")
	}
	d.events = append(d.events, events...)
	return d.dispatchErr
}

func (*dispatcherFake) Close(context.Context) error { return nil }

func newApplication(
	t *testing.T,
	repository domain.Repository,
	dispatcher event.Dispatcher,
) (*application.Application, *transactiontest.Driver) {
	t.Helper()
	driver := new(transactiontest.Driver)
	manager, err := transaction.NewManager(driver)
	if err != nil {
		t.Fatal(err)
	}
	return application.NewApplication(manager, repository, dispatcher), driver
}

// Intent: the User and its events are written in one committed transaction,
// so the Outbox records the events exactly when the User is saved.
func TestCreateUserSavesAndDispatchesInOneTransaction(t *testing.T) {
	order := make([]string, 0, 2)
	repository := &repositoryFake{order: &order}
	dispatcher := &dispatcherFake{order: &order}
	app, driver := newApplication(t, repository, dispatcher)

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
	if len(driver.Transactions()) != 1 || driver.Last().State() != transactiontest.Committed {
		t.Fatalf("transactions = %d, last = %s, want one committed", len(driver.Transactions()), driver.Last().State())
	}
	if !repository.inTransaction[0] || !dispatcher.inTransaction[0] {
		t.Fatalf("save/dispatch in transaction = %v/%v, want both", repository.inTransaction, dispatcher.inTransaction)
	}
}

func TestCreateUserDoesNotDispatchWhenSaveFails(t *testing.T) {
	saveErr := errors.New("database unavailable")
	dispatcher := &dispatcherFake{}
	app, driver := newApplication(t, &repositoryFake{saveErr: saveErr}, dispatcher)

	_, err := app.CreateUser(context.Background(), application.CreateUser{
		Name: "Alice", Password: "current-pass", Email: "alice@example.com",
	})
	if !errors.Is(err, saveErr) {
		t.Fatalf("CreateUser() error = %v, want %v", err, saveErr)
	}
	if len(dispatcher.events) != 0 {
		t.Fatalf("dispatched %d events after failed save", len(dispatcher.events))
	}
	if driver.Last().State() != transactiontest.RolledBack {
		t.Fatalf("transaction %s, want rolled back", driver.Last().State())
	}
}

// Intent: events that cannot be recorded fail the registration, and the User
// is rolled back with them, so no User exists without its events.
func TestCreateUserRollsBackWhenDispatchFails(t *testing.T) {
	dispatcher := &dispatcherFake{dispatchErr: event.ErrDispatcherClosed}
	app, driver := newApplication(t, &repositoryFake{}, dispatcher)

	_, err := app.CreateUser(context.Background(), application.CreateUser{
		Name: "Alice", Password: "current-pass", Email: "alice@example.com",
	})
	if !errors.Is(err, event.ErrDispatcherClosed) {
		t.Fatalf("CreateUser() error = %v, want %v", err, event.ErrDispatcherClosed)
	}
	if driver.Last().State() != transactiontest.RolledBack {
		t.Fatalf("transaction %s, want rolled back", driver.Last().State())
	}
}

func TestChangePasswordSavesChangedUser(t *testing.T) {
	user, err := domain.NewUser("Alice", "current-pass", "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	repository := &repositoryFake{user: user}
	app, _ := newApplication(t, repository, &dispatcherFake{})

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
	app, _ := newApplication(t, repository, &dispatcherFake{})

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
