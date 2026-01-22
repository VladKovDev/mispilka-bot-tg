package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/VladKovDev/promo-bot/internal/config"
	"github.com/VladKovDev/promo-bot/pkg/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Pool struct {
	*pgxpool.Pool
	logger logger.Logger
}

func NewPool(ctx context.Context, cfg *config.DatabaseConfig, logger logger.Logger) (*Pool, error) {
	dsn := fmt.Sprintf("postgres://%s", cfg.GetDatabaseDSN())
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse db config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.ConnMaxIdleTime

	poolConfig.AfterRelease = func(conn *pgx.Conn) bool {
		return true
	}

	poolConfig.ConnConfig.ConnectTimeout = 5 * time.Second

	logger.Info("connecting to database",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Name),
		zap.Int32("max_conns", poolConfig.MaxConns),
		zap.Int32("min_conns", poolConfig.MinConns),
	)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	logger.Info("successfully connected to database")

	return &Pool{
		Pool:   pool,
		logger: logger,
	}, nil
}

func (p *Pool) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := p.Ping(ctx); err != nil {
		p.logger.Error("PostgreSQL health check failed", zap.Error(err))
		return err
	}

	return nil
}

func (p *Pool) Close() {
	p.Pool.Close()
	p.logger.Info("PostgreSQL connection pool closed")
}
