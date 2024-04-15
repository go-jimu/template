package httpsrv

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-jimu/template/internal/pkg/bytesconv"
)

type rootController struct {
}

var _ Controller = (*rootController)(nil)

func newRootController() Controller {
	return &rootController{}
}

func (rc *rootController) Slug() string {
	return ""
}

func (rc *rootController) Middlewares() []Middleware {
	return []Middleware{
		{Middleware: CarryLog(), Scope: ScopeGlobal},
		{Middleware: middleware.RequestID, Scope: ScopeGlobal},
		{Middleware: RecordRequestID, Scope: ScopeGlobal},
		{Middleware: middleware.RealIP, Scope: ScopeGlobal},
		{Middleware: RequestLog, Scope: ScopeGlobal},
		{Middleware: middleware.Recoverer, Scope: ScopeGlobal},
	}
}

func (rc *rootController) APIs() []API {
	return []API{
		{
			Method:  http.MethodGet,
			Pattern: "/",
			Func: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(bytesconv.StringToBytes("hello world"))
			},
		},
		{
			Method:  http.MethodGet,
			Pattern: "/ping",
			Func: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(bytesconv.StringToBytes("pong"))
			},
		},
	}
}
