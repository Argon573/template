package trip

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type Trip struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	DriverID uuid.UUID

	StartPoint Coordinates
	EndPoint   Coordinates

	Price  int64
	Status Status

	StartedAt  time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type TripStatus struct {
	Id         int64
	TripId     uuid.UUID
	FromStatus *string
	ToStatus   string
	Reason     *string
	ChangedAt  time.Time
}

func (t *Trip) Finish(at time.Time) error {
	if t.Status == StatusCompleted {
		return ErrTripCompleted
	}
	if t.Status != StatusActive {
		return ErrTripCannotBeFinished
	}

	t.Status = StatusCompleted
	t.FinishedAt = &at
	t.UpdatedAt = at

	return nil
}

type CreateParams struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64
}

func New(params CreateParams, now time.Time) (*Trip, error) {
	if params.UserID == uuid.Nil {
		return nil, &ValidationErr{Field: "user_id", Reason: "cannot be empty"}
	}
	if params.DriverID == uuid.Nil {
		return nil, &ValidationErr{Field: "driver_id", Reason: "cannot be empty"}
	}
	if params.Price < 0 {
		return nil, &ValidationErr{Field: "price", Reason: "cannot be negative"}
	}

	return &Trip{
		ID:         uuid.New(),
		UserID:     params.UserID,
		DriverID:   params.DriverID,
		StartPoint: params.StartPoint,
		EndPoint:   params.EndPoint,
		Price:      params.Price,
		Status:     StatusActive,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}
