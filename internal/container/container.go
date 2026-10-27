package container

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Argon573/template/internal/config"
	"github.com/Argon573/template/internal/http/server"
	createhandler "github.com/Argon573/template/internal/http/trip/create"
	finishhandler "github.com/Argon573/template/internal/http/trip/finish"
	gethandler "github.com/Argon573/template/internal/http/trip/get"
	"github.com/Argon573/template/internal/infra/postgres"
	"github.com/Argon573/template/internal/infra/postgres/transaction"
	"github.com/Argon573/template/internal/infra/telemetry"
	repotrip "github.com/Argon573/template/internal/repository/trip"
	createuc "github.com/Argon573/template/internal/usecase/trip/create"
	finishuc "github.com/Argon573/template/internal/usecase/trip/finish"
	getuc "github.com/Argon573/template/internal/usecase/trip/get"
)

type Container struct {
	Logger     *slog.Logger
	HTTPServer *http.Server
	pool       *pgxpool.Pool
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

	createUsecase := createuc.New(repo, txMgr, tel.Logger, tel.TripMetrics)
	getUsecase := getuc.New(repo, txMgr)
	finishUsecase := finishuc.New(repo, txMgr, tel.Logger, tel.TripMetrics)

	apiServer := server.New(
		createhandler.New(createUsecase),
		gethandler.New(getUsecase),
		finishhandler.New(finishUsecase),
		txMgr.Ping,
	)

	handler := server.NewRouter(apiServer, tel.Logger, tel.HTTPMetrics, cfg.OTELServiceName)

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
		pool:       pool,
		telemetry:  tel,
	}, nil
}

func (c *Container) Close(ctx context.Context) error {
	c.pool.Close()
	return c.telemetry.Shutdown(ctx)
}
