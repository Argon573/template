//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"

	domain "github.com/Argon573/template/internal/domain/trip"
	createuc "github.com/Argon573/template/internal/usecase/trip/position/create"
)

type (
	usecase interface {
		Execute(ctx context.Context, params createuc.Params) (*domain.Position, error)
	}
)

type Handler struct {
	uc usecase
}
