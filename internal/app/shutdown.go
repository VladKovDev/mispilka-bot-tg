package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/VladKovDev/promo-bot/internal/infrastructure/repository/postgres"
	"github.com/VladKovDev/promo-bot/pkg/logger"
	"go.uber.org/zap"
)

func gracefulShutdown(ctx context.Context, logger logger.Logger, pool *postgres.Pool) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		logger.Info("context cancelled, starting shutdown")
	case sig := <-sigChan:
		logger.Info("received shutdown signal", zap.String("signal", sig.String()))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("closing database connections")
	pool.Close()

	select {
	case <-shutdownCtx.Done():
		logger.Warn("shutdown timeout exceeded")
		return shutdownCtx.Err()
	default:
		logger.Info("shutdown completed successfully")
		return nil
	}
}
