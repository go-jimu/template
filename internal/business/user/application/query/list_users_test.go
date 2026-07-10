package query_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/user/application/query"
	"github.com/samber/oops"
)

type queryRepositoryFake struct {
	filter  query.ListFilter
	getErr  error
	listErr error
}

func (r *queryRepositoryFake) Get(context.Context, string) (query.User, error) {
	return query.User{}, r.getErr
}

func (r *queryRepositoryFake) List(_ context.Context, filter query.ListFilter) (query.UserPage, error) {
	r.filter = filter
	return query.UserPage{Page: filter.Page, PageSize: filter.PageSize}, r.listErr
}

func TestListUsersNormalizesPagination(t *testing.T) {
	repository := new(queryRepositoryFake)
	handler := query.NewListUsersHandler(repository)
	var logs bytes.Buffer
	ctx := sloghelper.NewContext(
		context.Background(),
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)

	page, err := handler.Handle(ctx, query.ListUsers{
		NamePrefix: "Al",
		Page:       0,
		PageSize:   1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repository.filter.Page != 1 || repository.filter.PageSize != 100 {
		t.Fatalf("normalized filter = %#v", repository.filter)
	}
	if page.Page != 1 || page.PageSize != 100 {
		t.Fatalf("page = %#v", page)
	}
	for _, field := range []string{
		`"operation":"user.list"`,
		`"outcome":"success"`,
		`"duration_ms":`,
		`"page":1`,
		`"page_size":100`,
	} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("completion log %q does not contain %s", logs.String(), field)
		}
	}
}

func TestGetUserLogsFailure(t *testing.T) {
	repositoryErr := errors.New("read store unavailable")
	handler := query.NewGetUserHandler(&queryRepositoryFake{getErr: repositoryErr})
	var logs bytes.Buffer
	ctx := sloghelper.NewContext(
		context.Background(),
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)

	_, err := handler.Handle(ctx, "user-1")
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("Handle() error = %v, want %v", err, repositoryErr)
	}
	enhanced, ok := oops.AsOops(err)
	if !ok || enhanced.Context()["user_id"] != "user-1" {
		t.Fatalf("error context = %#v, want user_id user-1", enhanced.Context())
	}
	for _, field := range []string{
		`"operation":"user.get"`,
		`"outcome":"failed"`,
		`"duration_ms":`,
		`"user_id":"user-1"`,
	} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("completion log %q does not contain %s", logs.String(), field)
		}
	}
	if strings.Contains(logs.String(), `"error":`) {
		t.Fatalf("inner Application log contains error details: %q", logs.String())
	}
}
