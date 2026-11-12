package server

import (
	"context"
	"encoding/json"
	"net/http"

	api "github.com/Argon573/template/internal/generated"
	createhandler "github.com/Argon573/template/internal/http/trip/create"
	finishhandler "github.com/Argon573/template/internal/http/trip/finish"
	gethandler "github.com/Argon573/template/internal/http/trip/get"
)

type Server struct {
	api.Unimplemented

	create *createhandler.Handler
	get    *gethandler.Handler
	finish *finishhandler.Handler
	ping   func(ctx context.Context) error
}

func New(
	createH *createhandler.Handler,
	getH *gethandler.Handler,
	finishH *finishhandler.Handler,
	ping func(ctx context.Context) error,
) *Server {
	return &Server{
		create: createH,
		get:    getH,
		finish: finishH,
		ping:   ping,
	}
}

func (s *Server) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	s.create.Handle(w, r, params)
}

func (s *Server) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.get.Handle(w, r, tripID)
}

func (s *Server) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.finish.Handle(w, r, tripID)
}

func (s *Server) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: api.Ok})
}

func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	status := api.Ok
	code := http.StatusOK
	if err := s.ping(r.Context()); err != nil {
		status = api.Unavailable
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: status})
}
