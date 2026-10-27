package get

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Argon573/template/internal/domain/trip"
)

func New(repo tripRepository, tx txManager) *Usecase {
	return &Usecase{
		repo: repo,
		tx:   tx,
	}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (*trip.Trip, error) {
	var result *trip.Trip

	err := u.tx.DoSingle(ctx, func(ctx context.Context) error {
		t, err := u.repo.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("get trip: %w", err)
		}
		result = t
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
