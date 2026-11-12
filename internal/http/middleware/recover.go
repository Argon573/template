package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"

	api "github.com/Argon573/template/internal/generated"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.ErrorContext(r.Context(), "panic recovered",
						"panic", rec,
						"stack", string(debug.Stack()),
					)

					detail := "Internal server error"
					path := r.URL.Path
					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(api.Problem{
						Type:     "https://tripgo.example/problems/internal-error",
						Title:    "Internal Server Error",
						Status:   http.StatusInternalServerError,
						Detail:   &detail,
						Instance: &path,
						Code:     "internal_error",
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
