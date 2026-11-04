package main

import (
	"log"
	"os"
	"strconv"
	"time"
)

type DbConfig struct {
	DatabaseUrl             string
	DatabaseMaxConns        int
	DatabaseMinConns        int
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
}

func MakeDbConfig() *DbConfig {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		log.Fatalf("DATABASE_URL is required")
	}

	dbMaxConns, err := strconv.Atoi(os.Getenv("DATABASE_MAX_CONNS"))
	if err != nil {
		log.Fatalf("DATABASE_MAX_CONNS: %v", err)
	}
	if dbMaxConns <= 0 {
		log.Fatalf("DATABASE_MAX_CONNS must be > 0")
	}

	dbMinConns, err := strconv.Atoi(os.Getenv("DATABASE_MIN_CONNS"))
	if err != nil {
		log.Fatalf("DATABASE_MIN_CONNS: %v", err)
	}
	if dbMinConns < 0 {
		log.Fatalf("DATABASE_MIN_CONNS must be >= 0")
	}

	maxLifetime, err := time.ParseDuration(os.Getenv("DATABASE_MAX_CONN_LIFETIME"))
	if err != nil {
		log.Fatalf("DATABASE_MAX_CONN_LIFETIME: %v", err)
	}
	if maxLifetime <= 0 {
		log.Fatalf("DATABASE_MAX_CONN_LIFETIME must be > 0")
	}

	connectTimeout, err := time.ParseDuration(os.Getenv("DATABASE_CONNECT_TIMEOUT"))
	if err != nil {
		log.Fatalf("DATABASE_CONNECT_TIMEOUT: %v", err)
	}
	if connectTimeout <= 0 {
		log.Fatalf("DATABASE_CONNECT_TIMEOUT must be > 0")
	}

	queryTimeout, err := time.ParseDuration(os.Getenv("DATABASE_QUERY_TIMEOUT"))
	if err != nil {
		log.Fatalf("DATABASE_QUERY_TIMEOUT: %v", err)
	}
	if queryTimeout <= 0 {
		log.Fatalf("DATABASE_QUERY_TIMEOUT must be > 0")
	}

	return &DbConfig{
		DatabaseUrl:             dbUrl,
		DatabaseMaxConns:        dbMaxConns,
		DatabaseMinConns:        dbMinConns,
		DatabaseMaxConnLifetime: maxLifetime,
		DatabaseConnectTimeout:  connectTimeout,
		DatabaseQueryTimeout:    queryTimeout,
	}
}

type ServerConfig struct {
	HTTP_ADDR         string
	LogLevel          string
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func MakeServerConfig() *ServerConfig {
	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		log.Fatalf("HTTP_ADDR is required")
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		log.Fatalf("LOG_LEVEL is required")
	}

	shutdownTimeout, err := time.ParseDuration(os.Getenv("SHUTDOWN_TIMEOUT"))
	if err != nil {
		log.Fatalf("SHUTDOWN_TIMEOUT: %v", err)
	}
	if shutdownTimeout <= 0 {
		log.Fatalf("SHUTDOWN_TIMEOUT must be > 0")
	}

	readHeaderTimeout, err := time.ParseDuration(os.Getenv("READ_HEADER_TIMEOUT"))
	if err != nil {
		log.Fatalf("READ_HEADER_TIMEOUT: %v", err)
	}
	if readHeaderTimeout <= 0 {
		log.Fatalf("READ_HEADER_TIMEOUT must be > 0")
	}

	readTimeout, err := time.ParseDuration(os.Getenv("READ_TIMEOUT"))
	if err != nil {
		log.Fatalf("READ_TIMEOUT: %v", err)
	}
	if readTimeout <= 0 {
		log.Fatalf("READ_TIMEOUT must be > 0")
	}

	writeTimeout, err := time.ParseDuration(os.Getenv("WRITE_TIMEOUT"))
	if err != nil {
		log.Fatalf("WRITE_TIMEOUT: %v", err)
	}
	if writeTimeout <= 0 {
		log.Fatalf("WRITE_TIMEOUT must be > 0")
	}

	idleTimeout, err := time.ParseDuration(os.Getenv("IDLE_TIMEOUT"))
	if err != nil {
		log.Fatalf("IDLE_TIMEOUT: %v", err)
	}
	if idleTimeout <= 0 {
		log.Fatalf("IDLE_TIMEOUT must be > 0")
	}

	return &ServerConfig{
		HTTP_ADDR:         httpAddr,
		ShutdownTimeout:   shutdownTimeout,
		LogLevel:          logLevel,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}
