package database

import (
	"context"
	"database/sql"
	"errors"
	"github.com/cenkalti/backoff/v5"
	"github.com/sony/gobreaker"
	"log"
	"time"
)

type ResilientDatabase struct {
	*sql.DB
	bo backoff.BackOff
	cb *gobreaker.CircuitBreaker
}
type ResilientDatabaseSettings struct {
	InitialBackoff   time.Duration
	BreakerTimeout   time.Duration
	BreakerThreshold uint32
}

func WithResilience(db *sql.DB, settings ResilientDatabaseSettings) *ResilientDatabase {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = settings.InitialBackoff

	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:    "resilient-db",
		Timeout: settings.BreakerTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= settings.BreakerThreshold
		},
	})
	return &ResilientDatabase{
		DB: db,
		bo: bo,
		cb: cb,
	}
}

func (r *ResilientDatabase) retry(ctx context.Context, f func() (any, error)) (any, error) {
	exec := func() (any, error) {
		return r.cb.Execute(func() (any, error) {
			log.Printf("executing: %v", f)
			return f()
		})
	}
	return backoff.Retry(ctx, func() (any, error) {
		e, err := exec()
		if err != nil && isRetryable(err) {
			log.Printf("retrying: %v", err)
			return nil, err
		}
		return e, nil
	}, backoff.WithBackOff(r.bo))
}

func (r *ResilientDatabase) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	rows, err := r.retry(ctx, func() (any, error) {
		return r.DB.QueryContext(ctx, q, args...)
	})
	if err != nil {
		return nil, err
	}
	return rows.(*sql.Rows), nil
}
func (r *ResilientDatabase) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	return r.DB.QueryRowContext(ctx, q, args...)
}

func (r *ResilientDatabase) Transaction(ctx context.Context, fn func(context.Context, *sql.Tx) error) error {
	_, err := r.retry(ctx, func() (any, error) {
		tx, err := r.DB.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		if err := fn(ctx, tx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		return tx.Commit(), nil
	})
	return err
}

func (r *ResilientDatabase) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	res, err := r.retry(ctx, func() (any, error) {
		return r.DB.ExecContext(ctx, q, args...)
	})
	if err != nil {
		return nil, err
	}
	return res.(sql.Result), nil
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, gobreaker.ErrOpenState) {
		return false
	}
	return true
}
