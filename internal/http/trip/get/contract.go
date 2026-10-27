//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package get

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/Argon573/template/internal/domain/trip"
)

type (
	usecase interface {
		Execute(ctx context.Context, id uuid.UUID) (*domain.Trip, error)
	}
)

type Handler struct {
	uc usecase
}
