package create_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Argon573/template/internal/domain/trip"
	"github.com/Argon573/template/internal/usecase/trip/create"
)

func TestExecute_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	log := NewMocklogger(ctrl)
	metrics := NewMockmetrics(ctrl)

	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
	repo.EXPECT().SaveStatusHistory(gomock.Any(), gomock.Any()).Return(nil)
	metrics.EXPECT().IncCreated(gomock.Any())
	log.EXPECT().InfoContext(gomock.Any(), "trip created", gomock.Any(), gomock.Any())

	uc := create.New(repo, tx, log, metrics)
	params := trip.CreateParams{
		UserID:     uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		DriverID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.62},
		EndPoint:   trip.Coordinates{Latitude: 59.93, Longitude: 30.31},
		Price:      1500,
	}

	got, err := uc.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected trip, got nil")
	}
	if got.Status != trip.StatusActive {
		t.Fatalf("Status: got %s, want %s", got.Status, trip.StatusActive)
	}
	if got.UserID != params.UserID || got.DriverID != params.DriverID {
		t.Fatalf("unexpected ids: user=%v driver=%v", got.UserID, got.DriverID)
	}
	if got.Price != params.Price {
		t.Fatalf("Price: got %d, want %d", got.Price, params.Price)
	}
}

func TestExecute_DriverBusy(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	log := NewMocklogger(ctrl)
	metrics := NewMockmetrics(ctrl)

	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(trip.ErrDriverBusy)

	uc := create.New(repo, tx, log, metrics)
	params := trip.CreateParams{
		UserID:     uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		DriverID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.62},
		EndPoint:   trip.Coordinates{Latitude: 59.93, Longitude: 30.31},
		Price:      1500,
	}

	got, err := uc.Execute(context.Background(), params)
	if !errors.Is(err, trip.ErrDriverBusy) {
		t.Fatalf("got %v, want %v", err, trip.ErrDriverBusy)
	}
	if got != nil {
		t.Fatalf("expected nil trip, got %+v", got)
	}
}
