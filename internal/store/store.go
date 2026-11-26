package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ptickz/expenses-monitor-golang/internal/store/db"
)

type Store struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{
		pool: pool,
		q:    db.New(pool),
	}
}

func (s *Store) Queries() *db.Queries {
	return s.q
}

func (s *Store) InTx(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	err = fn(s.q.WithTx(tx))
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
