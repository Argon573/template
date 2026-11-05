package domainerror

import (
	"errors"
	"net/http"

	domain "github.com/Argon573/template/internal/domain/trip"
	api "github.com/Argon573/template/internal/generated"
)

func Problem(err error) (int, api.Problem) {
	var validationErr *domain.ValidationErr
	if errors.As(err, &validationErr) {
		detail := validationErr.Error()
		return http.StatusBadRequest, api.Problem{
			Type:   "https://tripgo.example/problems/invalid-request",
			Title:  "Invalid request",
			Status: int32(http.StatusBadRequest),
			Detail: &detail,
			Code:   "invalid_request",
		}
	}

	switch {
	case errors.Is(err, domain.ErrTripNotFound):
		detail := "Trip was not found"
		return http.StatusNotFound, api.Problem{
			Type:   "https://tripgo.example/problems/trip-not-found",
			Title:  "Trip not found",
			Status: int32(http.StatusNotFound),
			Detail: &detail,
			Code:   "trip_not_found",
		}
	case errors.Is(err, domain.ErrTripCompleted):
		detail := "Operation is not allowed for a completed trip"
		return http.StatusConflict, api.Problem{
			Type:   "https://tripgo.example/problems/trip-completed",
			Title:  "Trip completed",
			Status: int32(http.StatusConflict),
			Detail: &detail,
			Code:   "trip_completed",
		}
	case errors.Is(err, domain.ErrDriverBusy):
		detail := "Driver already has an active trip"
		return http.StatusConflict, api.Problem{
			Type:   "https://tripgo.example/problems/driver-busy",
			Title:  "Driver busy",
			Status: int32(http.StatusConflict),
			Detail: &detail,
			Code:   "driver_busy",
		}
	case errors.Is(err, domain.ErrIdempotencyConflict):
		detail := "Idempotency-Key was already used with a different request body"
		return http.StatusConflict, api.Problem{
			Type:   "https://tripgo.example/problems/idempotency-conflict",
			Title:  "Idempotency conflict",
			Status: int32(http.StatusConflict),
			Detail: &detail,
			Code:   "idempotency_conflict",
		}
	default:
		detail := "Internal server error"
		return http.StatusInternalServerError, api.Problem{
			Type:   "https://tripgo.example/problems/internal-error",
			Title:  "Internal Server Error",
			Status: int32(http.StatusInternalServerError),
			Detail: &detail,
			Code:   "internal_error",
		}
	}
}
