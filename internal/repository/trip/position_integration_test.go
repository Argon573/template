//go:build integration

package trip_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/Argon573/template/internal/domain/trip"
	"github.com/Argon573/template/internal/infra/postgres/transaction"
	repotrip "github.com/Argon573/template/internal/repository/trip"
)

func TestRepository_ListStaleTrips(t *testing.T) {
	pool := openPool(t)
	setup(t, pool)
	t.Cleanup(func() { teardown(t, pool) })

	repo := repotrip.New(pool)
	ctx := context.Background()
	now := time.Now().UTC()

	fresh := newActiveTrip()
	fresh.StartedAt = now.Add(-30 * time.Second)
	fresh.LastPositionAt = ptrTime(now.Add(-10 * time.Second))

	stale := newActiveTrip()
	stale.StartedAt = now.Add(-10 * time.Minute)

	completed := newActiveTrip()
	completed.Status = domain.StatusCompleted
	finished := now
	completed.FinishedAt = &finished
	completed.StartedAt = now.Add(-10 * time.Minute)

	for _, tr := range []domain.Trip{fresh, stale, completed} {
		if err := repo.Save(ctx, tr); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if tr.LastPositionAt != nil {
			_, err := pool.Exec(ctx, `UPDATE trips SET last_position_at = $2 WHERE id = $1`, tr.ID, *tr.LastPositionAt)
			if err != nil {
				t.Fatalf("set last_position_at: %v", err)
			}
		}
	}

	got, err := repo.ListStaleTrips(ctx, now.Add(-time.Minute))
	if err != nil {
		t.Fatalf("ListStaleTrips: %v", err)
	}

	if len(got) != 1 || got[0].ID != stale.ID {
		t.Fatalf("stale trips: %+v, want only %s", got, stale.ID)
	}
}

func TestRepository_SavePosition_UpdatesLastPositionAtAndRollback(t *testing.T) {
	pool := openPool(t)
	setup(t, pool)
	t.Cleanup(func() { teardown(t, pool) })

	repo := repotrip.New(pool)
	txMgr := transaction.New(pool, 3*time.Second)
	ctx := context.Background()
	tr := newActiveTrip()
	if err := repo.Save(ctx, tr); err != nil {
		t.Fatalf("Save trip: %v", err)
	}

	newer := tr.StartedAt.Add(2 * time.Minute)
	older := tr.StartedAt.Add(time.Minute)

	saved, err := repo.SavePosition(ctx, domain.Position{
		TripID:      tr.ID,
		Coordinates: domain.Coordinates{Latitude: 55.75, Longitude: 37.62},
		RecordedAt:  newer,
	})
	if err != nil {
		t.Fatalf("SavePosition newer: %v", err)
	}
	if saved.ID == 0 {
		t.Fatal("expected position id")
	}

	_, err = repo.SavePosition(ctx, domain.Position{
		TripID:      tr.ID,
		Coordinates: domain.Coordinates{Latitude: 55.76, Longitude: 37.63},
		RecordedAt:  older,
	})
	if err != nil {
		t.Fatalf("SavePosition older: %v", err)
	}

	got, err := repo.Get(ctx, tr.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.LastPositionAt == nil || !got.LastPositionAt.Equal(newer) {
		t.Fatalf("last_position_at: got %v, want %v", got.LastPositionAt, newer)
	}

	err = txMgr.Do(ctx, func(ctx context.Context) error {
		if _, err := repo.SavePosition(ctx, domain.Position{
			TripID:      tr.ID,
			Coordinates: domain.Coordinates{Latitude: 55.77, Longitude: 37.64},
			RecordedAt:  newer.Add(time.Minute),
		}); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM trip_positions WHERE trip_id = $1`, tr.ID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("positions after rollback: got %d, want 2", count)
	}

	got, err = repo.Get(ctx, tr.ID)
	if err != nil {
		t.Fatalf("Get after rollback: %v", err)
	}
	if got.LastPositionAt == nil || !got.LastPositionAt.Equal(newer) {
		t.Fatalf("last_position_at after rollback: got %v, want %v", got.LastPositionAt, newer)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
