//go:build integration

package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-jimu/template/internal/business/user/application/query"
	"github.com/go-jimu/template/internal/business/user/domain"
	_ "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"xorm.io/xorm"
)

// newMySQLEngine starts MySQL with the repository migrations applied, so the
// tests run against the schema production uses.
func newMySQLEngine(t *testing.T) *xorm.Engine {
	t.Helper()
	ctx := context.Background()
	container, err := mysql.Run(ctx, "mysql:8.4",
		mysql.WithDatabase("jimu"),
		mysql.WithUsername("jimu"),
		mysql.WithPassword("jimu"),
		mysql.WithScripts("../../../../migrations/001_create_user_account.sql"),
	)
	testcontainers.CleanupContainer(t, container)
	if err != nil {
		t.Fatal(err)
	}
	dsn, err := container.ConnectionString(ctx, "charset=utf8mb4", "parseTime=true", "loc=UTC")
	if err != nil {
		t.Fatal(err)
	}
	engine, err := xorm.NewEngine("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	return engine
}

func TestMySQL(t *testing.T) {
	engine := newMySQLEngine(t)
	repository := NewUserRepository(engine)
	queries := NewUserQueryRepository(engine)

	t.Run("repository round-trips and locks optimistically", func(t *testing.T) {
		ctx := context.Background()
		user, err := domain.NewUser("Round Trip", "current-pass", "round@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if err = repository.Save(ctx, user); err != nil {
			t.Fatal(err)
		}
		loaded, err := repository.Get(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Version != 1 || loaded.Email != user.Email {
			t.Fatalf("loaded User = %+v", loaded)
		}
		stale, err := repository.Get(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		if err = loaded.ChangePassword("current-pass", "next-pass-1"); err != nil {
			t.Fatal(err)
		}
		if err = repository.Save(ctx, loaded); err != nil {
			t.Fatal(err)
		}
		if err = stale.ChangePassword("current-pass", "next-pass-2"); err != nil {
			t.Fatal(err)
		}
		if err = repository.Save(ctx, stale); !errors.Is(err, domain.ErrConcurrentModification) {
			t.Fatalf("stale Save() error = %v, want %v", err, domain.ErrConcurrentModification)
		}
		if _, err = repository.Get(ctx, "missing"); !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("Get(missing) error = %v, want %v", err, domain.ErrUserNotFound)
		}
	})

	t.Run("query lists keyset pages newest first", func(t *testing.T) {
		ctx := context.Background()
		created := make([]string, 0, 5)
		for index := range 5 {
			user, err := domain.NewUser(fmt.Sprintf("Page %d", index), "current-pass", fmt.Sprintf("page%d@example.com", index))
			if err != nil {
				t.Fatal(err)
			}
			if err = repository.Save(ctx, user); err != nil {
				t.Fatal(err)
			}
			created = append(created, user.ID)
		}

		var listed []string
		filter := query.ListFilter{NamePrefix: "Page", Limit: 2}
		for {
			page, err := queries.List(ctx, filter)
			if err != nil {
				t.Fatal(err)
			}
			for _, user := range page.Users {
				listed = append(listed, user.ID)
			}
			if page.NextCursor == "" {
				break
			}
			filter.After = page.NextCursor
		}
		if len(listed) != len(created) {
			t.Fatalf("listed %d Users, want %d", len(listed), len(created))
		}
		for index := range created {
			if listed[index] != created[len(created)-1-index] {
				t.Fatalf("listed %v, want newest first of %v", listed, created)
			}
		}

		got, err := queries.Get(ctx, created[0])
		if err != nil || got.Name != "Page 0" {
			t.Fatalf("Get() = %+v, %v", got, err)
		}
	})

	t.Run("list query reads an index in order", func(t *testing.T) {
		rows, err := engine.QueryString(
			"EXPLAIN SELECT id, name, email FROM user_account " +
				"WHERE deleted_at = 0 AND id < 'ffffffff' ORDER BY id DESC LIMIT 21",
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0]["key"] == "" || strings.Contains(rows[0]["Extra"], "filesort") {
			t.Fatalf("EXPLAIN = %v, want an index scan without filesort", rows)
		}
	})
}
