package trip

import (
	"time"

	"github.com/google/uuid"
)

type Position struct {
	ID          int64
	TripID      uuid.UUID
	Coordinates Coordinates
	RecordedAt  time.Time
	CreatedAt   time.Time
}

func ValidateRecordedAt(recordedAt, now time.Time) error {
	if recordedAt.IsZero() {
		return &ValidationErr{
			Field:  "recorded_at",
			Reason: "is required",
		}
	}
	if recordedAt.After(now.Add(time.Minute)) {
		return &ValidationErr{
			Field:  "recorded_at",
			Reason: "cannot be more than 1 minute in the future",
		}
	}
	return nil
}

type StaleTrip struct {
	ID           uuid.UUID
	DriverID     uuid.UUID
	LastActivity time.Time
}
