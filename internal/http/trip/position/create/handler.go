package create

import (
	"encoding/json"
	"log/slog"
	"net/http"

	domain "github.com/Argon573/template/internal/domain/trip"
	api "github.com/Argon573/template/internal/generated"
	"github.com/Argon573/template/internal/http/domainerror"
	createuc "github.com/Argon573/template/internal/usecase/trip/position/create"
)

func New(uc usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var body api.PositionData
	if err := dec.Decode(&body); err != nil {
		respondError(w, r, &domain.ValidationErr{
			Field:  "body",
			Reason: "invalid json format",
		})
		return
	}

	coords, err := domain.NewCoordinates(body.Coordinates.Latitude, body.Coordinates.Longitude)
	if err != nil {
		respondError(w, r, &domain.ValidationErr{
			Field:  "coordinates",
			Reason: err.Error(),
		})
		return
	}

	pos, err := h.uc.Execute(r.Context(), createuc.Params{
		TripID:      tripID,
		Coordinates: *coords,
		RecordedAt:  body.RecordedAt.UTC(),
	})
	if err != nil {
		respondError(w, r, err)
		return
	}

	writePosition(w, http.StatusCreated, toAPIPosition(pos))
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

func writePosition(w http.ResponseWriter, status int, pos api.TripPosition) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(pos)
}

func toAPIPosition(p *domain.Position) api.TripPosition {
	return api.TripPosition{
		Id:     p.ID,
		TripId: p.TripID,
		Coordinates: api.Coordinates{
			Latitude:  p.Coordinates.Latitude,
			Longitude: p.Coordinates.Longitude,
		},
		RecordedAt: p.RecordedAt,
	}
}
