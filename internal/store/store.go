package store

import (
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
