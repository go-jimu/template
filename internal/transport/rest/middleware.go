package rest

import (
	"net/http"

	"github.com/go-jimu/template/internal/pkg/context"
)

func InjectContext(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// TODO: chi使用了自定义的Context，导致替换成原生的Context会导致某些中间件异常，需要采用合并的策略解决此问题
		c2, cancel := context.GenDefaultContext()
		defer cancel()
		ctx, cancel2 := context.MergeContext(r.Context(), c2)
		defer cancel2()

		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func Log(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {}
	return http.HandlerFunc(fn)
}
