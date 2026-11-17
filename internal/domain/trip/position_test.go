package trip_test

import (
	"testing"
	"time"

	domain "github.com/Argon573/template/internal/domain/trip"
)

func TestValidateRecordedAt(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	if err := domain.ValidateRecordedAt(now.Add(30*time.Second), now); err != nil {
		t.Fatalf("within minute: %v", err)
	}
	if err := domain.ValidateRecordedAt(now.Add(2*time.Minute), now); err == nil {
		t.Fatal("expected validation error for future recorded_at")
	}
	if err := domain.ValidateRecordedAt(time.Time{}, now); err == nil {
		t.Fatal("expected validation error for zero recorded_at")
	}
}
