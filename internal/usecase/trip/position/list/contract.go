//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package list

import (
	"context"

	"github.com/google/uuid"

	"github.com/Argon573/template/internal/domain/trip"
)

type (
	tripRepository interface {
		Get(ctx context.Context, id uuid.UUID) (*trip.Trip, error)
		ListPositions(ctx context.Context, tripID uuid.UUID) ([]trip.Position, error)
	}
)

type Usecase struct {
	repo tripRepository
}
