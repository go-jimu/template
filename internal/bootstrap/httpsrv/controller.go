package httpsrv

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-jimu/components/logger"
	internallog "github.com/go-jimu/template/internal/pkg/log"
)

type rootController struct {
	logger logger.Logger
}

var _ Controller = (*rootController)(nil)

func newRootController(log logger.Logger) Controller {
	return &rootController{logger: logger.With(
		log,
		"request_id", internallog.Carry(middleware.RequestIDKey)),
	}
}

func (rc *rootController) Slug() string {
	return ""
}

func (rc *rootController) Middlewares() []Middleware {
	return []Middleware{
		{Middleware: InjectContext, Scope: ScopeGlobal},
		{Middleware: middleware.RequestID, Scope: ScopeGlobal},
		{Middleware: middleware.RealIP, Scope: ScopeGlobal},
		{Middleware: CarryLog(rc.logger), Scope: ScopeGlobal},
		{Middleware: RequestLog(rc.logger), Scope: ScopeGlobal},
		{Middleware: middleware.Recoverer, Scope: ScopeGlobal},
	}
}

func (rc *rootController) APIs() []API {
	return []API{
		{
			Method:  http.MethodGet,
			Pattern: "/",
			Func: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("hello world"))
			},
		},
		{
			Method:  http.MethodGet,
			Pattern: "/ping",
			Func: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("pong"))
			},
		},
	}
}
