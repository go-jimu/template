package domain_test

import (
	"errors"
	"testing"

	"github.com/go-jimu/template/internal/business/user/domain"
)

func TestNewUserCreatesValidAggregateAndRecordsFact(t *testing.T) {
	user, err := domain.NewUser("  Alice  ", "correct-horse", "ALICE@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "Alice" || user.Email != "alice@example.com" {
		t.Fatalf("normalized user = %q/%q", user.Name, user.Email)
	}
	if user.Version != 0 {
		t.Fatalf("new User version = %d, want 0", user.Version)
	}

	events := user.Events.Drain()
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	created, ok := events[0].(domain.UserCreated)
	if !ok {
		t.Fatalf("event type = %T, want domain.UserCreated", events[0])
	}
	if created.UserID != user.ID || created.Email != user.Email || created.OccurredAt.IsZero() {
		t.Fatalf("created event = %#v", created)
	}
	if events = user.Events.Drain(); len(events) != 0 {
		t.Fatalf("second drain count = %d, want 0", len(events))
	}
}

func TestNewUserRejectsInvalidPassword(t *testing.T) {
	_, err := domain.NewUser("Alice", "short", "alice@example.com")
	if !errors.Is(err, domain.ErrInvalidPassword) {
		t.Fatalf("NewUser() error = %v, want %v", err, domain.ErrInvalidPassword)
	}
}

func TestChangePasswordProtectsCurrentCredential(t *testing.T) {
	user, err := domain.NewUser("Alice", "current-pass", "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	user.Events.Drain()

	if err = user.ChangePassword("wrong-pass", "replacement-pass"); !errors.Is(err, domain.ErrPasswordMismatch) {
		t.Fatalf("wrong current password error = %v", err)
	}
	if err = user.ChangePassword("current-pass", "replacement-pass"); err != nil {
		t.Fatal(err)
	}
	if err = user.ChangePassword("replacement-pass", "third-password"); err != nil {
		t.Fatalf("new credential was not installed: %v", err)
	}
}
