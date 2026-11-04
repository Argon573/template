package controller

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/Argon573/template/api/service"
	api "github.com/Argon573/template/internal/generated"
	"github.com/Argon573/template/internal/model"
	"github.com/google/uuid"
)

type Controller struct {
	api.Unimplemented
	service service.Service
}

func NewController(s service.Service) *Controller {
	return &Controller{service: s}
}

func (c *Controller) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(c.service.Health())
}

func (c *Controller) Ready(w http.ResponseWriter, r *http.Request) {
	resp := c.service.Ready(r.Context())

	w.Header().Set("Content-Type", "application/json")
	if resp.Status == api.Unavailable {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(resp)
}

type coordinatesBody struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

type createTripBody struct {
	UserID     *uuid.UUID       `json:"user_id"`
	DriverID   *uuid.UUID       `json:"driver_id"`
	StartPoint *coordinatesBody `json:"start_point"`
	EndPoint   *coordinatesBody `json:"end_point"`
	Price      *int64           `json:"price"`
}

func (c *Controller) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	dec := json.NewDecoder(r.Body)

	var raw map[string]json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		writeInvalidRequest(w, r)
		return
	}
	if dec.More() {
		writeInvalidRequest(w, r)
		return
	}

	allowed := map[string]struct{}{
		"user_id":     {},
		"driver_id":   {},
		"start_point": {},
		"end_point":   {},
		"price":       {},
	}
	for key := range raw {
		if _, ok := allowed[key]; !ok {
			writeInvalidRequest(w, r)
			return
		}
	}

	if err := checkCoordinateKeys(raw["start_point"]); err != nil {
		writeInvalidRequest(w, r)
		return
	}
	if err := checkCoordinateKeys(raw["end_point"]); err != nil {
		writeInvalidRequest(w, r)
		return
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		writeInvalidRequest(w, r)
		return
	}

	var body createTripBody
	if err := json.Unmarshal(encoded, &body); err != nil {
		writeInvalidRequest(w, r)
		return
	}

	req, err := validateCreateTrip(body)
	if err != nil {
		writeInvalidRequest(w, r)
		return
	}

	var idempotencyKey *uuid.UUID
	if params.IdempotencyKey != nil {
		key := uuid.UUID(*params.IdempotencyKey)
		idempotencyKey = &key
	}

	trip, created, err := c.service.CreateTrip(r.Context(), req, idempotencyKey)
	if err != nil {
		if errors.Is(err, service.ErrDriverBusy) {
			writeProblem(w, r, http.StatusConflict, "driver_busy",
				"https://tripgo.example/problems/driver-busy",
				"Driver busy",
				"Driver already has an active trip")
			return
		}
		if errors.Is(err, service.ErrIdempotencyConflict) {
			writeProblem(w, r, http.StatusConflict, "idempotency_conflict",
				"https://tripgo.example/problems/idempotency-conflict",
				"Idempotency conflict",
				"Idempotency-Key was already used with a different request body")
			return
		}

		log.Printf("create trip: %v", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error",
			"https://tripgo.example/problems/internal-error",
			"Internal Server Error",
			"Internal server error")
		return
	}

	status := http.StatusOK
	location := ""
	if created {
		status = http.StatusCreated
		location = "/api/v1/trips/" + trip.ID.String()
	}
	writeTrip(w, status, tripToAPI(trip), location)
}

func (c *Controller) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := c.service.GetTrip(r.Context(), tripId)
	if err != nil {
		if errors.Is(err, service.ErrTripNotFound) {
			writeProblem(w, r, http.StatusNotFound, "trip_not_found",
				"https://tripgo.example/problems/trip-not-found",
				"Trip not found",
				"Trip was not found")
			return
		}

		log.Printf("get trip: %v", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error",
			"https://tripgo.example/problems/internal-error",
			"Internal Server Error",
			"Internal server error")
		return
	}

	writeTrip(w, http.StatusOK, tripToAPI(trip), "")
}

