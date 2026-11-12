package domainerror_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	domain "github.com/Argon573/template/internal/domain/trip"
	"github.com/Argon573/template/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		leakText   string
	}{
		{
			name:       "validation",
			err:        &domain.ValidationErr{Field: "price", Reason: "must be >= 0"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "not_found",
			err:        domain.ErrTripNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
		},
		{
			name:       "driver_busy",
			err:        domain.ErrDriverBusy,
			wantStatus: http.StatusConflict,
			wantCode:   "driver_busy",
		},
		{
			name:       "trip_completed",
			err:        domain.ErrTripCompleted,
			wantStatus: http.StatusConflict,
			wantCode:   "trip_completed",
		},
		{
			name:       "idempotency_conflict",
			err:        domain.ErrIdempotencyConflict,
			wantStatus: http.StatusConflict,
			wantCode:   "idempotency_conflict",
		},
		{
			name:       "unknown",
			err:        errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
			leakText:   "db down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status, problem := domainerror.Problem(tt.err)
			if status != tt.wantStatus {
				t.Fatalf("status: got %d, want %d", status, tt.wantStatus)
			}
			if problem.Code != tt.wantCode {
				t.Fatalf("code: got %s, want %s", problem.Code, tt.wantCode)
			}
			if int(problem.Status) != tt.wantStatus {
				t.Fatalf("problem.Status: got %d, want %d", problem.Status, tt.wantStatus)
			}
			if problem.Detail == nil {
				t.Fatal("Detail must be set")
			}
			if tt.leakText != "" && strings.Contains(*problem.Detail, tt.leakText) {
				t.Fatalf("Detail must not leak original error: %q", *problem.Detail)
			}
		})
	}
}
