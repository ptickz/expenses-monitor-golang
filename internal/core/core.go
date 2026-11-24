package core

import (
	"log"
	"log/slog"

	"github.com/ptickz/expenses-monitor-golang/internal/store"
)

type Core struct {
	logger *slog.Logger
	st     *store.Store
}

func New(l *slog.Logger, st *store.Store) *Core {
	l = l.With("component", "core")

	return &Core{
		logger: l,
		st:     st,
	}
}

func (c *Core) DoStuff() {
	log.Println("doing stuff")
}
