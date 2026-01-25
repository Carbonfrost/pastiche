package contextual

import (
	"context"
	"net/http"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpserver"
)

// Middleware provides an action which provides server middleware that
// provides the contextual services
func Middleware() cli.Action {
	var captured []any
	return cli.Pipeline(
		func(c context.Context) {
			captured = []any{
				Workspace(c),
			}
		},
		httpserver.WithMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				r = r.WithContext(With(r.Context(), captured...))
				next.ServeHTTP(rw, r)
			})
		}),
	)
}
