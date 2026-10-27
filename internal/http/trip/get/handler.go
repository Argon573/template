package get

import (
	"encoding/json"
	"net/http"

	api "github.com/Argon573/template/internal/generated"
	"github.com/Argon573/template/internal/http/domainerror"
)

func New(uc usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	t, err := h.uc.Execute(r.Context(), tripID)
	if err != nil {
		respondError(w, r, err)
		return
	}

	writeTrip(w, http.StatusOK, toAPITrip(t), "")
}

func respondError(w http.ResponseWriter, r *http.Request, err error) {
	status, prob := domainerror.Problem(err)

	path := r.URL.Path
	prob.Instance = &path

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(prob)
}

func writeTrip(w http.ResponseWriter, status int, trip api.Trip, location string) {
	w.Header().Set("Content-Type", "application/json")
	if location != "" {
		w.Header().Set("Location", location)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(trip)
}
