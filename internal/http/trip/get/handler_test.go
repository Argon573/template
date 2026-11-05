package get_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/Argon573/template/internal/domain/trip"
	api "github.com/Argon573/template/internal/generated"
	"github.com/Argon573/template/internal/http/trip/get"
)

func TestHandle_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	uc := NewMockusecase(ctrl)

	tripID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	uc.EXPECT().Execute(gomock.Any(), tripID).Return(&domain.Trip{
		ID:         tripID,
		UserID:     uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		DriverID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		StartPoint: domain.Coordinates{Latitude: 55.75, Longitude: 37.62},
		EndPoint:   domain.Coordinates{Latitude: 59.93, Longitude: 30.31},
		Price:      1500,
		Status:     domain.StatusActive,
		StartedAt:  time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String(), nil)
	rec := httptest.NewRecorder()

	get.New(uc).Handle(rec, req, tripID)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp api.Trip
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Id != tripID {
		t.Fatalf("id: got %v, want %v", resp.Id, tripID)
	}
	if resp.Price != 1500 {
		t.Fatalf("price: got %d", resp.Price)
	}
}

func TestHandle_NotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	uc := NewMockusecase(ctrl)

	tripID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	uc.EXPECT().Execute(gomock.Any(), tripID).Return(nil, domain.ErrTripNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String(), nil)
	rec := httptest.NewRecorder()

	get.New(uc).Handle(rec, req, tripID)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	var problem api.Problem
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Code != "trip_not_found" {
		t.Fatalf("code: got %s, want trip_not_found", problem.Code)
	}
}

func TestHandle_UnknownUsecaseError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	uc := NewMockusecase(ctrl)

	tripID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	uc.EXPECT().Execute(gomock.Any(), tripID).Return(nil, errors.New("db down"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String(), nil)
	rec := httptest.NewRecorder()

	get.New(uc).Handle(rec, req, tripID)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want %d, body=%s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}

	var problem api.Problem
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Code != "internal_error" {
		t.Fatalf("code: got %s, want internal_error", problem.Code)
	}
	if problem.Detail != nil && *problem.Detail == "db down" {
		t.Fatal("Detail must not leak original error")
	}
}
