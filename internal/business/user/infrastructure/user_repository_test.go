package infrastructure

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-jimu/template/internal/business/user/domain"
	"xorm.io/xorm"
	"xorm.io/xorm/core"
)

func TestUserRepositoryInsertsNewAggregateWithStoredVersion(t *testing.T) {
	repository, mock := newMockRepository(t)
	user, err := domain.NewUser("Alice", "current-pass", "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("INSERT INTO .*user_account.*version.*VALUES").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err = repository.Save(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepositoryUpdateUsesVersionPredicateAndIncrement(t *testing.T) {
	repository, mock := newMockRepository(t)
	user := rehydratedUser(t, 3)
	mock.ExpectExec("UPDATE .*user_account.*version.*version.*id.*version.*deleted_at").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repository.Save(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepositoryReportsConcurrentModification(t *testing.T) {
	repository, mock := newMockRepository(t)
	user := rehydratedUser(t, 3)
	mock.ExpectExec("UPDATE .*user_account.*version.*version.*id.*version.*deleted_at").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repository.Save(context.Background(), user)
	if !errors.Is(err, domain.ErrConcurrentModification) {
		t.Fatalf("Save() error = %v, want %v", err, domain.ErrConcurrentModification)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func newMockRepository(t *testing.T) (domain.Repository, sqlmock.Sqlmock) {
	t.Helper()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	engine, err := xorm.NewEngineWithDB("mysql", "sqlmock", core.FromDB(database))
	if err != nil {
		t.Fatal(err)
	}
	return NewUserRepository(engine), mock
}

func rehydratedUser(t *testing.T, version int) *domain.User {
	t.Helper()
	user, err := domain.NewUser("Alice", "current-pass", "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	user.Version = version
	return user
}
