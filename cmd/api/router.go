package main

import (
	"encoding/json"
	"net/http"

	"github.com/Argon573/template/api/controller"
	api "github.com/Argon573/template/internal/generated"
	"github.com/go-chi/chi/v5"
)

func NewRouter(c *controller.Controller) http.Handler {
	r := chi.NewRouter()

	return api.HandlerWithOptions(c, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, req *http.Request, err error) {
			detail := "Request validation failed"
			instance := req.URL.Path
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(api.Problem{
				Type:     "https://tripgo.example/problems/invalid-request",
				Title:    "Invalid request",
				Status:   400,
				Detail:   &detail,
				Instance: &instance,
				Code:     "invalid_request",
			})
		},
	})
}
