//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Argon573/template/internal/domain/trip"
)

type (
	tripRepository interface {
		Get(ctx context.Context, id uuid.UUID) (*trip.Trip, error)
		SavePosition(ctx context.Context, pos trip.Position) (trip.Position, error)
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	queueCanceller interface {
		Cancel(tripID uuid.UUID)
	}

	metrics interface {
		IncPositionsSaved(ctx context.Context)
	}

	clock interface {
		Now() time.Time
	}
)

type Usecase struct {
	repo    tripRepository
	tx      txManager
	queue   queueCanceller
	metrics metrics
	clock   clock
}
