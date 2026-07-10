package infrastructure

import (
	"bytes"
	"errors"
	"testing"

	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/google/uuid"
)

func TestUserFromDOMapsPersistedAggregate(t *testing.T) {
	data := &userDO{
		ID:       uuid.NewString(),
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: []byte("persisted-hash"),
		Version:  3,
	}

	user, err := userFromDO(data)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != data.ID || user.Name != data.Name || user.Email != data.Email {
		t.Fatalf("mapped User = %#v", user)
	}
	if !bytes.Equal(user.HashedPassword, data.Password) || user.Version != data.Version {
		t.Fatalf("mapped credentials/version = %q/%d", user.HashedPassword, user.Version)
	}
	if events := user.Events.Drain(); len(events) != 0 {
		t.Fatalf("restored event count = %d, want 0", len(events))
	}
}

func TestUserFromDORejectsInvalidPersistedVersion(t *testing.T) {
	_, err := userFromDO(&userDO{Version: 0})
	if !errors.Is(err, domain.ErrInvalidUser) {
		t.Fatalf("userFromDO() error = %v, want %v", err, domain.ErrInvalidUser)
	}
}
