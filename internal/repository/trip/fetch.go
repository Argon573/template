package trip

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domain "github.com/Argon573/template/internal/domain/trip"
)

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	query, args, err := psql.
		Select(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status", "started_at", "finished_at",
			"created_at", "updated_at",
		).
		From("trips").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip query: %w", err)
	}

	var row dbTrip
	err = r.db(ctx).QueryRow(ctx, query, args...).Scan(
		&row.ID,
		&row.UserID,
		&row.DriverID,
		&row.StartLat,
		&row.StartLng,
		&row.EndLat,
		&row.EndLng,
		&row.Price,
		&row.Status,
		&row.StartedAt,
		&row.FinishedAt,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTripNotFound
		}
		return nil, fmt.Errorf("get trip: %w", err)
	}

	return toDomain(row)
}
