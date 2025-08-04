package database

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
	"time"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
}

type Database interface {
	QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row
	ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error)
	Transaction(ctx context.Context, fn func(context.Context, *sql.Tx) error) error
}

func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", "postgres://"+cfg.Username+":"+cfg.Password+"@"+cfg.Host+":"+cfg.Port+"/"+cfg.Database+"?sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}
	return db, nil
}
