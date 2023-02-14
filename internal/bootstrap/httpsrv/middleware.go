package httpsrv

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/template/internal/pkg/context"
)

type logEntry struct {
	log *logger.Helper
	req *http.Request
}

func newLogEntry(log logger.Logger, r *http.Request) middleware.LogEntry {
	return &logEntry{
		log: logger.NewHelper(log),
		req: r,
	}
}

func (le *logEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	log := le.log.WithContext(le.req.Context())
	log.Info("request complete",
		"request_method", le.req.Method,
		"request_path", le.req.URL.Path,
		"request_query", le.req.URL.RawQuery,
		// "user_agent", le.req.Header.Get("User-Agent"),
		"client_ip", le.req.RemoteAddr,
		"response_status_code", status,
		// "response_bytes_length", bytes,
		"elapsed", elapsed.String(),
	)
}

func (le *logEntry) Panic(v interface{}, stack []byte) {
	log := le.log.WithContext(le.req.Context())
	log.Error("broken request",
		"panic", v,
		"stack", string(stack),
	)
}

func CarryLog(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(logger.InContext(r.Context(), log))
		}
		return http.HandlerFunc(fn)
	}
}

func RequestLog(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			entry := newLogEntry(log, r)
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				entry.Write(ww.Status(), ww.BytesWritten(), ww.Header(), time.Since(start), nil)
			}()

			next.ServeHTTP(ww, middleware.WithLogEntry(r, entry))
		}
		return http.HandlerFunc(fn)
	}
}

func InjectContext(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.GenDefaultContext()
		defer cancel()
		mc, mcCancel := context.MergeContext(r.Context(), ctx)
		defer mcCancel()

		next.ServeHTTP(w, r.WithContext(mc))
	}

	return http.HandlerFunc(fn)
}
