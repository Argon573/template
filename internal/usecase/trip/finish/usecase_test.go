package finish_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Argon573/template/internal/domain/trip"
	"github.com/Argon573/template/internal/usecase/trip/finish"
)

func TestExecute_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	log := NewMocklogger(ctrl)
	metrics := NewMockmetrics(ctrl)

	tripID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	startedAt := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	active := &trip.Trip{
		ID:        tripID,
		UserID:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		DriverID:  uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Status:    trip.StatusActive,
		StartedAt: startedAt,
		Price:     1500,
	}

	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
	repo.EXPECT().Get(gomock.Any(), tripID).Return(active, nil)
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
	repo.EXPECT().SaveStatusHistory(gomock.Any(), gomock.Any()).Return(nil)
	metrics.EXPECT().IncCompleted(gomock.Any())
	log.EXPECT().InfoContext(gomock.Any(), "trip completed", gomock.Any(), gomock.Any())

	uc := finish.New(repo, tx, log, metrics)
	got, err := uc.Execute(context.Background(), tripID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected trip, got nil")
	}
	if got.Status != trip.StatusCompleted {
		t.Fatalf("Status: got %s, want %s", got.Status, trip.StatusCompleted)
	}
	if got.FinishedAt == nil {
		t.Fatal("FinishedAt must be set")
	}
}

func TestExecute_AlreadyCompleted(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	log := NewMocklogger(ctrl)
	metrics := NewMockmetrics(ctrl)

	tripID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	finishedAt := time.Date(2024, 6, 15, 14, 0, 0, 0, time.UTC)
	completed := &trip.Trip{
		ID:         tripID,
		Status:     trip.StatusCompleted,
		FinishedAt: &finishedAt,
	}

	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
	repo.EXPECT().Get(gomock.Any(), tripID).Return(completed, nil)

	uc := finish.New(repo, tx, log, metrics)
	got, err := uc.Execute(context.Background(), tripID)
	if !errors.Is(err, trip.ErrTripCompleted) {
		t.Fatalf("got %v, want %v", err, trip.ErrTripCompleted)
	}
	if got != nil {
		t.Fatalf("expected nil trip, got %+v", got)
	}
}
