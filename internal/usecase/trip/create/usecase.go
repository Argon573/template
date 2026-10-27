package create

import (
	"context"
	"fmt"
	"time"

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

func (u *Usecase) Execute(ctx context.Context, params trip.CreateParams) (*trip.Trip, error) {
	now := time.Now().UTC()
	t, err := trip.New(params, now)
	if err != nil {
		return nil, err
	}

	err = u.tx.Do(ctx, func(ctx context.Context) error {
		if err := u.repo.Save(ctx, *t); err != nil {
			return fmt.Errorf("save trip: %w", err)
		}

		if err := u.repo.SaveStatusHistory(ctx, trip.TripStatus{
			TripId:     t.ID,
			FromStatus: nil,
			ToStatus:   string(trip.StatusActive),
			ChangedAt:  t.StartedAt,
		}); err != nil {
			return fmt.Errorf("save trip status history: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	u.metrics.IncCreated(ctx)
	u.log.InfoContext(ctx, "trip created", "trip_id", t.ID.String())

	return t, nil
}
