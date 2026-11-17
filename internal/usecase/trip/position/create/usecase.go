package create

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Argon573/template/internal/domain/trip"
)

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

func New(repo tripRepository, tx txManager, queue queueCanceller, metrics metrics, clk clock) *Usecase {
	if clk == nil {
		clk = realClock{}
	}
	return &Usecase{
		repo:    repo,
		tx:      tx,
		queue:   queue,
		metrics: metrics,
		clock:   clk,
	}
}

type Params struct {
	TripID      uuid.UUID
	Coordinates trip.Coordinates
	RecordedAt  time.Time
}

func (u *Usecase) Execute(ctx context.Context, params Params) (*trip.Position, error) {
	if err := trip.ValidateRecordedAt(params.RecordedAt, u.clock.Now()); err != nil {
		return nil, err
	}

	var saved trip.Position
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		t, err := u.repo.Get(ctx, params.TripID)
		if err != nil {
			return err
		}
		if t.Status == trip.StatusCompleted {
			return trip.ErrTripCompleted
		}

		pos, err := u.repo.SavePosition(ctx, trip.Position{
			TripID:      params.TripID,
			Coordinates: params.Coordinates,
			RecordedAt:  params.RecordedAt.UTC(),
		})
		if err != nil {
			return fmt.Errorf("save position: %w", err)
		}
		saved = pos
		return nil
	})
	if err != nil {
		return nil, err
	}

	u.queue.Cancel(params.TripID)
	u.metrics.IncPositionsSaved(ctx)
	return &saved, nil
}
