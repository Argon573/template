//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/Argon573/template/internal/domain/trip"
)

type (
	tripRepository interface {
		Get(ctx context.Context, id uuid.UUID) (*trip.Trip, error)
	}

	txManager interface {
		DoSingle(ctx context.Context, fn func(ctx context.Context) error) error
	}
)

type Usecase struct {
	repo tripRepository
	tx   txManager
}
