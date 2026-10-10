package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTP_ADDR               string
	LogLevel                string
	ShutdownTimeout         time.Duration
	DatabaseUrl             string
	DatabaseMaxConns        int
	DatabaseMinConns        int
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
}

func MakeConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("env is missing")
	}

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		log.Fatalf(".env error, HTTP_ADDR is empty")
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		log.Fatalf(".env error, LOG_LEVEL is empty")
	}

	shutdownTimeout, err := time.ParseDuration(os.Getenv("SHUTDOWN_TIMEOUT"))
	if err != nil {
		log.Fatalf(err.Error())
	}

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		log.Fatalf(".env error, DatabaseUrl is empty")
	}

	dbMaxConns, err := strconv.Atoi(os.Getenv("DATABASE_MAX_CONNS"))
	if err != nil {
		log.Fatalf(err.Error())
	}

	dbMinConns, err := strconv.Atoi(os.Getenv("DATABASE_MIN_CONNS"))
	if err != nil {
		log.Fatalf(err.Error())
	}

	maxLifetime, err := time.ParseDuration(os.Getenv("DATABASE_MAX_CONN_LIFETIME"))
	if err != nil {
		log.Fatalf(err.Error())
	}

	connectTimeout, err := time.ParseDuration(os.Getenv("DATABASE_CONNECT_TIMEOUT"))
	if err != nil {
		log.Fatalf(err.Error())
	}

	queryTimeout, err := time.ParseDuration(os.Getenv("DATABASE_QUERY_TIMEOUT"))
	if err != nil {
		log.Fatalf(err.Error())
	}

	return &Config{
		HTTP_ADDR:               httpAddr,
		LogLevel:                logLevel,
		ShutdownTimeout:         shutdownTimeout,
		DatabaseUrl:             dbUrl,
		DatabaseMaxConns:        dbMaxConns,
		DatabaseMinConns:        dbMinConns,
		DatabaseMaxConnLifetime: maxLifetime,
		DatabaseConnectTimeout:  connectTimeout,
		DatabaseQueryTimeout:    queryTimeout,
	}
}
