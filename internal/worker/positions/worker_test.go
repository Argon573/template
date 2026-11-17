package positions

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/goleak"

	domain "github.com/Argon573/template/internal/domain/trip"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type stubRepo struct {
	mu    sync.Mutex
	trips []domain.StaleTrip
	err   error
}

func (r *stubRepo) ListStaleTrips(context.Context, time.Time) ([]domain.StaleTrip, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	out := make([]domain.StaleTrip, len(r.trips))
	copy(out, r.trips)
	return out, nil
}

type stubPush struct {
	mu      sync.Mutex
	calls   []uuid.UUID
	err     error
	started chan struct{}
	release chan struct{}
	done    chan struct{}
	active  atomic.Int32
	max     atomic.Int32
}

func (p *stubPush) RequestPosition(ctx context.Context, _, tripID uuid.UUID) error {
	cur := p.active.Add(1)
	for {
		old := p.max.Load()
		if cur <= old || p.max.CompareAndSwap(old, cur) {
			break
		}
	}
	defer p.active.Add(-1)

	if p.started != nil {
		select {
		case p.started <- struct{}{}:
		default:
		}
	}
	if p.release != nil {
		select {
		case <-p.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	p.mu.Lock()
	p.calls = append(p.calls, tripID)
	err := p.err
	p.mu.Unlock()
	if p.done != nil {
		p.done <- struct{}{}
	}
	return err
}

func (p *stubPush) Calls() []uuid.UUID {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]uuid.UUID, len(p.calls))
	copy(out, p.calls)
	return out
}

type stubMetrics struct {
	mu      sync.Mutex
	stale   map[string]int
	req     map[string]int
	staleCh chan string
}

func newStubMetrics() *stubMetrics {
	return &stubMetrics{
		stale:   map[string]int{},
		req:     map[string]int{},
		staleCh: make(chan string, 64),
	}
}

func (m *stubMetrics) IncStaleCheck(_ context.Context, result string) {
	m.mu.Lock()
	m.stale[result]++
	m.mu.Unlock()
	select {
	case m.staleCh <- result:
	default:
	}
}

func (m *stubMetrics) IncPositionRequest(_ context.Context, result string) {
	m.mu.Lock()
	m.req[result]++
	m.mu.Unlock()
}
func (m *stubMetrics) RecordPositionRequestDuration(context.Context, float64, string) {}
func (m *stubMetrics) SetQueueSize(context.Context, int64)                            {}
func (m *stubMetrics) IncWorkersActive(context.Context)                               {}
func (m *stubMetrics) DecWorkersActive(context.Context)                               {}

type stubLogger struct{}

func (stubLogger) InfoContext(context.Context, string, ...any)  {}
func (stubLogger) WarnContext(context.Context, string, ...any)  {}
func (stubLogger) ErrorContext(context.Context, string, ...any) {}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func testConfig(workers, queueSize int) Config {
	return Config{
		CheckInterval:  time.Hour,
		StaleAfter:     time.Minute,
		Workers:        workers,
		QueueSize:      queueSize,
		RequestTimeout: time.Second,
	}
}

func startWorker(t *testing.T, cfg Config, push pushClient, repo staleTripsRepository) *Worker {
	t.Helper()
	w := New(repo, push, stubLogger{}, newStubMetrics(), cfg)
	w.Start(context.Background())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		w.Stop(ctx)
	})
	return w
}

func TestEnqueue_Duplicate(t *testing.T) {
	t.Parallel()

	push := &stubPush{release: make(chan struct{})}
	w := startWorker(t, testConfig(1, 4), push, &stubRepo{})

	tripID := uuid.New()
	task := Task{TripID: tripID, DriverID: uuid.New()}
	if got := w.queue.TryEnqueue(task); got != EnqueueQueued {
		t.Fatalf("got %s, want queued", got)
	}
	if got := w.queue.TryEnqueue(task); got != EnqueueDuplicate {
		t.Fatalf("got %s, want duplicate", got)
	}
	close(push.release)
}

func TestEnqueue_Dropped(t *testing.T) {
	t.Parallel()

	push := &stubPush{release: make(chan struct{}), started: make(chan struct{}, 1)}
	w := startWorker(t, testConfig(1, 1), push, &stubRepo{})

	blocker := Task{TripID: uuid.New(), DriverID: uuid.New()}
	if got := w.queue.TryEnqueue(blocker); got != EnqueueQueued {
		t.Fatalf("blocker: got %s, want queued", got)
	}
	select {
	case <-push.started:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not pick blocker")
	}

	queued := Task{TripID: uuid.New(), DriverID: uuid.New()}
	if got := w.queue.TryEnqueue(queued); got != EnqueueQueued {
		t.Fatalf("queued: got %s, want queued", got)
	}
	if got := w.queue.TryEnqueue(Task{TripID: uuid.New(), DriverID: uuid.New()}); got != EnqueueDropped {
		t.Fatalf("overflow: got %s, want dropped", got)
	}
	close(push.release)
}

