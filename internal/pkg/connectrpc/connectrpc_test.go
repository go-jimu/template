package connectrpc_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	sharedconnectrpc "github.com/go-jimu/template/internal/pkg/connectrpc"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"google.golang.org/protobuf/types/known/emptypb"
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
	return send(t, method, url, nil)
}

// send makes one request; a POST is a Connect unary call with an empty JSON
// message.
func send(t *testing.T, method, url string, header http.Header) int {
	t.Helper()
	var body io.Reader
	if method == http.MethodPost {
		body = strings.NewReader("{}")
	}
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	if method == http.MethodPost {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, values := range header {
		request.Header[key] = values
	}
	response, err := (&http.Client{Timeout: time.Second}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	return response.StatusCode
}

// register adds a unary procedure, such as "/test.v1.Service/Call", that
// answers an empty message.
func register(server sharedconnectrpc.Server, procedure string) {
	server.Registry().Register(connect.Method{
		Spec: connect.Spec{StreamType: connect.StreamTypeUnary, Procedure: procedure},
		Handler: func(_ context.Context, _ connect.Spec, stream connect.ServerStream) error {
			if err := stream.Receive(&emptypb.Empty{}); err != nil {
				return err
			}
			return stream.Send(&emptypb.Empty{})
		},
	})
}

// Intent: private and public contracts listen on separate ports, so exposing
// the public port for web and app clients never exposes a private contract.
// Each port answers the probes and only its own services.
func TestPrivateAndPublicListenSeparately(t *testing.T) {
	app, s := startServers(t, func(private sharedconnectrpc.PrivateServer, public sharedconnectrpc.PublicServer) {
		register(private, "/test.private.v1.Service/Call")
		register(public, "/test.public.v1.Service/Call")
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
		{"private service on private port", privateURL + "/test.private.v1.Service/Call", http.StatusOK},
		{"public service on public port", publicURL + "/test.public.v1.Service/Call", http.StatusOK},
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

// Intent: a server routes the procedures registered before start, an
// unknown method of a registered service is Connect's unimplemented (501),
// probes answer GET only, and other paths are not found.
func TestServerRouting(t *testing.T) {
	app, s := startServers(t, func(private sharedconnectrpc.PrivateServer, _ sharedconnectrpc.PublicServer) {
		register(private, "/test.v1.Service/Call")
	})
	defer app.RequireStop()

	base := "http://" + s.private.Address()
	tests := []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/test.v1.Service/Call", http.StatusOK},
		{http.MethodPost, "/test.v1.Service/Unknown", http.StatusNotImplemented},
		{http.MethodPost, "/unknown.v1.Service/Call", http.StatusNotFound},
		{http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
		{http.MethodPost, "/readyz", http.StatusMethodNotAllowed},
	}
	for _, test := range tests {
		if got := status(t, test.method, base+test.path); got != test.want {
			t.Errorf("%s %s status = %d, want %d", test.method, test.path, got, test.want)
		}
	}
}

// syncBuffer is a log destination safe to write from server goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Intent: every RPC runs through the shared interceptor, which logs it under
// the caller's X-Request-ID, so a request is traceable across services.
func TestServersRunTheSharedInterceptor(t *testing.T) {
	var logs syncBuffer
	out, err := sharedconnectrpc.NewServer(nil, sharedconnectrpc.Option{
		Addr: "127.0.0.1:0", PublicAddr: "127.0.0.1:0",
	}, slog.New(slog.NewJSONHandler(&logs, nil)))
	if err != nil {
		t.Fatal(err)
	}
	register(out.Private, "/test.private.v1.Service/Call")
	register(out.Public, "/test.public.v1.Service/Call")
	if err = out.ServeHook.OnStart(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{
		"http://" + out.Private.Address() + "/test.private.v1.Service/Call",
		"http://" + out.Public.Address() + "/test.public.v1.Service/Call",
	} {
		if got := send(t, http.MethodPost, url, http.Header{"X-Request-Id": {"caller-request"}}); got != http.StatusOK {
			t.Fatalf("POST %s = %d, want 200", url, got)
		}
	}
	if err = out.ServeHook.OnStop(context.Background()); err != nil {
		t.Fatal(err)
	}

	for _, procedure := range []string{"/test.private.v1.Service/Call", "/test.public.v1.Service/Call"} {
		want := `"request_id":"caller-request","operation":"` + procedure + `"`
		if !strings.Contains(logs.String(), want) {
			t.Errorf("logs do not contain %s:\n%s", want, logs.String())
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

// startDirect starts the servers without Fx, so a test can call the stop
// hook with its own ctx.
func startDirect(t *testing.T, delay string) (sharedconnectrpc.ServeOut, func(context.Context) error) {
	t.Helper()
	out, err := sharedconnectrpc.NewServer(nil, sharedconnectrpc.Option{
		Addr: "127.0.0.1:0", PublicAddr: "127.0.0.1:0", ShutdownDelay: delay,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	register(out.Public, "/test.public.v1.Service/Call")
	if err = out.ServeHook.OnStart(context.Background()); err != nil {
		t.Fatal(err)
	}
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			_ = out.ServeHook.OnStop(context.Background())
		}
	})
	return out, func(ctx context.Context) error {
		stopped = true
		return out.ServeHook.OnStop(ctx)
	}
}

// Intent: on stop, both ports report not ready for shutdown-delay while still
// serving requests, so load balancers stop routing before the listeners
// close and no client meets a refused connection.
func TestShutdownDelayKeepsServingWhileNotReady(t *testing.T) {
	const delay = 400 * time.Millisecond
	out, stop := startDirect(t, delay.String())
	privateURL, publicURL := "http://"+out.Private.Address(), "http://"+out.Public.Address()

	started := time.Now()
	done := make(chan error, 1)
	go func() { done <- stop(context.Background()) }()

	deadline := time.Now().Add(delay / 2)
	for status(t, http.MethodGet, publicURL+"/readyz") != http.StatusServiceUnavailable {
		if time.Now().After(deadline) {
			t.Fatal("/readyz still ready after stop began")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := status(t, http.MethodGet, privateURL+"/readyz"); got != http.StatusServiceUnavailable {
		t.Errorf("private /readyz = %d during the delay, want 503", got)
	}
	if got := status(t, http.MethodPost, publicURL+"/test.public.v1.Service/Call"); got != http.StatusOK {
		t.Errorf("request during the delay = %d, want 200: the listener must still serve", got)
	}

	if err := <-done; err != nil {
		t.Fatalf("stop error = %v", err)
	}
	if elapsed := time.Since(started); elapsed < delay {
		t.Errorf("stop returned after %v, before the %v delay", elapsed, delay)
	}
}

// Intent: the delay counts against the stop deadline: a stop ctx that ends
// first cuts the delay short instead of overrunning Fx's StopTimeout.
func TestShutdownDelayEndsWithStopContext(t *testing.T) {
	_, stop := startDirect(t, "10s")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	started := time.Now()
	_ = stop(ctx)
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("stop took %v with a 100ms ctx and a 10s delay", elapsed)
	}
}
