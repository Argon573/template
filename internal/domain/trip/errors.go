package trip

import (
	"errors"
	"fmt"
)

var (
	ErrTripNotFound         = errors.New("trip not found")
	ErrTripCompleted        = errors.New("trip already completed")
	ErrDriverBusy           = errors.New("driver already has an active trip")
	ErrIdempotencyConflict  = errors.New("idempotency conflict")
	ErrTripCannotBeFinished = errors.New("cannot finish trip in current status")
)

type ValidationErr struct {
	Field  string
	Reason string
}

func (e *ValidationErr) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Reason)
}