func (c *Controller) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := c.service.FinishTrip(r.Context(), tripId)
	if err != nil {
		if errors.Is(err, service.ErrTripNotFound) {
			writeProblem(w, r, http.StatusNotFound, "trip_not_found",
				"https://tripgo.example/problems/trip-not-found",
				"Trip not found",
				"Trip was not found")
			return
		}
		if errors.Is(err, service.ErrTripCompleted) {
			writeProblem(w, r, http.StatusConflict, "trip_completed",
				"https://tripgo.example/problems/trip-completed",
				"Trip completed",
				"Operation is not allowed for a completed trip")
			return
		}

		log.Printf("finish trip: %v", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error",
			"https://tripgo.example/problems/internal-error",
			"Internal Server Error",
			"Internal server error")
		return
	}

	writeTrip(w, http.StatusOK, tripToAPI(trip), "")
}

func checkCoordinateKeys(raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return err
	}

	allowed := map[string]struct{}{
		"latitude":  {},
		"longitude": {},
	}
	for key := range obj {
		if _, ok := allowed[key]; !ok {
			return errors.New("unknown coordinate field")
		}
	}
	return nil
}

func validateCreateTrip(body createTripBody) (api.TripData, error) {
	var zero api.TripData

	if body.UserID == nil || body.DriverID == nil || body.StartPoint == nil || body.EndPoint == nil || body.Price == nil {
		return zero, errors.New("missing required field")
	}

	if *body.UserID == uuid.Nil || *body.DriverID == uuid.Nil {
		return zero, errors.New("user_id and driver_id must be non-empty UUIDs")
	}

	if body.StartPoint.Latitude == nil || body.StartPoint.Longitude == nil {
		return zero, errors.New("start_point latitude and longitude are required")
	}
	if body.EndPoint.Latitude == nil || body.EndPoint.Longitude == nil {
		return zero, errors.New("end_point latitude and longitude are required")
	}

	if err := validateCoordinates(*body.StartPoint.Latitude, *body.StartPoint.Longitude); err != nil {
		return zero, err
	}
	if err := validateCoordinates(*body.EndPoint.Latitude, *body.EndPoint.Longitude); err != nil {
		return zero, err
	}

	if *body.Price < 0 {
		return zero, errors.New("price must be >= 0")
	}

	return api.TripData{
		UserId:   *body.UserID,
		DriverId: *body.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  *body.StartPoint.Latitude,
			Longitude: *body.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  *body.EndPoint.Latitude,
			Longitude: *body.EndPoint.Longitude,
		},
		Price: *body.Price,
	}, nil
}

func validateCoordinates(lat, lon float64) error {
	if lat < -90 || lat > 90 {
		return errors.New("latitude must be between -90 and 90")
	}
	if lon < -180 || lon > 180 {
		return errors.New("longitude must be between -180 and 180")
	}
	return nil
}

func writeInvalidRequest(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, r, http.StatusBadRequest, "invalid_request",
		"https://tripgo.example/problems/invalid-request",
		"Invalid request",
		"Request validation failed")
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, typ, title, detail string) {
	instance := r.URL.Path
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.Problem{
		Type:     typ,
		Title:    title,
		Status:   int32(status),
		Detail:   &detail,
		Instance: &instance,
		Code:     code,
	})
}

func writeTrip(w http.ResponseWriter, status int, trip api.Trip, location string) {
	w.Header().Set("Content-Type", "application/json")
	if location != "" {
		w.Header().Set("Location", location)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(trip)
}

func tripToAPI(trip *model.Trip) api.Trip {
	return api.Trip{
		Id:       trip.ID,
		UserId:   trip.UserID,
		DriverId: trip.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  trip.StartPoint.Latitude,
			Longitude: trip.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  trip.EndPoint.Latitude,
			Longitude: trip.EndPoint.Longitude,
		},
		Price:          trip.Price,
		Status:         api.TripStatus(trip.Status),
		StartedAt:      trip.StartedAt,
		FinishedAt:     trip.FinishedAt,
		LastPositionAt: nil,
	}
}
