package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Argon573/template/internal/domain/trip"
)

func TestNew(t *testing.T) {
	t.Parallel()

	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name    string
		params  trip.CreateParams
		wantErr error
	}{
		{
			name: "success",
			params: trip.CreateParams{
				UserID:     userID,
				DriverID:   driverID,
				StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.62},
				EndPoint:   trip.Coordinates{Latitude: 59.93, Longitude: 30.31},
				Price:      1500,
			},
		},
		{
			name: "empty_user_id",
			params: trip.CreateParams{
				UserID:   uuid.Nil,
				DriverID: driverID,
				Price:    100,
			},
			wantErr: &trip.ValidationErr{Field: "user_id", Reason: "cannot be empty"},
		},
		{
			name: "empty_driver_id",
			params: trip.CreateParams{
				UserID:   userID,
				DriverID: uuid.Nil,
				Price:    100,
			},
			wantErr: &trip.ValidationErr{Field: "driver_id", Reason: "cannot be empty"},
		},
		{
			name: "negative_price",
			params: trip.CreateParams{
				UserID:   userID,
				DriverID: driverID,
				Price:    -1,
			},
			wantErr: &trip.ValidationErr{Field: "price", Reason: "cannot be negative"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := trip.New(tt.params, now)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var validationErr *trip.ValidationErr
				if !errors.As(err, &validationErr) {
					t.Fatalf("expected ValidationErr, got %T: %v", err, err)
				}
				expected := tt.wantErr.(*trip.ValidationErr)
				if validationErr.Field != expected.Field || validationErr.Reason != expected.Reason {
					t.Fatalf("got %+v, want %+v", validationErr, expected)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.UserID != tt.params.UserID {
				t.Fatalf("UserID: got %v, want %v", got.UserID, tt.params.UserID)
			}
			if got.DriverID != tt.params.DriverID {
				t.Fatalf("DriverID: got %v, want %v", got.DriverID, tt.params.DriverID)
			}
			if got.Price != tt.params.Price {
				t.Fatalf("Price: got %d, want %d", got.Price, tt.params.Price)
			}
			if got.Status != trip.StatusActive {
				t.Fatalf("Status: got %s, want %s", got.Status, trip.StatusActive)
			}
			if !got.StartedAt.Equal(now) || !got.CreatedAt.Equal(now) || !got.UpdatedAt.Equal(now) {
				t.Fatalf("timestamps must equal now: started=%v created=%v updated=%v", got.StartedAt, got.CreatedAt, got.UpdatedAt)
			}
			if got.ID == uuid.Nil {
				t.Fatal("ID must be set")
			}
		})
	}
}

func TestFinish(t *testing.T) {
	t.Parallel()

	at := time.Date(2024, 7, 1, 18, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		status  trip.Status
		wantErr error
	}{
		{
			name:   "active_to_completed",
			status: trip.StatusActive,
		},
		{
			name:    "already_completed",
			status:  trip.StatusCompleted,
			wantErr: trip.ErrTripCompleted,
		},
		{
			name:    "unknown_status",
			status:  trip.Status("cancelled"),
			wantErr: trip.ErrTripCannotBeFinished,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tr := &trip.Trip{
				ID:     uuid.MustParse("33333333-3333-3333-3333-333333333333"),
				Status: tt.status,
			}

			err := tr.Finish(at)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tr.Status != trip.StatusCompleted {
				t.Fatalf("Status: got %s, want %s", tr.Status, trip.StatusCompleted)
			}
			if tr.FinishedAt == nil || !tr.FinishedAt.Equal(at) {
				t.Fatalf("FinishedAt: got %v, want %v", tr.FinishedAt, at)
			}
			if !tr.UpdatedAt.Equal(at) {
				t.Fatalf("UpdatedAt: got %v, want %v", tr.UpdatedAt, at)
			}
		})
	}
}

func TestNewCoordinates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		latitude  float64
		longitude float64
		wantErr   bool
	}{
		{name: "valid", latitude: 55.75, longitude: 37.62},
		{name: "latitude_too_high", latitude: 91, longitude: 0, wantErr: true},
		{name: "latitude_too_low", latitude: -91, longitude: 0, wantErr: true},
		{name: "longitude_too_high", latitude: 0, longitude: 181, wantErr: true},
		{name: "longitude_too_low", latitude: 0, longitude: -181, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := trip.NewCoordinates(tt.latitude, tt.longitude)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Latitude != tt.latitude || got.Longitude != tt.longitude {
				t.Fatalf("got %+v, want lat=%v lng=%v", got, tt.latitude, tt.longitude)
			}
		})
	}
}
