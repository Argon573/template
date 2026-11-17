package list

import (
	"encoding/json"
	"log/slog"
	"net/http"

	api "github.com/Argon573/template/internal/generated"
	"github.com/Argon573/template/internal/http/domainerror"
)

func New(uc usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	positions, err := h.uc.Execute(r.Context(), tripID)
	if err != nil {
		respondError(w, r, err)
		return
	}

	out := make([]api.TripPosition, 0, len(positions))
	for i := range positions {
		p := positions[i]
		out = append(out, api.TripPosition{
			Id:     p.ID,
			TripId: p.TripID,
			Coordinates: api.Coordinates{
				Latitude:  p.Coordinates.Latitude,
				Longitude: p.Coordinates.Longitude,
			},
			RecordedAt: p.RecordedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(api.TripPositionsResponse{Positions: out})
}

func respondError(w http.ResponseWriter, r *http.Request, err error) {
	status, prob := domainerror.Problem(err)
	if status >= http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "request failed", "error", err)
	}

	path := r.URL.Path
	prob.Instance = &path

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(prob)
}
