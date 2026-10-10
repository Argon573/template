package repository

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Argon573/template/internal/database/txmanager"
	"github.com/Argon573/template/internal/model"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type Repository interface {
	CreateTrip(ctx context.Context, trip *model.Trip) error
	CreateTripStatusHistory(ctx context.Context, trip *model.TripStatus) error
	GetTrip(ctx context.Context, id uuid.UUID) (*model.Trip, error)
	FinishTrip(ctx context.Context, id uuid.UUID) (*model.Trip, error)
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) db(ctx context.Context) txmanager.DBTX {
	return txmanager.Executor(ctx, r.pool)
}

func (r *repository) CreateTrip(ctx context.Context, trip *model.Trip) error {
	query, args, err := psql.
		Insert("trips").
		Columns(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status", "started_at",
		).
		Values(
			trip.ID, trip.UserID, trip.DriverID,
			trip.StartPoint.Latitude, trip.StartPoint.Longitude,
			trip.EndPoint.Latitude, trip.EndPoint.Longitude,
			trip.Price, trip.Status, trip.StartedAt,
		).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db(ctx).Exec(ctx, query, args...)
	return err
}

func (r *repository) CreateTripStatusHistory(ctx context.Context, trip *model.TripStatus) error {
	query, args, err := psql.
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason", "changed_at").
		Values(trip.TripId, trip.FromStatus, trip.ToStatus, trip.Reason, trip.ChangedAt).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db(ctx).Exec(ctx, query, args...)
	return err
}

func (r *repository) GetTrip(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
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
		return nil, err
	}

	var trip model.Trip
	err = r.db(ctx).QueryRow(ctx, query, args...).Scan(
		&trip.ID,
		&trip.UserID,
		&trip.DriverID,
		&trip.StartPoint.Latitude,
		&trip.StartPoint.Longitude,
		&trip.EndPoint.Latitude,
		&trip.EndPoint.Longitude,
		&trip.Price,
		&trip.Status,
		&trip.StartedAt,
		&trip.FinishedAt,
		&trip.CreatedAt,
		&trip.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &trip, nil
}

func (r *repository) FinishTrip(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	query, args, err := psql.
		Update("trips").
		Set("status", "completed").
		Set("finished_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id, "status": "active"}).
		Suffix(`RETURNING
			id, user_id, driver_id,
			start_latitude, start_longitude,
			end_latitude, end_longitude,
			price, status, started_at, finished_at,
			created_at, updated_at`).
		ToSql()
	if err != nil {
		return nil, err
	}

	var trip model.Trip
	err = r.db(ctx).QueryRow(ctx, query, args...).Scan(
		&trip.ID,
		&trip.UserID,
		&trip.DriverID,
		&trip.StartPoint.Latitude,
		&trip.StartPoint.Longitude,
		&trip.EndPoint.Latitude,
		&trip.EndPoint.Longitude,
		&trip.Price,
		&trip.Status,
		&trip.StartedAt,
		&trip.FinishedAt,
		&trip.CreatedAt,
		&trip.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("failed to finish trip: %w", err)
	}

	return &trip, nil
}
