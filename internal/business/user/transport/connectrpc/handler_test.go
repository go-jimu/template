package connectrpc_test

import (
	"context"
	"net/http/httptest"
	"testing"

	connect "connectrpc.com/connect"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/application/query"
	"github.com/go-jimu/template/internal/business/user/domain"
	userconnectrpc "github.com/go-jimu/template/internal/business/user/transport/connectrpc"
	userv1 "github.com/go-jimu/template/pkg/gen/user/v1"
	"github.com/go-jimu/template/pkg/gen/user/v1/userv1connect"
)

type queryRepositoryStub struct {
	user query.User
	err  error
}

func (r *queryRepositoryStub) Get(context.Context, string) (query.User, error) {
	return r.user, r.err
}

func (*queryRepositoryStub) List(context.Context, query.ListFilter) (query.UserPage, error) {
	return query.UserPage{}, nil
}

func TestGetUsesGeneratedConnectContract(t *testing.T) {
	repository := &queryRepositoryStub{user: query.User{
		ID: "0197e113-98c8-7c96-a6cd-1d1fc868825a", Name: "Alice", Email: "alice@example.com",
	}}
	client, closeServer := newUserClient(t, repository)
	defer closeServer()

	response, err := client.Get(
		context.Background(),
		connect.NewRequest(&userv1.GetRequest{Id: repository.user.ID}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetUser().GetEmail() != repository.user.Email {
		t.Fatalf("response user = %#v", response.Msg.GetUser())
	}
}

func TestGetMapsNotFoundToConnectCode(t *testing.T) {
	client, closeServer := newUserClient(t, &queryRepositoryStub{err: domain.ErrUserNotFound})
	defer closeServer()

	_, err := client.Get(
		context.Background(),
		connect.NewRequest(&userv1.GetRequest{Id: "missing-user"}),
	)
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("Connect code = %s, want %s (error: %v)", connect.CodeOf(err), connect.CodeNotFound, err)
	}
}

func newUserClient(
	t *testing.T,
	repository query.Repository,
) (userv1connect.UserServiceClient, func()) {
	t.Helper()
	app := &application.Application{
		Queries: application.Queries{Get: query.NewGetUserHandler(repository)},
	}
	service := userconnectrpc.NewHandler(app)
	_, handler := userv1connect.NewUserServiceHandler(service)
	server := httptest.NewServer(handler)
	client := userv1connect.NewUserServiceClient(server.Client(), server.URL)
	return client, server.Close
}
