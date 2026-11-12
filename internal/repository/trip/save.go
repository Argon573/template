package trip

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	domain "github.com/Argon573/template/internal/domain/trip"
)

func (r *Repository) Save(ctx context.Context, t domain.Trip) error {
	row := toDB(t)

	query, args, err := psql.
		Insert("trips").
		Columns(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status", "started_at", "finished_at",
			"created_at", "updated_at",
		).
		Values(
			row.ID, row.UserID, row.DriverID,
			row.StartLat, row.StartLng,
			row.EndLat, row.EndLng,
			row.Price, row.Status, row.StartedAt, row.FinishedAt,
			row.CreatedAt, row.UpdatedAt,
		).
		Suffix(`
			ON CONFLICT (id) DO UPDATE SET
				status = EXCLUDED.status,
				finished_at = EXCLUDED.finished_at,
				updated_at = EXCLUDED.updated_at
		`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build save trip query: %w", err)
	}

	_, err = r.db(ctx).Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDriverBusy
		}
		return fmt.Errorf("save trip: %w", err)
	}

	return nil
}

func (r *Repository) SaveStatusHistory(ctx context.Context, status domain.TripStatus) error {
	query, args, err := psql.
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason", "changed_at").
		Values(status.TripId, status.FromStatus, status.ToStatus, status.Reason, status.ChangedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build save status history query: %w", err)
	}

	_, err = r.db(ctx).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("save trip status history: %w", err)
	}

	return nil
}
