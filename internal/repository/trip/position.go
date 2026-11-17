package trip

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	domain "github.com/Argon573/template/internal/domain/trip"
)

func (r *Repository) SavePosition(ctx context.Context, pos domain.Position) (domain.Position, error) {
	query, args, err := psql.
		Insert("trip_positions").
		Columns("trip_id", "latitude", "longitude", "recorded_at").
		Values(pos.TripID, pos.Coordinates.Latitude, pos.Coordinates.Longitude, pos.RecordedAt).
		Suffix("RETURNING id, created_at").
		ToSql()
	if err != nil {
		return domain.Position{}, fmt.Errorf("build save position query: %w", err)
	}

	err = r.db(ctx).QueryRow(ctx, query, args...).Scan(&pos.ID, &pos.CreatedAt)
	if err != nil {
		return domain.Position{}, fmt.Errorf("save position: %w", err)
	}

	touch := `
		UPDATE trips
		SET last_position_at = CASE
			WHEN last_position_at IS NULL OR last_position_at < $2 THEN $2
			ELSE last_position_at
		END,
		updated_at = $3
		WHERE id = $1
	`
	_, err = r.db(ctx).Exec(ctx, touch, pos.TripID, pos.RecordedAt, time.Now().UTC())
	if err != nil {
		return domain.Position{}, fmt.Errorf("update last_position_at: %w", err)
	}

	return pos, nil
}

func (r *Repository) ListPositions(ctx context.Context, tripID uuid.UUID) ([]domain.Position, error) {
	query, args, err := psql.
		Select("id", "trip_id", "latitude", "longitude", "recorded_at", "created_at").
		From("trip_positions").
		Where(sq.Eq{"trip_id": tripID}).
		OrderBy("recorded_at ASC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list positions query: %w", err)
	}

	rows, err := r.db(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list positions: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Position, 0)
	for rows.Next() {
		var (
			p   domain.Position
			lat float64
			lng float64
		)
		if err := rows.Scan(&p.ID, &p.TripID, &lat, &lng, &p.RecordedAt, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}
		coords, err := domain.NewCoordinates(lat, lng)
		if err != nil {
			return nil, err
		}
		p.Coordinates = *coords
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list positions rows: %w", err)
	}
	return out, nil
}

func (r *Repository) ListStaleTrips(ctx context.Context, staleBefore time.Time) ([]domain.StaleTrip, error) {
	query := `
		SELECT id, driver_id, COALESCE(last_position_at, started_at) AS last_activity
		FROM trips
		WHERE status = 'active'
		  AND COALESCE(last_position_at, started_at) < $1
		ORDER BY COALESCE(last_position_at, started_at) ASC
	`

	rows, err := r.db(ctx).Query(ctx, query, staleBefore)
	if err != nil {
		return nil, fmt.Errorf("list stale trips: %w", err)
	}
	defer rows.Close()

	out := make([]domain.StaleTrip, 0)
	for rows.Next() {
		var t domain.StaleTrip
		if err := rows.Scan(&t.ID, &t.DriverID, &t.LastActivity); err != nil {
			return nil, fmt.Errorf("scan stale trip: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list stale trips rows: %w", err)
	}
	return out, nil
}
