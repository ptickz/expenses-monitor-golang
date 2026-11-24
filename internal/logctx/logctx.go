package logctx

import (
	"context"
	"log/slog"
	"slices"
)

type ctxKey struct{}

func With(ctx context.Context, attrs ...slog.Attr) context.Context {
	existing, _ := ctx.Value(ctxKey{}).([]slog.Attr)
	return context.WithValue(ctx, ctxKey{}, append(slices.Clip(existing), attrs...))
}

type Handler struct{ next slog.Handler }

func NewHandler(next slog.Handler) Handler { return Handler{next: next} }

func (h Handler) Enabled(ctx context.Context, l slog.Level) bool { return h.next.Enabled(ctx, l) }

func (h Handler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(ctxKey{}).([]slog.Attr); ok {
		r.AddAttrs(attrs...)
	}
	return h.next.Handle(ctx, r)
}

func (h Handler) WithAttrs(a []slog.Attr) slog.Handler { return Handler{next: h.next.WithAttrs(a)} }
func (h Handler) WithGroup(n string) slog.Handler      { return Handler{next: h.next.WithGroup(n)} }
