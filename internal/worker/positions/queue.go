package positions

import (
	"sync"

	"github.com/google/uuid"
)

type EnqueueResult string

const (
	EnqueueQueued    EnqueueResult = "queued"
	EnqueueDuplicate EnqueueResult = "duplicate"
	EnqueueDropped   EnqueueResult = "dropped"
)

type queue struct {
	ch      chan Task
	pending map[uuid.UUID]struct{}
	mu      sync.Mutex
}

func newQueue(size int) *queue {
	return &queue{
		ch:      make(chan Task, size),
		pending: make(map[uuid.UUID]struct{}),
	}
}

func (q *queue) TryEnqueue(task Task) EnqueueResult {
	q.mu.Lock()
	if _, ok := q.pending[task.TripID]; ok {
		q.mu.Unlock()
		return EnqueueDuplicate
	}
	q.pending[task.TripID] = struct{}{}
	q.mu.Unlock()

	select {
	case q.ch <- task:
		return EnqueueQueued
	default:
		q.mu.Lock()
		delete(q.pending, task.TripID)
		q.mu.Unlock()
		return EnqueueDropped
	}
}

func (q *queue) Cancel(tripID uuid.UUID) {
	q.mu.Lock()
	delete(q.pending, tripID)
	q.mu.Unlock()
}

func (q *queue) IsPending(tripID uuid.UUID) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	_, ok := q.pending[tripID]
	return ok
}

func (q *queue) Done(tripID uuid.UUID) {
	q.mu.Lock()
	delete(q.pending, tripID)
	q.mu.Unlock()
}

func (q *queue) Len() int {
	return len(q.ch)
}

func (q *queue) Ch() <-chan Task {
	return q.ch
}

func (q *queue) Close() {
	close(q.ch)
}
