package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Argon573/template/api/repository"
	"github.com/Argon573/template/internal/database/txmanager"
	api "github.com/Argon573/template/internal/generated"
	"github.com/Argon573/template/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrTripNotFound  = errors.New("trip not found")
	ErrTripCompleted = errors.New("trip completed")
	ErrDriverBusy    = errors.New("driver busy")
)

type Service interface {
	Health() api.HealthResponse
	Ready(ctx context.Context) api.HealthResponse
	CreateTrip(ctx context.Context, data api.TripData) (*model.Trip, error)
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

func (s *service) CreateTrip(ctx context.Context, data api.TripData) (*model.Trip, error) {
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

	err := s.txMgr.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.CreateTrip(ctx, trip); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
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
		return nil
	})
	if err != nil {
		return nil, err
	}

	return trip, nil
}

func (s *service) GetTrip(ctx context.Context, id uuid.UUID) (*model.Trip, error) {

	var result *model.Trip

	err := s.txMgr.DoSingle(ctx, func(ctx context.Context) error {
		trip, err := s.repo.GetTrip(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
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

		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("finish trip: %w", err)
		}

		_, getErr := s.repo.GetTrip(ctx, id)
		if errors.Is(getErr, pgx.ErrNoRows) {
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
