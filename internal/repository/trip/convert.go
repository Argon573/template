package trip

import (
	"time"

	"github.com/google/uuid"

	domain "github.com/Argon573/template/internal/domain/trip"
)

type dbTrip struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartLat   float64
	StartLng   float64
	EndLat     float64
	EndLng     float64
	Price      int64
	Status     string
	StartedAt  time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func toDB(t domain.Trip) dbTrip {
	return dbTrip{
		ID:         t.ID,
		UserID:     t.UserID,
		DriverID:   t.DriverID,
		StartLat:   t.StartPoint.Latitude,
		StartLng:   t.StartPoint.Longitude,
		EndLat:     t.EndPoint.Latitude,
		EndLng:     t.EndPoint.Longitude,
		Price:      t.Price,
		Status:     string(t.Status),
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}

func toDomain(row dbTrip) (*domain.Trip, error) {
	start, err := domain.NewCoordinates(row.StartLat, row.StartLng)
	if err != nil {
		return nil, err
	}
	end, err := domain.NewCoordinates(row.EndLat, row.EndLng)
	if err != nil {
		return nil, err
	}

	return &domain.Trip{
		ID:         row.ID,
		UserID:     row.UserID,
		DriverID:   row.DriverID,
		StartPoint: *start,
		EndPoint:   *end,
		Price:      row.Price,
		Status:     domain.Status(row.Status),
		StartedAt:  row.StartedAt,
		FinishedAt: row.FinishedAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}, nil
}
