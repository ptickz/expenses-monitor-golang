package core

import (
	"log"

	"github.com/ptickz/expenses-monitor-golang/internal/store"
)

type Core struct {
	st *store.Store
}

func New(st *store.Store) *Core {
	return &Core{
		st: st,
	}
}

func (c *Core) DoStuff() {
	log.Println("doing stuff")
}
