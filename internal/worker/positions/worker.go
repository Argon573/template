package positions

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	CheckInterval  time.Duration
	StaleAfter     time.Duration
	Workers        int
	QueueSize      int
	RequestTimeout time.Duration
}

type Worker struct {
	repo    staleTripsRepository
	push    pushClient
	log     logger
	metrics metrics
	cfg     Config
	queue   *queue
	tickers tickerFactory
	clock   clock

	scannerCancel context.CancelFunc
	scannerDone   chan struct{}
	workersWG     sync.WaitGroup
	scanMu        sync.Mutex
}

func New(
	repo staleTripsRepository,
	push pushClient,
	log logger,
	metrics metrics,
	cfg Config,
	opts ...Option,
) *Worker {
	w := &Worker{
		repo:        repo,
		push:        push,
		log:         log,
		metrics:     metrics,
		cfg:         cfg,
		queue:       newQueue(cfg.QueueSize),
		tickers:     realTickerFactory{},
		clock:       realClock{},
		scannerDone: make(chan struct{}),
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

type Option func(*Worker)

func WithTickerFactory(f tickerFactory) Option {
	return func(w *Worker) { w.tickers = f }
}

func WithClock(c clock) Option {
	return func(w *Worker) { w.clock = c }
}

func (w *Worker) Cancel(tripID uuid.UUID) {
	w.queue.Cancel(tripID)
	w.metrics.SetQueueSize(context.Background(), int64(w.queue.Len()))
}

func (w *Worker) Start(ctx context.Context) {
	scannerCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	w.scannerCancel = cancel

	for i := 0; i < w.cfg.Workers; i++ {
		w.workersWG.Add(1)
		go w.runWorker(context.WithoutCancel(ctx))
	}

	go w.runScanner(scannerCtx)
	w.log.InfoContext(ctx, "positions worker started",
		"workers", w.cfg.Workers,
		"queue_size", w.cfg.QueueSize,
		"check_interval", w.cfg.CheckInterval.String(),
	)
}

func (w *Worker) Stop(ctx context.Context) {
	w.log.InfoContext(ctx, "positions worker stopping")
	if w.scannerCancel != nil {
		w.scannerCancel()
	}
	select {
	case <-w.scannerDone:
	case <-ctx.Done():
	}

	w.queue.Close()

	done := make(chan struct{})
	go func() {
		w.workersWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		w.log.InfoContext(ctx, "positions worker stopped")
	case <-ctx.Done():
		w.log.WarnContext(ctx, "positions worker stop timed out")
	}
}

func (w *Worker) runScanner(ctx context.Context) {
	defer close(w.scannerDone)

	t := w.tickers.NewTicker(w.cfg.CheckInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C():
			if !w.scanMu.TryLock() {
				continue
			}
			w.scanOnce(ctx)
			w.scanMu.Unlock()
		}
	}
}

func (w *Worker) scanOnce(ctx context.Context) {
	tracer := otel.Tracer("github.com/Argon573/template/internal/worker/positions")
	ctx, span := tracer.Start(ctx, "stale_positions_scan")
	defer span.End()

	staleBefore := w.clock.Now().Add(-w.cfg.StaleAfter)
	trips, err := w.repo.ListStaleTrips(ctx, staleBefore)
	if err != nil {
		w.log.ErrorContext(ctx, "list stale trips failed", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	for _, trip := range trips {
		task := Task{
			TripID:      trip.ID,
			DriverID:    trip.DriverID,
			ScanSpanCtx: span.SpanContext(),
		}
		result := w.queue.TryEnqueue(task)
		w.metrics.IncStaleCheck(ctx, string(result))
		switch result {
		case EnqueueDropped:
			w.metrics.IncPositionRequest(ctx, "dropped")
			w.log.WarnContext(ctx, "position request queue full, task dropped",
				"trip_id", trip.ID.String(),
			)
		case EnqueueQueued:
			w.metrics.SetQueueSize(ctx, int64(w.queue.Len()))
		}
	}
}

func (w *Worker) runWorker(ctx context.Context) {
	defer w.workersWG.Done()

	for task := range w.queue.Ch() {
		w.metrics.SetQueueSize(ctx, int64(w.queue.Len()))
		w.processTask(ctx, task)
	}
}

func (w *Worker) processTask(ctx context.Context, task Task) {
	if !w.queue.IsPending(task.TripID) {
		return
	}

	w.metrics.IncWorkersActive(ctx)
	defer w.metrics.DecWorkersActive(ctx)
	defer w.queue.Done(task.TripID)

	tracer := otel.Tracer("github.com/Argon573/template/internal/worker/positions")
	opts := []trace.SpanStartOption{
		trace.WithAttributes(attribute.String("trip.id", task.TripID.String())),
	}
	if task.ScanSpanCtx.IsValid() {
		opts = append(opts, trace.WithLinks(trace.Link{SpanContext: task.ScanSpanCtx}))
	}
	ctx, span := tracer.Start(ctx, "position_request", opts...)
	defer span.End()

	reqCtx, cancel := context.WithTimeout(ctx, w.cfg.RequestTimeout)
	defer cancel()

	start := w.clock.Now()
	err := w.push.RequestPosition(reqCtx, task.DriverID, task.TripID)
	duration := w.clock.Now().Sub(start).Seconds()

	if err != nil {
		w.metrics.IncPositionRequest(ctx, "error")
		w.metrics.RecordPositionRequestDuration(ctx, duration, "error")
		w.log.WarnContext(ctx, "push service unavailable",
			"trip_id", task.TripID.String(),
			"error", err,
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	w.metrics.IncPositionRequest(ctx, "success")
	w.metrics.RecordPositionRequestDuration(ctx, duration, "success")
}
