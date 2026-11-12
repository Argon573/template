package finish

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Argon573/template/internal/domain/trip"
)

func New(repo tripRepository, tx txManager, log logger, metrics metrics) *Usecase {
	return &Usecase{
		repo:    repo,
		tx:      tx,
		log:     log,
		metrics: metrics,
	}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (*trip.Trip, error) {
	var result *trip.Trip

	err := u.tx.Do(ctx, func(ctx context.Context) error {
		t, err := u.repo.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("get trip: %w", err)
		}

		now := time.Now().UTC()
		from := string(t.Status)

		if err := t.Finish(now); err != nil {
			return err
		}

		if err := u.repo.Save(ctx, *t); err != nil {
			return fmt.Errorf("save trip: %w", err)
		}

		if err := u.repo.SaveStatusHistory(ctx, trip.TripStatus{
			TripId:     t.ID,
			FromStatus: &from,
			ToStatus:   string(trip.StatusCompleted),
			ChangedAt:  now,
		}); err != nil {
			return fmt.Errorf("save trip status history: %w", err)
		}

		result = t
		return nil
	})
	if err != nil {
		return nil, err
	}

	u.metrics.IncCompleted(ctx)
	u.log.InfoContext(ctx, "trip completed", "trip_id", result.ID.String())

	return result, nil
}
