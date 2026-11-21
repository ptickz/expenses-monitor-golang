package core

import (
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Core struct {
	db *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Core {
	return &Core{
		db: pool,
	}
}

func (c *Core) DoStuff() {
	log.Println("doing stuff")
}
