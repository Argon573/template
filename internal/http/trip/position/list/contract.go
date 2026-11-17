//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package list

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/Argon573/template/internal/domain/trip"
)

type (
	usecase interface {
		Execute(ctx context.Context, tripID uuid.UUID) ([]domain.Position, error)
	}
)

type Handler struct {
	uc usecase
}
