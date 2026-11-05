package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	domain "github.com/Argon573/template/internal/domain/trip"
	api "github.com/Argon573/template/internal/generated"
	"github.com/Argon573/template/internal/http/domainerror"
	"github.com/Argon573/template/internal/http/middleware"
)

func NewRouter(
	srv *Server,
	logger *slog.Logger,
	metricsMiddleware func(http.Handler) http.Handler,
	serviceName string,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recover(logger))
	r.Use(middleware.Tracing(serviceName))
	r.Use(middleware.Logging(logger))

	return api.HandlerWithOptions(srv, api.ChiServerOptions{
		BaseRouter:  r,
		Middlewares: []api.MiddlewareFunc{metricsMiddleware},
		ErrorHandlerFunc: func(w http.ResponseWriter, req *http.Request, err error) {
			status, problem := domainerror.Problem(&domain.ValidationErr{
				Field:  "request",
				Reason: "invalid request",
			})
			path := req.URL.Path
			problem.Instance = &path
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(problem)
		},
	})
}
