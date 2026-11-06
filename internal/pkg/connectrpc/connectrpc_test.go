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
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/grpchealth/v2"
	sharedconnectrpc "github.com/go-jimu/template/internal/pkg/connectrpc"
	"github.com/go-jimu/template/internal/pkg/requestid"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"google.golang.org/protobuf/types/known/emptypb"
)

// startServer starts the server on a random port with register applied
// before start, and returns it with its Fx app.
func startServer(t *testing.T, register func(sharedconnectrpc.Server)) (*fxtest.App, sharedconnectrpc.Server) {
	t.Helper()
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
		fx.Invoke(register),
		fx.Populate(&server),
	)
	app.RequireStart()
	return app, server
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

// health returns the status a grpc.health.v1 Check reports for service; the
// empty service is readiness. client defaults to an HTTP/1.1 Connect client.
func health(t *testing.T, base, service string, options ...connecthttp.Option) grpchealth.Status {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client := grpchealth.NewClient(connect.NewClient(connecthttp.NewTransport(http.DefaultClient, base, options...)))
	response, err := client.Check(ctx, &grpchealth.CheckRequest{Service: service})
	if err != nil {
		t.Fatalf("health check %q: %v", service, err)
	}
	return response.Status
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

// Intent: public and private are contracts, not listeners: one listener
// serves every registered contract and the probes, and the gateway's routes
// decide which contracts leave the deployment.
func TestOneListenerServesEveryContract(t *testing.T) {
	app, server := startServer(t, func(server sharedconnectrpc.Server) {
		register(server, "/test.private.v1.Service/Call")
		register(server, "/test.public.v1.Service/Call")
	})
	defer app.RequireStop()

	base := "http://" + server.Address()
	for _, procedure := range []string{"/test.private.v1.Service/Call", "/test.public.v1.Service/Call"} {
		if got := status(t, http.MethodPost, base+procedure); got != http.StatusOK {
			t.Errorf("POST %s = %d, want 200", procedure, got)
		}
	}
	for _, service := range []string{"", sharedconnectrpc.LivenessService} {
		if got := health(t, base, service); got != grpchealth.StatusServing {
			t.Errorf("health %q = %v, want serving", service, got)
		}
	}
}

// Intent: Kubernetes gRPC probes and gRPC clients speak HTTP/2 without TLS
// and without an upgrade (prior knowledge), so the listener serves h2c.
func TestServerSpeaksGRPCOverH2C(t *testing.T) {
	app, server := startServer(t, func(sharedconnectrpc.Server) {})
	defer app.RequireStop()

	var protocols http.Protocols
	protocols.SetUnencryptedHTTP2(true)
	h2c := &http.Client{Transport: &http.Transport{Protocols: &protocols}}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client := grpchealth.NewClient(connect.NewClient(
		connecthttp.NewTransport(h2c, "http://"+server.Address(), connecthttp.WithGRPC())))
	response, err := client.Check(ctx, &grpchealth.CheckRequest{})
	if err != nil {
		t.Fatalf("gRPC health check over h2c: %v", err)
	}
	if response.Status != grpchealth.StatusServing {
		t.Fatalf("status = %v, want serving", response.Status)
	}
}

// Intent: a server routes the procedures registered before start, an
// unknown method of a registered service is Connect's unimplemented (501),
// and other paths are not found.
func TestServerRouting(t *testing.T) {
	app, server := startServer(t, func(server sharedconnectrpc.Server) {
		register(server, "/test.v1.Service/Call")
	})
	defer app.RequireStop()

	base := "http://" + server.Address()
	tests := []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/test.v1.Service/Call", http.StatusOK},
		{http.MethodPost, "/test.v1.Service/Unknown", http.StatusNotImplemented},
		{http.MethodPost, "/unknown.v1.Service/Call", http.StatusNotFound},
		{http.MethodGet, "/healthz", http.StatusNotFound},
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
func TestServerRunsTheSharedInterceptor(t *testing.T) {
	var logs syncBuffer
	out, err := sharedconnectrpc.NewServer(nil, sharedconnectrpc.Option{Addr: "127.0.0.1:0"},
		requestid.Logger(slog.New(slog.NewJSONHandler(&logs, nil))))
	if err != nil {
		t.Fatal(err)
	}
	register(out.Server, "/test.private.v1.Service/Call")
	register(out.Server, "/test.public.v1.Service/Call")
	if err = out.ServeHook.OnStart(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, procedure := range []string{"/test.private.v1.Service/Call", "/test.public.v1.Service/Call"} {
		url := "http://" + out.Server.Address() + procedure
		if got := send(t, http.MethodPost, url, http.Header{"X-Request-Id": {"caller-request"}}); got != http.StatusOK {
			t.Fatalf("POST %s = %d, want 200", url, got)
		}
	}
	if err = out.ServeHook.OnStop(context.Background()); err != nil {
		t.Fatal(err)
	}

	for _, procedure := range []string{"/test.private.v1.Service/Call", "/test.public.v1.Service/Call"} {
		var found bool
		for _, line := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
			if strings.Contains(line, `"operation":"`+procedure+`"`) {
				found = strings.Contains(line, `"request_id":"caller-request"`)
				break
			}
		}
		if !found {
			t.Errorf("no completion log of %s with request_id caller-request:\n%s", procedure, logs.String())
		}
	}
}

// Intent: stopping takes the listener down.
func TestServerShutsDown(t *testing.T) {
	app, server := startServer(t, func(sharedconnectrpc.Server) {})
	app.RequireStop()

	transport := &http.Transport{DisableKeepAlives: true}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: time.Second}
	if _, err := client.Get("http://" + server.Address() + "/"); err == nil {
		t.Errorf("%s still serves after shutdown", server.Address())
	}
}

// startDirect starts the server without Fx, so a test can call the stop
// hook with its own ctx.
func startDirect(t *testing.T, delay string) (sharedconnectrpc.ServeOut, func(context.Context) error) {
	t.Helper()
	out, err := sharedconnectrpc.NewServer(nil, sharedconnectrpc.Option{
		Addr: "127.0.0.1:0", ShutdownDelay: delay,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	register(out.Server, "/test.public.v1.Service/Call")
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

// Intent: on stop, the server reports not ready for shutdown-delay while
// still serving requests, so load balancers stop routing before the listener
// closes and no client meets a refused connection.
func TestShutdownDelayKeepsServingWhileNotReady(t *testing.T) {
	const delay = 400 * time.Millisecond
	out, stop := startDirect(t, delay.String())
	base := "http://" + out.Server.Address()

	started := time.Now()
	done := make(chan error, 1)
	go func() { done <- stop(context.Background()) }()

	deadline := time.Now().Add(delay / 2)
	for health(t, base, "") != grpchealth.StatusNotServing {
		if time.Now().After(deadline) {
			t.Fatal("readiness still serving after stop began")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := health(t, base, sharedconnectrpc.LivenessService); got != grpchealth.StatusServing {
		t.Errorf("liveness = %v during the delay, want serving: the process is alive", got)
	}
	if got := status(t, http.MethodPost, base+"/test.public.v1.Service/Call"); got != http.StatusOK {
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
