package httpsrv

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-jimu/components/sloghelper"
)

type logEntry struct {
	log *slog.Logger
	req *http.Request
}

func newLogEntry(log *slog.Logger, r *http.Request) middleware.LogEntry {
	return &logEntry{
		log: log,
		req: r,
	}
}

func (le *logEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	le.log.InfoContext(le.req.Context(), "request complete",
		slog.String("client_ip", le.req.RemoteAddr),
		slog.Group("request", slog.String("method", le.req.Method), slog.String("path", le.req.URL.Path), slog.String("query", le.req.URL.RawQuery)),
		slog.Group("response", slog.Int("status_code", status), slog.Int("bytes_length", bytes)),
		slog.String("elapsed", elapsed.String()),
	)
}

func (le *logEntry) Panic(v interface{}, stack []byte) {
	le.log.ErrorContext(le.req.Context(), "broken request",
		slog.String("stack", string(stack)), slog.String("panic", fmt.Sprintf("%+v", v)))
}

func CarryLog() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(sloghelper.NewContext(r.Context(), slog.Default())))
		}
		return http.HandlerFunc(fn)
	}
}

func RequestLog(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		entry := newLogEntry(sloghelper.FromContext(r.Context()), r)
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			entry.Write(ww.Status(), ww.BytesWritten(), ww.Header(), time.Since(start), nil)
		}()

		next.ServeHTTP(ww, middleware.WithLogEntry(r, entry))
	}
	return http.HandlerFunc(fn)
}

func RecordRequestID(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := sloghelper.FromContext(ctx)
		logger = logger.With(slog.String("request_id", middleware.GetReqID(r.Context())))
		ctx = sloghelper.NewContext(ctx, logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}
