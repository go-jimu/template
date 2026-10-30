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
