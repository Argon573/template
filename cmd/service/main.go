package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Argon573/template/internal/config"
	"github.com/Argon573/template/internal/container"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	rootCtx := context.Background()
	c, err := container.New(rootCtx, cfg)
	if err != nil {
		log.Fatalf("container: %v", err)
	}

	ctx, stop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	c.Logger.InfoContext(ctx, "service starting", "addr", cfg.HTTPAddr)

	go func() {
		if err := c.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			c.Logger.ErrorContext(ctx, "server failed", "error", err.Error())
			stop()
		}
	}()

	<-ctx.Done()
	c.Logger.InfoContext(context.Background(), "shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := c.HTTPServer.Shutdown(shutdownCtx); err != nil {
		c.Logger.ErrorContext(shutdownCtx, "forced shutdown", "error", err.Error())
	} else {
		c.Logger.InfoContext(shutdownCtx, "http server stopped")
	}

	if err := c.Close(shutdownCtx); err != nil {
		c.Logger.ErrorContext(shutdownCtx, "close failed", "error", err.Error())
	}

	c.Logger.InfoContext(context.Background(), "service stopped")
}
