//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package positions

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	domain "github.com/Argon573/template/internal/domain/trip"
)

type (
	staleTripsRepository interface {
		ListStaleTrips(ctx context.Context, staleBefore time.Time) ([]domain.StaleTrip, error)
	}

	pushClient interface {
		RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error
	}

	logger interface {
		InfoContext(ctx context.Context, msg string, args ...any)
		WarnContext(ctx context.Context, msg string, args ...any)
		ErrorContext(ctx context.Context, msg string, args ...any)
	}

	metrics interface {
		IncStaleCheck(ctx context.Context, result string)
		IncPositionRequest(ctx context.Context, result string)
		RecordPositionRequestDuration(ctx context.Context, seconds float64, result string)
		SetQueueSize(ctx context.Context, size int64)
		IncWorkersActive(ctx context.Context)
		DecWorkersActive(ctx context.Context)
	}

	ticker interface {
		C() <-chan time.Time
		Stop()
	}

	tickerFactory interface {
		NewTicker(d time.Duration) ticker
	}

	clock interface {
		Now() time.Time
	}
)

type Task struct {
	TripID      uuid.UUID
	DriverID    uuid.UUID
	ScanSpanCtx trace.SpanContext
}
