package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Metrics(m httpMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				next.ServeHTTP(w, r)
				return
			}

			m.IncActive(r.Context(), r.Method, route)
			defer m.DecActive(r.Context(), r.Method, route)

			next.ServeHTTP(w, r)
		})
	}
}
