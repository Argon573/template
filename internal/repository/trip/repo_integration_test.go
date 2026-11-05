//go:build integration

package trip_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	domain "github.com/Argon573/template/internal/domain/trip"
	"github.com/Argon573/template/internal/infra/postgres/transaction"
	repotrip "github.com/Argon573/template/internal/repository/trip"
)

func openPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("pool.Ping: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

func setup(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	teardown(t, pool)
}

func teardown(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := pool.Exec(ctx, `TRUNCATE trip_status_history, trips RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func newActiveTrip() domain.Trip {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	return domain.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: domain.Coordinates{Latitude: 55.75, Longitude: 37.62},
		EndPoint:   domain.Coordinates{Latitude: 59.93, Longitude: 30.31},
		Price:      1500,
		Status:     domain.StatusActive,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func TestRepository_SaveAndGet(t *testing.T) {
	pool := openPool(t)
	setup(t, pool)
	t.Cleanup(func() { teardown(t, pool) })

	repo := repotrip.New(pool)
	ctx := context.Background()
	want := newActiveTrip()

	if err := repo.Save(ctx, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.Get(ctx, want.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != want.ID {
		t.Fatalf("ID: got %v, want %v", got.ID, want.ID)
	}
	if got.UserID != want.UserID || got.DriverID != want.DriverID {
		t.Fatalf("ids mismatch: got user=%v driver=%v", got.UserID, got.DriverID)
	}
	if got.Status != domain.StatusActive {
		t.Fatalf("Status: got %s, want %s", got.Status, domain.StatusActive)
	}
	if got.Price != want.Price {
		t.Fatalf("Price: got %d, want %d", got.Price, want.Price)
	}
	if got.StartPoint.Latitude != want.StartPoint.Latitude {
		t.Fatalf("StartPoint: got %+v, want %+v", got.StartPoint, want.StartPoint)
	}
}

func TestRepository_SaveStatusHistory(t *testing.T) {
	pool := openPool(t)
	setup(t, pool)
	t.Cleanup(func() { teardown(t, pool) })

	repo := repotrip.New(pool)
	ctx := context.Background()
	tr := newActiveTrip()

	if err := repo.Save(ctx, tr); err != nil {
		t.Fatalf("Save trip: %v", err)
	}

	want := domain.TripStatus{
		TripId:     tr.ID,
		FromStatus: nil,
		ToStatus:   string(domain.StatusActive),
		ChangedAt:  tr.StartedAt,
	}
	err := repo.SaveStatusHistory(ctx, want)
	if err != nil {
		t.Fatalf("SaveStatusHistory: %v", err)
	}

	var (
		gotTripID    uuid.UUID
		gotFrom      *string
		gotTo        string
		gotChangedAt time.Time
	)
	err = pool.QueryRow(ctx, `
		SELECT trip_id, from_status, to_status, changed_at
		FROM trip_status_history
		WHERE trip_id = $1
	`, tr.ID).Scan(&gotTripID, &gotFrom, &gotTo, &gotChangedAt)
	if err != nil {
		t.Fatalf("read status history: %v", err)
	}
	if gotTripID != want.TripId {
		t.Fatalf("trip_id: got %v, want %v", gotTripID, want.TripId)
	}
	if gotFrom != nil {
		t.Fatalf("from_status: got %v, want nil", *gotFrom)
	}
	if gotTo != want.ToStatus {
		t.Fatalf("to_status: got %s, want %s", gotTo, want.ToStatus)
	}
	if !gotChangedAt.Equal(want.ChangedAt) {
		t.Fatalf("changed_at: got %v, want %v", gotChangedAt, want.ChangedAt)
	}
}

func TestRepository_TransactionRollback(t *testing.T) {
	pool := openPool(t)
	setup(t, pool)
	t.Cleanup(func() { teardown(t, pool) })

	repo := repotrip.New(pool)
	txMgr := transaction.New(pool, 3*time.Second)
	ctx := context.Background()
	tr := newActiveTrip()

	err := txMgr.Do(ctx, func(ctx context.Context) error {
		if err := repo.Save(ctx, tr); err != nil {
			return err
		}
		return repo.SaveStatusHistory(ctx, domain.TripStatus{
			TripId:     tr.ID,
			FromStatus: nil,
			ToStatus:   "invalid",
			ChangedAt:  tr.StartedAt,
		})
	})
	if err == nil {
		t.Fatal("expected history insert error, got nil")
	}

	_, getErr := repo.Get(ctx, tr.ID)
	if !errors.Is(getErr, domain.ErrTripNotFound) {
		t.Fatalf("trip must be rolled back: got %v, want %v", getErr, domain.ErrTripNotFound)
	}

	var historyCount int
	countErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM trip_status_history WHERE trip_id = $1`, tr.ID).Scan(&historyCount)
	if countErr != nil {
		t.Fatalf("count history: %v", countErr)
	}
	if historyCount != 0 {
		t.Fatalf("history must be rolled back: got count %d", historyCount)
	}
}

func TestRepository_DriverBusy(t *testing.T) {
	pool := openPool(t)
	setup(t, pool)
	t.Cleanup(func() { teardown(t, pool) })

	repo := repotrip.New(pool)
	ctx := context.Background()

	first := newActiveTrip()
	second := newActiveTrip()
	second.DriverID = first.DriverID

	if err := repo.Save(ctx, first); err != nil {
		t.Fatalf("Save first: %v", err)
	}

	err := repo.Save(ctx, second)
	if !errors.Is(err, domain.ErrDriverBusy) {
		t.Fatalf("got %v, want %v", err, domain.ErrDriverBusy)
	}
}
