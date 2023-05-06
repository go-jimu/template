package httpsrv

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-jimu/template/internal/pkg/log"
	"golang.org/x/exp/slog"
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
	le.log.InfoCtx(le.req.Context(), "request complete",
		slog.String("client_ip", le.req.RemoteAddr),
		slog.Group("request", slog.String("method", le.req.Method), slog.String("path", le.req.URL.Path), slog.String("query", le.req.URL.RawQuery)),
		slog.Group("response", slog.Int("status_code", status), slog.Int("bytes_lenght", bytes)),
		slog.String("elapsed", elapsed.String()),
	)
}

func (le *logEntry) Panic(v interface{}, stack []byte) {
	le.log.ErrorCtx(le.req.Context(), "broken request", slog.Any("panic", v))
}

func CarryLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(log.InContext(r.Context(), logger)))
		}
		return http.HandlerFunc(fn)
	}
}

func RequestLog(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		entry := newLogEntry(log.FromContext(r.Context()), r)
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			entry.Write(ww.Status(), ww.BytesWritten(), ww.Header(), time.Since(start), nil)
		}()

		next.ServeHTTP(ww, middleware.WithLogEntry(r, entry))
	}
	return http.HandlerFunc(fn)
}

// func InjectContext(next http.Handler) http.Handler {
// 	fn := func(w http.ResponseWriter, r *http.Request) {
// 		ctx, cancel := context.GenDefaultContext()
// 		defer cancel()
// 		mc, mcCancel := context.MergeContext(r.Context(), ctx)
// 		defer mcCancel()

// 		next.ServeHTTP(w, r.WithContext(mc))
// 	}

// 	return http.HandlerFunc(fn)
// }

func RecordRequestID(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := log.FromContext(ctx)
		logger = logger.With(slog.String("request_id", middleware.GetReqID(r.Context())))
		ctx = log.InContext(ctx, logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}
