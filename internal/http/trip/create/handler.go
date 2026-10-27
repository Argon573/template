package create

import (
	"encoding/json"
	"net/http"

	domain "github.com/Argon573/template/internal/domain/trip"
	api "github.com/Argon573/template/internal/generated"
	"github.com/Argon573/template/internal/http/domainerror"
)

func New(uc usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var body createTripBody
	if err := dec.Decode(&body); err != nil {
		respondError(w, r, &domain.ValidationErr{
			Field:  "body",
			Reason: "invalid json format",
		})
		return
	}

	req, err := validateCreateTrip(body)
	if err != nil {
		respondError(w, r, err)
		return
	}

	t, err := h.uc.Execute(r.Context(), toCreateParams(req))
	if err != nil {
		respondError(w, r, err)
		return
	}

	writeTrip(w, http.StatusCreated, toAPITrip(t), "/api/v1/trips/"+t.ID.String())
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
