package connectrpc_test

import (
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	sharedconnectrpc "github.com/go-jimu/template/internal/pkg/connectrpc"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestServerStartsHealthEndpointAndShutsDown(t *testing.T) {
	var server sharedconnectrpc.Server
	app := fxtest.New(
		t,
		fx.Supply(
			sharedconnectrpc.Option{Addr: "127.0.0.1:0"},
			slog.New(slog.NewTextHandler(io.Discard, nil)),
		),
		fx.Provide(sharedconnectrpc.NewServer),
		fx.Invoke(func(lifecycle fx.Lifecycle, hook sharedconnectrpc.ServeHook) {
			lifecycle.Append(hook.Hook)
		}),
		fx.Invoke(func(server sharedconnectrpc.Server) {
			server.Register("/test.Service/", http.HandlerFunc(
				func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(http.StatusNoContent)
				},
			))
		}),
		fx.Populate(&server),
	)
	app.RequireStart()

	client := &http.Client{Timeout: time.Second}
	for _, probe := range []string{"/healthz", "/readyz"} {
		response, err := client.Get("http://" + server.Address() + probe)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", probe, response.StatusCode)
		}
	}
	response, err := client.Get("http://" + server.Address() + "/test.Service/Call")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("registered handler status = %d, want 204", response.StatusCode)
	}

	app.RequireStop()
	transport := &http.Transport{DisableKeepAlives: true}
	stoppedClient := &http.Client{Transport: transport, Timeout: time.Second}
	t.Cleanup(transport.CloseIdleConnections)
	if _, err = stoppedClient.Get("http://" + server.Address() + "/healthz"); err == nil {
		t.Fatal("request succeeded after server shutdown")
	}
}

// Intent: routing keeps the contract generated Connect handlers and probes
// rely on: a service is mounted on its path prefix with or without the
// trailing slash, probes answer GET only, and other paths are not found.
func TestServerRouting(t *testing.T) {
	var server sharedconnectrpc.Server
	app := fxtest.New(
		t,
		fx.Supply(
			sharedconnectrpc.Option{Addr: "127.0.0.1:0"},
			slog.New(slog.NewTextHandler(io.Discard, nil)),
		),
		fx.Provide(sharedconnectrpc.NewServer),
		fx.Invoke(func(lifecycle fx.Lifecycle, hook sharedconnectrpc.ServeHook) {
			lifecycle.Append(hook.Hook)
		}),
		fx.Invoke(func(server sharedconnectrpc.Server) {
			for pattern, status := range map[string]int{
				"/with.Slash/":   http.StatusNoContent,
				"/without.Slash": http.StatusAccepted,
			} {
				server.Register(pattern, http.HandlerFunc(
					func(response http.ResponseWriter, _ *http.Request) { response.WriteHeader(status) },
				))
			}
		}),
		fx.Populate(&server),
	)
	app.RequireStart()
	defer app.RequireStop()

	client := &http.Client{Timeout: time.Second}
	tests := []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/with.Slash/Call", http.StatusNoContent},
		{http.MethodPost, "/without.Slash/Call", http.StatusAccepted},
		{http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
		{http.MethodPost, "/readyz", http.StatusMethodNotAllowed},
		{http.MethodGet, "/unknown.Service/Call", http.StatusNotFound},
	}
	for _, test := range tests {
		request, err := http.NewRequest(test.method, "http://"+server.Address()+test.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != test.want {
			t.Errorf("%s %s status = %d, want %d", test.method, test.path, response.StatusCode, test.want)
		}
	}
}
