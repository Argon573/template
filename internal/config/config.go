package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr                string
	LogLevel                string
	ShutdownTimeout         time.Duration
	DatabaseURL             string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
	OTELServiceName         string
	OTELEndpoint            string
	OTELProtocol            string
	OTELTracesSampler       string

	PositionCheckInterval time.Duration
	PositionStaleAfter    time.Duration
	PositionWorkers       int
	PositionQueueSize     int
	PushHTTPURL           string
	PushRequestTimeout    time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		return nil, fmt.Errorf("HTTP_ADDR is required")
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		return nil, fmt.Errorf("LOG_LEVEL is required")
	}

	shutdownTimeout, err := time.ParseDuration(os.Getenv("SHUTDOWN_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("SHUTDOWN_TIMEOUT: %w", err)
	}
	if shutdownTimeout <= 0 {
		return nil, fmt.Errorf("SHUTDOWN_TIMEOUT must be > 0")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	maxConns64, err := strconv.ParseInt(os.Getenv("DATABASE_MAX_CONNS"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_MAX_CONNS: %w", err)
	}

	minConns64, err := strconv.ParseInt(os.Getenv("DATABASE_MIN_CONNS"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_MIN_CONNS: %w", err)
	}

	maxLifetime, err := time.ParseDuration(os.Getenv("DATABASE_MAX_CONN_LIFETIME"))
	if err != nil {
		return nil, fmt.Errorf("DATABASE_MAX_CONN_LIFETIME: %w", err)
	}
	if maxLifetime <= 0 {
		return nil, fmt.Errorf("DATABASE_MAX_CONN_LIFETIME must be > 0")
	}

	connectTimeout, err := time.ParseDuration(os.Getenv("DATABASE_CONNECT_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("DATABASE_CONNECT_TIMEOUT: %w", err)
	}
	if connectTimeout <= 0 {
		return nil, fmt.Errorf("DATABASE_CONNECT_TIMEOUT must be > 0")
	}

	queryTimeout, err := time.ParseDuration(os.Getenv("DATABASE_QUERY_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("DATABASE_QUERY_TIMEOUT: %w", err)
	}
	if queryTimeout <= 0 {
		return nil, fmt.Errorf("DATABASE_QUERY_TIMEOUT must be > 0")
	}

	otelServiceName := os.Getenv("OTEL_SERVICE_NAME")
	if otelServiceName == "" {
		return nil, fmt.Errorf("OTEL_SERVICE_NAME is required")
	}

	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		return nil, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT is required")
	}

	otelProtocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")
	if otelProtocol == "" {
		return nil, fmt.Errorf("OTEL_EXPORTER_OTLP_PROTOCOL is required")
	}

	otelSampler := os.Getenv("OTEL_TRACES_SAMPLER")
	if otelSampler == "" {
		return nil, fmt.Errorf("OTEL_TRACES_SAMPLER is required")
	}

	checkInterval, err := time.ParseDuration(os.Getenv("POSITION_CHECK_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("POSITION_CHECK_INTERVAL: %w", err)
	}
	if checkInterval <= 0 {
		return nil, fmt.Errorf("POSITION_CHECK_INTERVAL must be > 0")
	}

	staleAfter, err := time.ParseDuration(os.Getenv("POSITION_STALE_AFTER"))
	if err != nil {
		return nil, fmt.Errorf("POSITION_STALE_AFTER: %w", err)
	}
	if staleAfter <= 0 {
		return nil, fmt.Errorf("POSITION_STALE_AFTER must be > 0")
	}

	workers64, err := strconv.ParseInt(os.Getenv("POSITION_WORKERS"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("POSITION_WORKERS: %w", err)
	}
	if workers64 <= 0 {
		return nil, fmt.Errorf("POSITION_WORKERS must be > 0")
	}

	queueSize64, err := strconv.ParseInt(os.Getenv("POSITION_QUEUE_SIZE"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("POSITION_QUEUE_SIZE: %w", err)
	}
	if queueSize64 <= 0 {
		return nil, fmt.Errorf("POSITION_QUEUE_SIZE must be > 0")
	}

	pushHTTPURL := os.Getenv("PUSH_HTTP_URL")
	if pushHTTPURL == "" {
		return nil, fmt.Errorf("PUSH_HTTP_URL is required")
	}

	pushTimeout, err := time.ParseDuration(os.Getenv("PUSH_REQUEST_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("PUSH_REQUEST_TIMEOUT: %w", err)
	}
	if pushTimeout <= 0 {
		return nil, fmt.Errorf("PUSH_REQUEST_TIMEOUT must be > 0")
	}

	return &Config{
		HTTPAddr:                httpAddr,
		LogLevel:                logLevel,
		ShutdownTimeout:         shutdownTimeout,
		DatabaseURL:             databaseURL,
		DatabaseMaxConns:        int32(maxConns64),
		DatabaseMinConns:        int32(minConns64),
		DatabaseMaxConnLifetime: maxLifetime,
		DatabaseConnectTimeout:  connectTimeout,
		DatabaseQueryTimeout:    queryTimeout,
		OTELServiceName:         otelServiceName,
		OTELEndpoint:            otelEndpoint,
		OTELProtocol:            otelProtocol,
		OTELTracesSampler:       otelSampler,
		PositionCheckInterval:   checkInterval,
		PositionStaleAfter:      staleAfter,
		PositionWorkers:         int(workers64),
		PositionQueueSize:       int(queueSize64),
		PushHTTPURL:             pushHTTPURL,
		PushRequestTimeout:      pushTimeout,
	}, nil
}
