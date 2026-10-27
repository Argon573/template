package create_test

import (
	"bytes"
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
	"github.com/Argon573/template/internal/http/trip/create"
)

const validCreateBody = `{
		"user_id":"11111111-1111-1111-1111-111111111111",
		"driver_id":"22222222-2222-2222-2222-222222222222",
		"start_point":{"latitude":55.75,"longitude":37.62},
		"end_point":{"latitude":59.93,"longitude":30.31},
		"price":1500
	}`

func TestHandle_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	uc := NewMockusecase(ctrl)

	tripID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	startedAt := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&domain.Trip{
		ID:         tripID,
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: domain.Coordinates{Latitude: 55.75, Longitude: 37.62},
		EndPoint:   domain.Coordinates{Latitude: 59.93, Longitude: 30.31},
		Price:      1500,
		Status:     domain.StatusActive,
		StartedAt:  startedAt,
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBufferString(validCreateBody))
	rec := httptest.NewRecorder()

	create.New(uc).Handle(rec, req, api.CreateTripParams{})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "/api/v1/trips/"+tripID.String() {
		t.Fatalf("Location: got %q", loc)
	}

	var resp api.Trip
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Id != tripID {
		t.Fatalf("id: got %v, want %v", resp.Id, tripID)
	}
	if resp.Status != api.TripStatus(domain.StatusActive) {
		t.Fatalf("status: got %s", resp.Status)
	}
	if resp.Price != 1500 {
		t.Fatalf("price: got %d", resp.Price)
	}
}

func TestHandle_InvalidJSON(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	uc := NewMockusecase(ctrl)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBufferString(`{not-json`))
	rec := httptest.NewRecorder()

	create.New(uc).Handle(rec, req, api.CreateTripParams{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	var problem api.Problem
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Code != "invalid_request" {
		t.Fatalf("code: got %s, want invalid_request", problem.Code)
	}
}

func TestHandle_DriverBusy(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	uc := NewMockusecase(ctrl)

	uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil, domain.ErrDriverBusy)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBufferString(validCreateBody))
	rec := httptest.NewRecorder()

	create.New(uc).Handle(rec, req, api.CreateTripParams{})

	if rec.Code != http.StatusConflict {
		t.Fatalf("status: got %d, want %d, body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}

	var problem api.Problem
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Code != "driver_busy" {
		t.Fatalf("code: got %s, want driver_busy", problem.Code)
	}
}

func TestHandle_UnknownUsecaseError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	uc := NewMockusecase(ctrl)

	uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil, errors.New("db down"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBufferString(validCreateBody))
	rec := httptest.NewRecorder()

	create.New(uc).Handle(rec, req, api.CreateTripParams{})

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
	if problem.Detail != nil && bytes.Contains([]byte(*problem.Detail), []byte("db down")) {
		t.Fatalf("Detail must not leak original error: %q", *problem.Detail)
	}
}
