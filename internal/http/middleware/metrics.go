package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type HTTPMetrics interface {
	IncActive(ctx context.Context, method, route string)
	DecActive(ctx context.Context, method, route string)
	RecordDuration(ctx context.Context, seconds float64, method, route string, status int)
}

func Metrics(m HTTPMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = r.URL.Path
			}

			m.IncActive(r.Context(), r.Method, route)
			defer m.DecActive(r.Context(), r.Method, route)

			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			m.RecordDuration(r.Context(), time.Since(start).Seconds(), r.Method, route, rec.status)
		})
	}
}
