package list

import (
	"context"

	"github.com/google/uuid"

	"github.com/Argon573/template/internal/domain/trip"
)

func New(repo tripRepository) *Usecase {
	return &Usecase{repo: repo}
}

func (u *Usecase) Execute(ctx context.Context, tripID uuid.UUID) ([]trip.Position, error) {
	if _, err := u.repo.Get(ctx, tripID); err != nil {
		return nil, err
	}
	return u.repo.ListPositions(ctx, tripID)
}
