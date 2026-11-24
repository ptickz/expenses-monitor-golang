package store

import (
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/ptickz/expenses-monitor-golang/internal/store/db"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	_ = godotenv.Load("../../.env")
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN is not set")
	}
	if dsn == os.Getenv("DB_DSN") {
		t.Fatal("TEST_DB_DSN must not point to the dev database")
	}

	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	resetDB(t, pool)

	return New(pool)
}

func resetDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(t.Context(), `
		DO $$
		DECLARE
			tables text;
		BEGIN
			SELECT string_agg(quote_ident(tablename), ', ') INTO tables
			FROM pg_tables
			WHERE schemaname = 'public' AND tablename NOT IN ('goose_db_version', 'currencies');
			EXECUTE 'TRUNCATE ' || tables || ' RESTART IDENTITY CASCADE';
		END $$;
		UPDATE currencies SET active = false;
	`)
	if err != nil {
		t.Fatalf("reset db: %v", err)
	}
}

func TestUserAndAdminQueries(t *testing.T) {
	s := newTestStore(t)
	ctx := t.Context()

	u, err := s.q.UpsertUser(ctx, db.UpsertUserParams{Source: db.UserSourceTELEGRAM, ExternalID: "42", Name: "old"})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	again, err := s.q.UpsertUser(ctx, db.UpsertUserParams{Source: db.UserSourceTELEGRAM, ExternalID: "42", Name: "new"})
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if again.ID != u.ID || again.Name != "new" {
		t.Fatalf("upsert must keep id and update name: got id %d name %q, want id %d name %q", again.ID, again.Name, u.ID, "new")
	}

	isAdmin, err := s.q.IsAdmin(ctx, u.ID)
	if err != nil {
		t.Fatalf("is admin: %v", err)
	}
	if isAdmin {
		t.Fatal("new user must not be admin")
	}

	for range 2 {
		if err := s.q.GrantAdmin(ctx, u.ID); err != nil {
			t.Fatalf("grant admin: %v", err)
		}
	}

	isAdmin, err = s.q.IsAdmin(ctx, u.ID)
	if err != nil {
		t.Fatalf("is admin: %v", err)
	}
	if !isAdmin {
		t.Fatal("user must be admin after grant")
	}
}
