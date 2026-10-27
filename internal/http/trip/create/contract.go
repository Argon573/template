//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"

	domain "github.com/Argon573/template/internal/domain/trip"
)

type (
	usecase interface {
		Execute(ctx context.Context, params domain.CreateParams) (*domain.Trip, error)
	}
)

type Handler struct {
	uc usecase
}