func TestCancel_SkipsPush(t *testing.T) {
	t.Parallel()

	push := &stubPush{
		release: make(chan struct{}),
		started: make(chan struct{}, 1),
		done:    make(chan struct{}, 8),
	}
	w := startWorker(t, testConfig(1, 4), push, &stubRepo{})

	blocker := Task{TripID: uuid.New(), DriverID: uuid.New()}
	target := uuid.New()
	if got := w.queue.TryEnqueue(blocker); got != EnqueueQueued {
		t.Fatalf("blocker: got %s, want queued", got)
	}
	select {
	case <-push.started:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not pick blocker")
	}

	if got := w.queue.TryEnqueue(Task{TripID: target, DriverID: uuid.New()}); got != EnqueueQueued {
		t.Fatalf("target: got %s, want queued", got)
	}
	w.Cancel(target)
	close(push.release)

	select {
	case <-push.done:
	case <-time.After(2 * time.Second):
		t.Fatal("blocker push did not finish")
	}
	if got := push.Calls(); len(got) != 1 || got[0] != blocker.TripID {
		t.Fatalf("push calls=%v, want only blocker", got)
	}
	if w.queue.IsPending(target) {
		t.Fatal("target still pending")
	}
}

func TestWorkers_ProcessInParallel(t *testing.T) {
	t.Parallel()

	const n = 8
	push := &stubPush{
		release: make(chan struct{}),
		started: make(chan struct{}, n),
		done:    make(chan struct{}, n),
	}
	w := startWorker(t, testConfig(4, n), push, &stubRepo{})

	for i := 0; i < n; i++ {
		if got := w.queue.TryEnqueue(Task{TripID: uuid.New(), DriverID: uuid.New()}); got != EnqueueQueued {
			t.Fatalf("enqueue %d: %s", i, got)
		}
	}

	for i := 0; i < 4; i++ {
		select {
		case <-push.started:
		case <-time.After(2 * time.Second):
			t.Fatalf("worker %d did not start", i)
		}
	}
	if got := push.max.Load(); got < 4 {
		t.Fatalf("max active workers: got %d, want >= 4", got)
	}
	close(push.release)

	for i := 0; i < n; i++ {
		select {
		case <-push.done:
		case <-time.After(2 * time.Second):
			t.Fatalf("processed %d, want %d", i, n)
		}
	}
}

func TestPushError_DoesNotStopWorker(t *testing.T) {
	t.Parallel()

	push := &stubPush{err: errors.New("push down"), done: make(chan struct{}, 2)}
	w := startWorker(t, testConfig(1, 4), push, &stubRepo{})

	_ = w.queue.TryEnqueue(Task{TripID: uuid.New(), DriverID: uuid.New()})
	_ = w.queue.TryEnqueue(Task{TripID: uuid.New(), DriverID: uuid.New()})

	for i := 0; i < 2; i++ {
		select {
		case <-push.done:
		case <-time.After(2 * time.Second):
			t.Fatalf("processed %d, want 2", i)
		}
	}
}

type manualTicker struct {
	ch chan time.Time
}

func (t *manualTicker) C() <-chan time.Time { return t.ch }
func (t *manualTicker) Stop()               {}

type manualTickerFactory struct {
	ticker *manualTicker
}

func (f manualTickerFactory) NewTicker(time.Duration) ticker { return f.ticker }

func TestScanner_EnqueueFromRepo(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tripID := uuid.New()
	repo := &stubRepo{trips: []domain.StaleTrip{{
		ID: tripID, DriverID: uuid.New(), LastActivity: now.Add(-2 * time.Minute),
	}}}
	push := &stubPush{release: make(chan struct{})}
	metrics := newStubMetrics()
	tick := &manualTicker{ch: make(chan time.Time)}

	w := New(repo, push, stubLogger{}, metrics, testConfig(1, 4),
		WithClock(fixedClock{now: now}),
		WithTickerFactory(manualTickerFactory{ticker: tick}),
	)
	w.Start(context.Background())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		w.Stop(ctx)
	})

	tick.ch <- now
	select {
	case got := <-metrics.staleCh:
		if got != "queued" {
			t.Fatalf("first tick: got %s, want queued", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ticker did not queue trip")
	}

	tick.ch <- now
	select {
	case got := <-metrics.staleCh:
		if got != "duplicate" {
			t.Fatalf("second tick: got %s, want duplicate", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second tick did not report duplicate")
	}
	close(push.release)
}

func TestStop_NoGoroutineLeak(t *testing.T) {
	_ = startWorker(t, testConfig(2, 4), &stubPush{}, &stubRepo{})
}
