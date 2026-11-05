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

type servers struct {
	private sharedconnectrpc.PrivateServer
	public  sharedconnectrpc.PublicServer
}

// startServers starts both listeners on random ports with register applied
// before start, and returns them with their Fx app.
func startServers(t *testing.T, register func(sharedconnectrpc.PrivateServer, sharedconnectrpc.PublicServer)) (*fxtest.App, servers) {
	t.Helper()
	var s servers
	app := fxtest.New(
		t,
		fx.Supply(
			sharedconnectrpc.Option{Addr: "127.0.0.1:0", PublicAddr: "127.0.0.1:0"},
			slog.New(slog.NewTextHandler(io.Discard, nil)),
		),
		fx.Provide(sharedconnectrpc.NewServer),
		fx.Invoke(func(lifecycle fx.Lifecycle, hook sharedconnectrpc.ServeHook) {
			lifecycle.Append(hook.Hook)
		}),
		fx.Invoke(register),
		fx.Populate(&s.private, &s.public),
	)
	app.RequireStart()
	return app, s
}

func status(t *testing.T, method, url string) int {
	t.Helper()
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := (&http.Client{Timeout: time.Second}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	return response.StatusCode
}

func handlerWithStatus(code int) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) { response.WriteHeader(code) })
}

// Intent: private and public contracts listen on separate ports, so exposing
// the public port for web and app clients never exposes a private contract.
// Each port answers the probes and only its own services.
func TestPrivateAndPublicListenSeparately(t *testing.T) {
	app, s := startServers(t, func(private sharedconnectrpc.PrivateServer, public sharedconnectrpc.PublicServer) {
		private.Register("/test.private.v1.Service/", handlerWithStatus(http.StatusNoContent))
		public.Register("/test.public.v1.Service/", handlerWithStatus(http.StatusAccepted))
	})
	defer app.RequireStop()

	privateURL, publicURL := "http://"+s.private.Address(), "http://"+s.public.Address()
	if privateURL == publicURL {
		t.Fatalf("both listen on %s", privateURL)
	}
	tests := []struct {
		name, url string
		want      int
	}{
		{"private service on private port", privateURL + "/test.private.v1.Service/Call", http.StatusNoContent},
		{"public service on public port", publicURL + "/test.public.v1.Service/Call", http.StatusAccepted},
		{"private service on public port", publicURL + "/test.private.v1.Service/Call", http.StatusNotFound},
		{"public service on private port", privateURL + "/test.public.v1.Service/Call", http.StatusNotFound},
	}
	for _, test := range tests {
		if got := status(t, http.MethodPost, test.url); got != test.want {
			t.Errorf("%s: status = %d, want %d", test.name, got, test.want)
		}
	}
	for _, base := range []string{privateURL, publicURL} {
		for _, probe := range []string{"/healthz", "/readyz"} {
			if got := status(t, http.MethodGet, base+probe); got != http.StatusOK {
				t.Errorf("GET %s%s = %d, want 200", base, probe, got)
			}
		}
	}
}

// Intent: routing keeps the contract generated Connect handlers and probes
// rely on: a service is mounted on its path prefix with or without the
// trailing slash, probes answer GET only, and other paths are not found.
func TestServerRouting(t *testing.T) {
	app, s := startServers(t, func(private sharedconnectrpc.PrivateServer, _ sharedconnectrpc.PublicServer) {
		private.Register("/with.Slash/", handlerWithStatus(http.StatusNoContent))
		private.Register("/without.Slash", handlerWithStatus(http.StatusAccepted))
	})
	defer app.RequireStop()

	base := "http://" + s.private.Address()
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
		if got := status(t, test.method, base+test.path); got != test.want {
			t.Errorf("%s %s status = %d, want %d", test.method, test.path, got, test.want)
		}
	}
}

// Intent: stopping takes both listeners down.
func TestServersShutDown(t *testing.T) {
	app, s := startServers(t, func(sharedconnectrpc.PrivateServer, sharedconnectrpc.PublicServer) {})
	app.RequireStop()

	transport := &http.Transport{DisableKeepAlives: true}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: time.Second}
	for _, address := range []string{s.private.Address(), s.public.Address()} {
		if _, err := client.Get("http://" + address + "/healthz"); err == nil {
			t.Errorf("%s still serves after shutdown", address)
		}
	}
}
