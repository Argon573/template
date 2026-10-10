package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Argon573/template/api/controller"
	"github.com/Argon573/template/api/repository"
	"github.com/Argon573/template/api/service"
	"github.com/Argon573/template/internal/database/txmanager"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// parse env

	if err := godotenv.Load(); err != nil {
		log.Fatalf("env is missing")
	}

	cfg := MakeConfig()

	// create pool

	config, err := pgxpool.ParseConfig(cfg.DatabaseUrl)
	if err != nil {
		log.Fatalf("pgxpool parse: %v", err)
	}

	config.MaxConns = int32(cfg.DatabaseMaxConns)
	config.MinConns = int32(cfg.DatabaseMinConns)
	config.MaxConnLifetime = cfg.DatabaseMaxConnLifetime

	connCtx, cancel := context.WithTimeout(ctx, cfg.DatabaseConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connCtx, config)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Fatalf("Превышено время ожидания подключения к БД (%v): %v", cfg.DatabaseConnectTimeout, err)
		} else if errors.Is(err, context.Canceled) {
			log.Fatalf("Подключение к БД было отменено: %v", err)
		} else {
			log.Fatalf("Ошибка инициализации pgxpool: %v", err)
		}
	}

	if err := pool.Ping(connCtx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}

	// initialization layers

	txMgr := txmanager.NewTxManager(pool, cfg.DatabaseQueryTimeout)
	repo := repository.NewRepository(pool)
	svc := service.NewService(repo, txMgr)
	ctrl := controller.NewController(svc)

	// start server

	srv := &http.Server{
		Addr:              cfg.HTTP_ADDR,
		Handler:           NewRouter(ctrl),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Println("server started, log_level=", cfg.LogLevel)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server crash: %v", err)
		}
	}()

	// shutdown

	<-ctx.Done()
	log.Println("Получен сигнал завершения. Остановка сервера")

	ctx, cancel = context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Ошибка при остановке сервера: %v", err)
	} else {
		log.Println("Сервер успешно завершил работу")
	}

	log.Println("Закрытие соединения с БД")
	pool.Close()
	log.Println("Пул соединений с бд закрыт")

	log.Println("Приложение полностью остановлено")
}
