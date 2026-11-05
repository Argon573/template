//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package finish

import (
	"context"

	"github.com/google/uuid"

	"github.com/Argon573/template/internal/domain/trip"
)

type (
	tripRepository interface {
		Get(ctx context.Context, id uuid.UUID) (*trip.Trip, error)
		Save(ctx context.Context, t trip.Trip) error
		SaveStatusHistory(ctx context.Context, status trip.TripStatus) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	logger interface {
		InfoContext(ctx context.Context, msg string, args ...any)
	}

	metrics interface {
		IncCompleted(ctx context.Context)
	}
)

type Usecase struct {
	repo    tripRepository
	tx      txManager
	log     logger
	metrics metrics
}
