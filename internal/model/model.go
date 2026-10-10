package model

import (
	"time"

	"github.com/google/uuid"
)

type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Trip struct {
	ID       uuid.UUID `json:"id"`
	UserID   uuid.UUID `json:"user_id"`
	DriverID uuid.UUID `json:"driver_id"`

	StartPoint Coordinates `json:"start_point"`
	EndPoint   Coordinates `json:"end_point"`

	Price  int64  `json:"price"`
	Status string `json:"status"`

	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type TripStatus struct {
	Id         int64     `json:"id"`
	TripId     uuid.UUID `json:"trip_id"`
	FromStatus *string   `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Reason     *string   `json:"reason"`
	ChangedAt  time.Time `json:"changed_at"`
}
