package create_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/Argon573/template/internal/domain/trip"
	create "github.com/Argon573/template/internal/usecase/trip/position/create"
)

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

type fakeCanceller struct{ cancelled uuid.UUID }

func (c *fakeCanceller) Cancel(id uuid.UUID) { c.cancelled = id }

type fakeMetrics struct{ saved int }

func (m *fakeMetrics) IncPositionsSaved(context.Context) { m.saved++ }

func TestExecute_SavesAndCancels(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tripID := uuid.New()
	canceller := &fakeCanceller{}
	metrics := &fakeMetrics{}

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	repo.EXPECT().Get(gomock.Any(), tripID).Return(&domain.Trip{
		ID: tripID, Status: domain.StatusActive,
	}, nil)
	repo.EXPECT().SavePosition(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, pos domain.Position) (domain.Position, error) {
			pos.ID = 7
			pos.CreatedAt = now
			return pos, nil
		},
	)

	uc := create.New(repo, tx, canceller, metrics, fakeClock{now: now})
	got, err := uc.Execute(context.Background(), create.Params{
		TripID:      tripID,
		Coordinates: domain.Coordinates{Latitude: 59.93, Longitude: 30.33},
		RecordedAt:  now,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got.ID != 7 {
		t.Fatalf("id: got %d", got.ID)
	}
	if canceller.cancelled != tripID {
		t.Fatalf("cancel: got %s", canceller.cancelled)
	}
	if metrics.saved != 1 {
		t.Fatalf("metrics saved: %d", metrics.saved)
	}
}

func TestExecute_CompletedTrip(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tripID := uuid.New()

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	repo.EXPECT().Get(gomock.Any(), tripID).Return(&domain.Trip{
		ID: tripID, Status: domain.StatusCompleted,
	}, nil)

	uc := create.New(repo, tx, &fakeCanceller{}, &fakeMetrics{}, fakeClock{now: now})
	_, err := uc.Execute(context.Background(), create.Params{
		TripID:      tripID,
		Coordinates: domain.Coordinates{Latitude: 59.93, Longitude: 30.33},
		RecordedAt:  now,
	})
	if err != domain.ErrTripCompleted {
		t.Fatalf("got %v, want ErrTripCompleted", err)
	}
}
