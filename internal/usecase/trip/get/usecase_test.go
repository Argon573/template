package get_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Argon573/template/internal/domain/trip"
	"github.com/Argon573/template/internal/usecase/trip/get"
)

func TestExecute_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	tripID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	expected := &trip.Trip{
		ID:        tripID,
		UserID:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		DriverID:  uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Status:    trip.StatusActive,
		Price:     1500,
		StartedAt: time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
	}

	tx.EXPECT().
		DoSingle(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
	repo.EXPECT().Get(gomock.Any(), tripID).Return(expected, nil)

	uc := get.New(repo, tx)
	got, err := uc.Execute(context.Background(), tripID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.ID != tripID {
		t.Fatalf("got %+v, want id %v", got, tripID)
	}
}

func TestExecute_NotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	tripID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tx.EXPECT().
		DoSingle(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
	repo.EXPECT().Get(gomock.Any(), tripID).Return(nil, trip.ErrTripNotFound)

	uc := get.New(repo, tx)
	got, err := uc.Execute(context.Background(), tripID)
	if !errors.Is(err, trip.ErrTripNotFound) {
		t.Fatalf("got %v, want %v", err, trip.ErrTripNotFound)
	}
	if got != nil {
		t.Fatalf("expected nil trip, got %+v", got)
	}
}
