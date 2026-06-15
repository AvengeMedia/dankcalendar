package repo

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func openRaw(t *testing.T, name string) (*sql.DB, string) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), name) + "?_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	return db, dsn
}

func TestMigrateFreshDatabase(t *testing.T) {
	ctx := context.Background()
	db, _ := openRaw(t, "fresh.db")
	defer db.Close()

	if err := migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	for _, col := range []string{"needs_reauth", "auth_error"} {
		ok, err := columnExists(ctx, db, "accounts", col)
		if err != nil || !ok {
			t.Fatalf("column %q missing on fresh db: ok=%v err=%v", col, ok, err)
		}
	}
}

// A database created by the old auto-migration has the pre-reauth schema and no
// goose history; it must be baselined and then upgraded without losing rows.
func TestMigrateBaselinesLegacyDatabase(t *testing.T) {
	ctx := context.Background()
	db, _ := openRaw(t, "legacy.db")
	defer db.Close()

	_, err := db.ExecContext(ctx, `CREATE TABLE accounts (
		id text NOT NULL, kind text NOT NULL, display_name text NOT NULL,
		settings json NULL, created_at datetime NOT NULL, updated_at datetime NOT NULL,
		PRIMARY KEY (id));
		INSERT INTO accounts (id, kind, display_name, created_at, updated_at)
		VALUES ('a@b.com','google','A','2026-01-01','2026-01-01');`)
	if err != nil {
		t.Fatal(err)
	}

	if err := migrate(ctx, db); err != nil {
		t.Fatalf("migrate legacy: %v", err)
	}

	ok, err := columnExists(ctx, db, "accounts", "needs_reauth")
	if err != nil || !ok {
		t.Fatalf("needs_reauth missing after upgrade: ok=%v err=%v", ok, err)
	}
	var n int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM accounts WHERE id = 'a@b.com';").Scan(&n); err != nil || n != 1 {
		t.Fatalf("existing row not preserved: n=%d err=%v", n, err)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db, _ := openRaw(t, "idem.db")
	defer db.Close()

	for i := 0; i < 3; i++ {
		if err := migrate(ctx, db); err != nil {
			t.Fatalf("migrate run %d: %v", i, err)
		}
	}
}
