package middleware

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				return
			}

			span := trace.SpanFromContext(r.Context())
			span.SetAttributes(semconv.HTTPRoute(route))
			span.SetName(fmt.Sprintf("%s %s", r.Method, route))
		}), serviceName,
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				route := chi.RouteContext(r.Context()).RoutePattern()
				if route == "" {
					route = r.URL.Path
				}
				return fmt.Sprintf("%s %s", r.Method, route)
			}),
		)
	}
}
