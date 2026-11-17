package container

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Argon573/template/internal/config"
	pushgateway "github.com/Argon573/template/internal/gateway/push"
	"github.com/Argon573/template/internal/http/middleware"
	"github.com/Argon573/template/internal/http/server"
	createhandler "github.com/Argon573/template/internal/http/trip/create"
	finishhandler "github.com/Argon573/template/internal/http/trip/finish"
	gethandler "github.com/Argon573/template/internal/http/trip/get"
	poscreatehandler "github.com/Argon573/template/internal/http/trip/position/create"
	poslisthandler "github.com/Argon573/template/internal/http/trip/position/list"
	"github.com/Argon573/template/internal/infra/postgres"
	"github.com/Argon573/template/internal/infra/postgres/transaction"
	"github.com/Argon573/template/internal/infra/telemetry"
	repotrip "github.com/Argon573/template/internal/repository/trip"
	createuc "github.com/Argon573/template/internal/usecase/trip/create"
	finishuc "github.com/Argon573/template/internal/usecase/trip/finish"
	getuc "github.com/Argon573/template/internal/usecase/trip/get"
	poscreateuc "github.com/Argon573/template/internal/usecase/trip/position/create"
	poslistuc "github.com/Argon573/template/internal/usecase/trip/position/list"
	posworker "github.com/Argon573/template/internal/worker/positions"
)

type Container struct {
	Logger     *slog.Logger
	HTTPServer *http.Server
	Worker     *posworker.Worker
	pool       *pgxpool.Pool
	push       *pushgateway.Client
	telemetry  *telemetry.Provider
}

func New(ctx context.Context, cfg *config.Config) (*Container, error) {
	tel, err := telemetry.New(ctx, telemetry.Config{
		ServiceName: cfg.OTELServiceName,
		Endpoint:    cfg.OTELEndpoint,
		Protocol:    cfg.OTELProtocol,
		Sampler:     cfg.OTELTracesSampler,
		LogLevel:    cfg.LogLevel,
	})
	if err != nil {
		return nil, fmt.Errorf("telemetry: %w", err)
	}

	pool, err := postgres.NewPool(ctx, postgres.Config{
		URL:             cfg.DatabaseURL,
		MaxConns:        cfg.DatabaseMaxConns,
		MinConns:        cfg.DatabaseMinConns,
		MaxConnLifetime: cfg.DatabaseMaxConnLifetime,
		ConnectTimeout:  cfg.DatabaseConnectTimeout,
	})
	if err != nil {
		_ = tel.Shutdown(ctx)
		return nil, fmt.Errorf("postgres: %w", err)
	}

	txMgr := transaction.New(pool, cfg.DatabaseQueryTimeout)
	repo := repotrip.New(pool)

	pushClient := pushgateway.New(pushgateway.Config{
		BaseURL: cfg.PushHTTPURL,
		Timeout: cfg.PushRequestTimeout,
	})

	worker := posworker.New(
		repo,
		pushClient,
		tel.Logger,
		tel.PositionMetrics,
		posworker.Config{
			CheckInterval:  cfg.PositionCheckInterval,
			StaleAfter:     cfg.PositionStaleAfter,
			Workers:        cfg.PositionWorkers,
			QueueSize:      cfg.PositionQueueSize,
			RequestTimeout: cfg.PushRequestTimeout,
		},
	)

	createUsecase := createuc.New(repo, txMgr, tel.Logger, tel.TripMetrics)
	getUsecase := getuc.New(repo, txMgr)
	finishUsecase := finishuc.New(repo, txMgr, tel.Logger, tel.TripMetrics)
	posCreateUsecase := poscreateuc.New(repo, txMgr, worker, tel.PositionMetrics, nil)
	posListUsecase := poslistuc.New(repo)

	apiServer := server.New(
		createhandler.New(createUsecase),
		gethandler.New(getUsecase),
		finishhandler.New(finishUsecase),
		poscreatehandler.New(posCreateUsecase),
		poslisthandler.New(posListUsecase),
		txMgr.Ping,
	)

	handler := server.NewRouter(apiServer, tel.Logger, middleware.Metrics(tel.HTTPMetrics), cfg.OTELServiceName)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &Container{
		Logger:     tel.Logger,
		HTTPServer: httpServer,
		Worker:     worker,
		pool:       pool,
		push:       pushClient,
		telemetry:  tel,
	}, nil
}

func (c *Container) Close(ctx context.Context) error {
	c.push.Close()
	c.pool.Close()
	return c.telemetry.Shutdown(ctx)
}
