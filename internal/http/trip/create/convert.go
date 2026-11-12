package create

import (
	"github.com/google/uuid"

	domain "github.com/Argon573/template/internal/domain/trip"
	api "github.com/Argon573/template/internal/generated"
)

type createTripBody struct {
	UserID     *uuid.UUID       `json:"user_id"`
	DriverID   *uuid.UUID       `json:"driver_id"`
	StartPoint *api.Coordinates `json:"start_point"`
	EndPoint   *api.Coordinates `json:"end_point"`
	Price      *int64           `json:"price"`
}

func validateCreateTrip(body createTripBody) (api.TripData, error) {
	var zero api.TripData

	if body.UserID == nil || body.DriverID == nil || body.StartPoint == nil || body.EndPoint == nil || body.Price == nil {
		return zero, &domain.ValidationErr{Field: "body", Reason: "missing required field"}
	}

	if *body.UserID == uuid.Nil || *body.DriverID == uuid.Nil {
		return zero, &domain.ValidationErr{Field: "user_id/driver_id", Reason: "must be non-empty UUIDs"}
	}

	if err := validateCoordinates("start_point", *body.StartPoint); err != nil {
		return zero, err
	}
	if err := validateCoordinates("end_point", *body.EndPoint); err != nil {
		return zero, err
	}

	if *body.Price < 0 {
		return zero, &domain.ValidationErr{Field: "price", Reason: "must be >= 0"}
	}

	return api.TripData{
		UserId:     *body.UserID,
		DriverId:   *body.DriverID,
		StartPoint: *body.StartPoint,
		EndPoint:   *body.EndPoint,
		Price:      *body.Price,
	}, nil
}

func validateCoordinates(field string, c api.Coordinates) error {
	if c.Latitude < -90 || c.Latitude > 90 {
		return &domain.ValidationErr{Field: field + ".latitude", Reason: "must be between -90 and 90"}
	}
	if c.Longitude < -180 || c.Longitude > 180 {
		return &domain.ValidationErr{Field: field + ".longitude", Reason: "must be between -180 and 180"}
	}
	return nil
}

func toCreateParams(data api.TripData) domain.CreateParams {
	return domain.CreateParams{
		UserID:   data.UserId,
		DriverID: data.DriverId,
		StartPoint: domain.Coordinates{
			Latitude:  data.StartPoint.Latitude,
			Longitude: data.StartPoint.Longitude,
		},
		EndPoint: domain.Coordinates{
			Latitude:  data.EndPoint.Latitude,
			Longitude: data.EndPoint.Longitude,
		},
		Price: data.Price,
	}
}

func toAPITrip(t *domain.Trip) api.Trip {
	return api.Trip{
		Id:       t.ID,
		UserId:   t.UserID,
		DriverId: t.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  t.StartPoint.Latitude,
			Longitude: t.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  t.EndPoint.Latitude,
			Longitude: t.EndPoint.Longitude,
		},
		Price:          t.Price,
		Status:         api.TripStatus(t.Status),
		StartedAt:      t.StartedAt,
		FinishedAt:     t.FinishedAt,
		LastPositionAt: nil,
	}
}
