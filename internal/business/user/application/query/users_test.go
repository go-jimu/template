package query_test

import (
	"context"
	"testing"

	"github.com/go-jimu/template/internal/business/user/application/query"
)

type repositoryFake struct {
	filter query.ListFilter
}

func (*repositoryFake) Get(context.Context, string) (query.User, error) {
	return query.User{}, nil
}

func (r *repositoryFake) List(_ context.Context, filter query.ListFilter) (query.UserPage, error) {
	r.filter = filter
	return query.UserPage{}, nil
}

func TestListNormalizesPageSize(t *testing.T) {
	for _, tc := range []struct {
		pageSize, wantLimit int
	}{{0, 20}, {-1, 20}, {7, 7}, {500, 100}} {
		repository := &repositoryFake{}

		if _, err := query.NewUsers(repository).List(context.Background(), query.ListUsers{
			NamePrefix: "Al", PageSize: tc.pageSize, After: "cursor",
		}); err != nil {
			t.Fatal(err)
		}
		want := query.ListFilter{NamePrefix: "Al", Limit: tc.wantLimit, After: "cursor"}
		if repository.filter != want {
			t.Fatalf("page size %d: filter = %#v, want %#v", tc.pageSize, repository.filter, want)
		}
	}
}
