package database

import (
	"context"
	"time"

	"dev.mfr/go-chi-sqlc-auth/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(cfg config.DBConfig) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
