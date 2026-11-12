package repository

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Argon573/template/internal/model"
)

func (r *repository) LockIdempotencyKey(ctx context.Context, key uuid.UUID) error {
	_, err := r.db(ctx).Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`, key.String())
	if err != nil {
		return fmt.Errorf("lock idempotency key: %w", err)
	}
	return nil
}

func (r *repository) GetIdempotencyKey(ctx context.Context, key uuid.UUID) (*model.IdempotencyRecord, error) {
	query, args, err := psql.
		Select("key", "request_hash", "trip_id", "created_at").
		From("idempotency_keys").
		Where(sq.Eq{"key": key}).
		ToSql()
	if err != nil {
		return nil, err
	}

	var rec model.IdempotencyRecord
	err = r.db(ctx).QueryRow(ctx, query, args...).Scan(
		&rec.Key,
		&rec.RequestHash,
		&rec.TripID,
		&rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &rec, nil
}

func (r *repository) SaveIdempotencyKey(ctx context.Context, key uuid.UUID, requestHash string, tripID uuid.UUID) error {
	query, args, err := psql.
		Insert("idempotency_keys").
		Columns("key", "request_hash", "trip_id").
		Values(key, requestHash, tripID).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db(ctx).Exec(ctx, query, args...)
	return err
}

func (r *repository) DeleteIdempotencyKey(ctx context.Context, key uuid.UUID) error {
	query, args, err := psql.
		Delete("idempotency_keys").
		Where(sq.Eq{"key": key}).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db(ctx).Exec(ctx, query, args...)
	return err
}
