package connectrpc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	userv1 "github.com/go-jimu/template/gen/user/private/v1"
	"github.com/go-jimu/template/gen/user/private/v1/userv1connect"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/application/query"
	"github.com/go-jimu/template/internal/business/user/domain"
	userconnectrpc "github.com/go-jimu/template/internal/business/user/transport/connectrpc"
)

type queryRepositoryStub struct {
	user   query.User
	err    error
	page   query.UserPage
	filter query.ListFilter
}

func (r *queryRepositoryStub) Get(context.Context, string) (query.User, error) {
	return r.user, r.err
}

func (r *queryRepositoryStub) List(_ context.Context, filter query.ListFilter) (query.UserPage, error) {
	r.filter = filter
	return r.page, nil
}

func TestGetUsesGeneratedConnectContract(t *testing.T) {
	repository := &queryRepositoryStub{user: query.User{
		ID: "0197e113-98c8-7c96-a6cd-1d1fc868825a", Name: "Alice", Email: "alice@example.com",
	}}
	client, closeServer := newUserClient(t, repository)
	defer closeServer()

	response, err := client.Get(
		context.Background(),
		&userv1.GetRequest{Id: repository.user.ID},
	)
	if err != nil {
		t.Fatal(err)
	}
	if response.GetUser().GetEmail() != repository.user.Email {
		t.Fatalf("response user = %#v", response.GetUser())
	}
}

func TestGetMapsNotFoundToConnectCode(t *testing.T) {
	client, closeServer := newUserClient(t, &queryRepositoryStub{err: domain.ErrUserNotFound})
	defer closeServer()

	_, err := client.Get(
		context.Background(),
		&userv1.GetRequest{Id: "missing-user"},
	)
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("Connect code = %s, want %s (error: %v)", connect.CodeOf(err), connect.CodeNotFound, err)
	}
}

func TestListRoundTripsPageToken(t *testing.T) {
	cursor := "0197e113-98c8-7c96-a6cd-1d1fc868825a"
	repository := &queryRepositoryStub{page: query.UserPage{NextCursor: cursor}}
	client, closeServer := newUserClient(t, repository)
	defer closeServer()

	first, err := client.List(context.Background(), &userv1.ListRequest{PageSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	token := first.GetNextPageToken()
	if token == "" || token == cursor {
		t.Fatalf("next_page_token = %q, want an opaque token", token)
	}
	if _, err = client.List(context.Background(), &userv1.ListRequest{PageToken: token}); err != nil {
		t.Fatal(err)
	}
	if repository.filter.After != cursor {
		t.Fatalf("After = %q, want %q", repository.filter.After, cursor)
	}
}

func TestListRejectsMalformedPageToken(t *testing.T) {
	client, closeServer := newUserClient(t, &queryRepositoryStub{})
	defer closeServer()

	_, err := client.List(context.Background(), &userv1.ListRequest{PageToken: "%%%"})
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("Connect code = %s, want %s", connect.CodeOf(err), connect.CodeInvalidArgument)
	}
}

func newUserClient(
	t *testing.T,
	repository query.Repository,
) (userv1connect.UserServiceClient, func()) {
	t.Helper()
	service := userconnectrpc.NewHandler(application.NewApplication(nil, nil, nil), query.NewUsers(repository))
	registry := connect.NewServer()
	userv1connect.RegisterUserServiceHandler(registry, service)
	mux := http.NewServeMux()
	connecthttp.Mount(mux, registry)
	server := httptest.NewServer(mux)
	client := userv1connect.NewUserServiceClient(connect.NewClient(connecthttp.NewTransport(server.Client(), server.URL)))
	return client, server.Close
}
