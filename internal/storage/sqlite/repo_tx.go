package sqlite

import (
	"context"
	"database/sql"

	"remi-trip-planner/internal/trips"
)

type queryRunner interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (r *Repository) runner() queryRunner {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *Repository) execContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return r.runner().ExecContext(ctx, query, args...)
}

func (r *Repository) queryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return r.runner().QueryContext(ctx, query, args...)
}

func (r *Repository) queryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return r.runner().QueryRowContext(ctx, query, args...)
}

func (r *Repository) RunInTx(ctx context.Context, fn func(trips.Repository) error) error {
	if r.tx != nil {
		return fn(r)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txRepo := &Repository{db: r.db, tx: tx}
	if err := fn(txRepo); err != nil {
		return err
	}
	return tx.Commit()
}
