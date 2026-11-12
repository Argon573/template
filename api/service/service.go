package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Argon573/template/api/repository"
	"github.com/Argon573/template/internal/database/txmanager"
	api "github.com/Argon573/template/internal/generated"
	"github.com/Argon573/template/internal/model"
	"github.com/google/uuid"
)

const idempotencyTTL = 24 * time.Hour

var (
	ErrTripNotFound        = errors.New("trip not found")
	ErrTripCompleted       = errors.New("trip completed")
	ErrDriverBusy          = errors.New("driver busy")
	ErrIdempotencyConflict = errors.New("idempotency conflict")
)

type Service interface {
	Health() api.HealthResponse
	Ready(ctx context.Context) api.HealthResponse
	CreateTrip(ctx context.Context, data api.TripData, idempotencyKey *uuid.UUID) (*model.Trip, bool, error)
	GetTrip(ctx context.Context, id uuid.UUID) (*model.Trip, error)
	FinishTrip(ctx context.Context, id uuid.UUID) (*model.Trip, error)
}

type service struct {
	repo  repository.Repository
	txMgr txmanager.TxManager
}

func NewService(repo repository.Repository, txMgr txmanager.TxManager) Service {
	return &service{
		repo:  repo,
		txMgr: txMgr,
	}
}

func (s *service) Health() api.HealthResponse {
	return api.HealthResponse{Status: api.Ok}
}

func (s *service) Ready(ctx context.Context) api.HealthResponse {
	if err := s.txMgr.Ping(ctx); err != nil {
		return api.HealthResponse{Status: api.Unavailable}
	}
	return api.HealthResponse{Status: api.Ok}
}

func (s *service) CreateTrip(ctx context.Context, data api.TripData, idempotencyKey *uuid.UUID) (*model.Trip, bool, error) {
	requestHash, err := hashTripData(data)
	if err != nil {
		return nil, false, fmt.Errorf("hash request: %w", err)
	}

	var (
		result  *model.Trip
		created bool
	)

	err = s.txMgr.Do(ctx, func(ctx context.Context) error {
		if idempotencyKey != nil {
			if err := s.repo.LockIdempotencyKey(ctx, *idempotencyKey); err != nil {
				return err
			}

			rec, getErr := s.repo.GetIdempotencyKey(ctx, *idempotencyKey)
			if getErr != nil {
				return fmt.Errorf("get idempotency key: %w", getErr)
			}
			if rec != nil {
				if time.Since(rec.CreatedAt) > idempotencyTTL {
					if err := s.repo.DeleteIdempotencyKey(ctx, *idempotencyKey); err != nil {
						return fmt.Errorf("delete expired idempotency key: %w", err)
					}
				} else {
					if rec.RequestHash != requestHash {
						return ErrIdempotencyConflict
					}
					trip, err := s.repo.GetTrip(ctx, rec.TripID)
					if err != nil {
						if errors.Is(err, repository.ErrTripNotFound) {
							return ErrTripNotFound
						}
						return fmt.Errorf("get trip by idempotency key: %w", err)
					}
					result = trip
					created = false
					return nil
				}
			}
		}

		trip := &model.Trip{
			ID:       uuid.New(),
			UserID:   data.UserId,
			DriverID: data.DriverId,
			StartPoint: model.Coordinates{
				Latitude:  data.StartPoint.Latitude,
				Longitude: data.StartPoint.Longitude,
			},
			EndPoint: model.Coordinates{
				Latitude:  data.EndPoint.Latitude,
				Longitude: data.EndPoint.Longitude,
			},
			Price:     data.Price,
			Status:    "active",
			StartedAt: time.Now().UTC(),
		}

		if err := s.repo.CreateTrip(ctx, trip); err != nil {
			if errors.Is(err, repository.ErrDriverBusy) {
				return ErrDriverBusy
			}
			return fmt.Errorf("create trip: %w", err)
		}

		if err := s.repo.CreateTripStatusHistory(ctx, &model.TripStatus{
			TripId:     trip.ID,
			FromStatus: nil,
			ToStatus:   "active",
			ChangedAt:  trip.StartedAt,
		}); err != nil {
			return fmt.Errorf("create trip status history: %w", err)
		}

		if idempotencyKey != nil {
			if err := s.repo.SaveIdempotencyKey(ctx, *idempotencyKey, requestHash, trip.ID); err != nil {
				return fmt.Errorf("save idempotency key: %w", err)
			}
		}

		result = trip
		created = true
		return nil
	})
	if err != nil {
		return nil, false, err
	}

	return result, created, nil
}

func (s *service) GetTrip(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	var result *model.Trip

	err := s.txMgr.DoSingle(ctx, func(ctx context.Context) error {
		trip, err := s.repo.GetTrip(ctx, id)
		if err != nil {
			if errors.Is(err, repository.ErrTripNotFound) {
				return ErrTripNotFound
			}
			return fmt.Errorf("get trip: %w", err)
		}
		result = trip
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *service) FinishTrip(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	var result *model.Trip

	err := s.txMgr.Do(ctx, func(ctx context.Context) error {
		trip, err := s.repo.FinishTrip(ctx, id)
		if err == nil {
			from := "active"
			if err := s.repo.CreateTripStatusHistory(ctx, &model.TripStatus{
				TripId:     trip.ID,
				FromStatus: &from,
				ToStatus:   "completed",
				ChangedAt:  time.Now().UTC(),
			}); err != nil {
				return fmt.Errorf("create trip status history: %w", err)
			}
			result = trip
			return nil
		}

		if !errors.Is(err, repository.ErrTripNotFound) {
			return fmt.Errorf("finish trip: %w", err)
		}

		_, getErr := s.repo.GetTrip(ctx, id)
		if errors.Is(getErr, repository.ErrTripNotFound) {
			return ErrTripNotFound
		}
		if getErr != nil {
			return fmt.Errorf("get trip after finish miss: %w", getErr)
		}

		return ErrTripCompleted
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func hashTripData(data api.TripData) (string, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}
