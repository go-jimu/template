package connectrpc_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	userpublicv1 "github.com/go-jimu/template/gen/user/public/v1"
	"github.com/go-jimu/template/gen/user/public/v1/userpublicv1connect"
	"github.com/go-jimu/template/internal/business/user/application/query"
	"github.com/go-jimu/template/internal/business/user/domain"
	userconnectrpc "github.com/go-jimu/template/internal/business/user/transport/connectrpc"
)

// newProfileServer serves the public ProfileService over the read side and
// records the HTTP method of each request.
func newProfileServer(t *testing.T, repository query.Repository) (*httptest.Server, *[]string) {
	t.Helper()
	methods := &[]string{}
	registry := connect.NewServer()
	userpublicv1connect.RegisterProfileServiceHandler(registry, userconnectrpc.NewProfileHandler(query.NewUsers(repository)))
	mux := http.NewServeMux()
	connecthttp.Mount(mux, registry)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		*methods = append(*methods, request.Method)
		mux.ServeHTTP(response, request)
	}))
	t.Cleanup(server.Close)
	return server, methods
}

var alice = query.User{ID: "0197e113-98c8-7c96-a6cd-1d1fc868825a", Name: "Alice", Email: "alice@example.com"}

// Intent: an app client gets the public profile, id and name only, and the
// no-side-effect RPC travels as HTTP GET when the client asks for it.
func TestGetProfileOverConnectGet(t *testing.T) {
	server, methods := newProfileServer(t, &queryRepositoryStub{user: alice})
	client := userpublicv1connect.NewProfileServiceClient(connect.NewClient(connecthttp.NewTransport(server.Client(), server.URL, connecthttp.WithHTTPGet())))

	response, err := client.GetProfile(context.Background(),
		&userpublicv1.GetProfileRequest{Id: alice.ID})
	if err != nil {
		t.Fatal(err)
	}
	profile := response.GetProfile()
	if profile.GetId() != alice.ID || profile.GetName() != alice.Name {
		t.Fatalf("profile = %v, want id %s and name %s", profile, alice.ID, alice.Name)
	}
	if len(*methods) != 1 || (*methods)[0] != http.MethodGet {
		t.Fatalf("HTTP methods = %v, want one GET", *methods)
	}
}

// Intent: a web page calls the RPC with a plain fetch: GET with the JSON
// message in the query string, and reads a JSON body without the email.
func TestGetProfileFromWebAsJSON(t *testing.T) {
	server, _ := newProfileServer(t, &queryRepositoryStub{user: alice})

	query := url.Values{"encoding": {"json"}, "message": {`{"id":"` + alice.ID + `"}`}}
	response, err := server.Client().Get(server.URL + "/user.public.v1.ProfileService/GetProfile?" + query.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body %s", response.StatusCode, body)
	}
	var got map[string]map[string]any
	if err = json.Unmarshal(body, &got); err != nil {
		t.Fatalf("body %s: %v", body, err)
	}
	want := map[string]any{"id": alice.ID, "name": alice.Name}
	if len(got["profile"]) != len(want) || got["profile"]["id"] != want["id"] || got["profile"]["name"] != want["name"] {
		t.Fatalf("profile = %v, want exactly %v", got["profile"], want)
	}
}

// Intent: public outcomes are Connect codes: a missing id is
// InvalidArgument and an unknown user NotFound, each with a message the
// client can act on. Any other failure is Internal with a fixed message, so
// its cause never reaches the client.
func TestGetProfileErrors(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		err         error
		wantCode    connect.Code
		wantMessage string
	}{
		{"missing id", "", nil, connect.CodeInvalidArgument, "id is required"},
		{"unknown user", "missing-user", domain.ErrUserNotFound, connect.CodeNotFound, "user not found"},
		{"server fault", "user-1", errors.New("database password rejected"), connect.CodeInternal, "internal server error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, _ := newProfileServer(t, &queryRepositoryStub{err: test.err})
			client := userpublicv1connect.NewProfileServiceClient(connect.NewClient(connecthttp.NewTransport(server.Client(), server.URL)))
			_, err := client.GetProfile(context.Background(),
				&userpublicv1.GetProfileRequest{Id: test.id})
			if connect.CodeOf(err) != test.wantCode {
				t.Fatalf("code = %v (%v), want %v", connect.CodeOf(err), err, test.wantCode)
			}
			var connectErr *connect.Error
			if !errors.As(err, &connectErr) || connectErr.Message() != test.wantMessage {
				t.Fatalf("error = %v, want message %q", err, test.wantMessage)
			}
		})
	}
}
