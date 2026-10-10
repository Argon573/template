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
	openapi_types "github.com/oapi-codegen/runtime/types"
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

type createTripBody struct {
	UserID     *openapi_types.UUID `json:"user_id"`
	DriverID   *openapi_types.UUID `json:"driver_id"`
	StartPoint *api.Coordinates    `json:"start_point"`
	EndPoint   *api.Coordinates    `json:"end_point"`
	Price      *int64              `json:"price"`
}

func (c *Controller) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var body createTripBody
	if err := dec.Decode(&body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request",
			"https://tripgo.example/problems/invalid-request",
			"Invalid request",
			"Request validation failed")
		return
	}

	req, err := validateCreateTrip(body)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request",
			"https://tripgo.example/problems/invalid-request",
			"Invalid request",
			err.Error())
		return
	}

	trip, err := c.service.CreateTrip(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrDriverBusy) {
			writeProblem(w, r, http.StatusConflict, "driver_busy",
				"https://tripgo.example/problems/driver-busy",
				"Driver busy",
				"Driver already has an active trip")
			return
		}

		log.Printf("create trip: %v", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error",
			"https://tripgo.example/problems/internal-error",
			"Internal Server Error",
			"Internal server error")
		return
	}

	writeTrip(w, http.StatusCreated, tripToAPI(trip), "/api/v1/trips/"+trip.ID.String())
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

func validateCreateTrip(body createTripBody) (api.TripData, error) {
	var zero api.TripData

	if body.UserID == nil || body.DriverID == nil || body.StartPoint == nil || body.EndPoint == nil || body.Price == nil {
		return zero, errors.New("missing required field")
	}

	userID := uuid.UUID(*body.UserID)
	driverID := uuid.UUID(*body.DriverID)
	if userID == uuid.Nil || driverID == uuid.Nil {
		return zero, errors.New("user_id and driver_id must be non-empty UUIDs")
	}

	if err := validateCoordinates(*body.StartPoint); err != nil {
		return zero, err
	}
	if err := validateCoordinates(*body.EndPoint); err != nil {
		return zero, err
	}

	if *body.Price < 0 {
		return zero, errors.New("price must be >= 0")
	}

	return api.TripData{
		UserId:     *body.UserID,
		DriverId:   *body.DriverID,
		StartPoint: *body.StartPoint,
		EndPoint:   *body.EndPoint,
		Price:      *body.Price,
	}, nil
}

func validateCoordinates(c api.Coordinates) error {
	if c.Latitude < -90 || c.Latitude > 90 {
		return errors.New("latitude must be between -90 and 90")
	}
	if c.Longitude < -180 || c.Longitude > 180 {
		return errors.New("longitude must be between -180 and 180")
	}
	return nil
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
