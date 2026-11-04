package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	_ = godotenv.Load()

	dbcfg := MakeDbConfig()
	serverCfg := MakeServerConfig()

	config, err := pgxpool.ParseConfig(dbcfg.DatabaseUrl)
	if err != nil {
		log.Fatalf("pgxpool parse: %v", err)
	}

	config.MaxConns = int32(dbcfg.DatabaseMaxConns)
	config.MinConns = int32(dbcfg.DatabaseMinConns)
	config.MaxConnLifetime = dbcfg.DatabaseMaxConnLifetime

	connCtx, cancel := context.WithTimeout(ctx, dbcfg.DatabaseConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connCtx, config)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Fatalf("Превышено время ожидания подключения к БД (%v): %v", dbcfg.DatabaseConnectTimeout, err)
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

	txMgr := txmanager.NewTxManager(pool, dbcfg.DatabaseQueryTimeout)
	repo := repository.NewRepository(pool)
	svc := service.NewService(repo, txMgr)
	ctrl := controller.NewController(svc)

	// start server

	srv := &http.Server{
		Addr:              serverCfg.HTTP_ADDR,
		Handler:           NewRouter(ctrl),
		ReadHeaderTimeout: serverCfg.ReadHeaderTimeout,
		ReadTimeout:       serverCfg.ReadTimeout,
		WriteTimeout:      serverCfg.WriteTimeout,
		IdleTimeout:       serverCfg.IdleTimeout,
	}

	go func() {
		log.Println("server started, log_level=", serverCfg.LogLevel)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server crash: %v", err)
		}
	}()

	// shutdown

	<-ctx.Done()
	log.Println("Получен сигнал завершения. Остановка приложения")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), serverCfg.ShutdownTimeout)
	defer cancel()

	done := make(chan struct{})

	go func() {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Ошибка при остановке сервера: %v", err)
		} else {
			log.Println("Сервер успешно завершил работу")
		}

		log.Println("Закрытие соединения с БД")
		pool.Close()
		log.Println("Пул соединений с БД закрыт")

		close(done)
	}()

	select {
	case <-done:
		log.Println("Приложение полностью остановлено штатно")
	case <-shutdownCtx.Done():
		log.Printf("Превышен лимит времени на остановку (%v). Принудительное завершение", serverCfg.ShutdownTimeout)
	}
}
