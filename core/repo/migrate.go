package repo

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/pressly/goose/v3"

	entmigrate "github.com/AvengeMedia/dankcalendar/core/ent/migrate"
)

const migrationsDir = "migrations"

// gooseOnce guards goose's package-level configuration so concurrent Open calls
// (e.g. in tests) don't race on the shared setters.
var gooseOnce sync.Once

func configureGoose() {
	gooseOnce.Do(func() {
		goose.SetBaseFS(entmigrate.Migrations)
		goose.SetLogger(goose.NopLogger())
		// The embedded SQL is plain SQLite; the modernc driver speaks it even
		// though goose calls the dialect "sqlite3".
		_ = goose.SetDialect("sqlite3")
	})
}

// migrate brings the database up to the latest schema. Databases created before
// versioned migrations existed are baselined first so goose applies only the
// newer migrations instead of recreating tables that are already there.
func migrate(ctx context.Context, db *sql.DB) error {
	configureGoose()

	if err := baselineLegacyDB(ctx, db); err != nil {
		return fmt.Errorf("baseline database: %w", err)
	}
	if err := goose.UpContext(ctx, db, migrationsDir); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// baselineLegacyDB stamps the version history for a database that was created by
// the old auto-migration path: its tables already exist but goose has never
// tracked it. We mark the initial migration as applied, plus the reauth-fields
// migration when those columns are already present, so goose only runs what is
// genuinely missing.
func baselineLegacyDB(ctx context.Context, db *sql.DB) error {
	switch hasAccounts, err := tableExists(ctx, db, "accounts"); {
	case err != nil:
		return err
	case !hasAccounts:
		return nil // fresh database; goose creates everything
	}

	switch hasVersion, err := tableExists(ctx, db, "goose_db_version"); {
	case err != nil:
		return err
	case hasVersion:
		return nil // already managed by goose
	}

	applied := []int64{0, entmigrate.InitialVersion}
	switch hasReauth, err := columnExists(ctx, db, "accounts", "needs_reauth"); {
	case err != nil:
		return err
	case hasReauth:
		applied = append(applied, entmigrate.ReauthFieldsVersion)
	}

	return stampVersions(ctx, db, applied)
}

func stampVersions(ctx context.Context, db *sql.DB, versions []int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `CREATE TABLE goose_db_version (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version_id INTEGER NOT NULL,
		is_applied INTEGER NOT NULL,
		tstamp TIMESTAMP DEFAULT (datetime('now'))
	);`); err != nil {
		return err
	}
	for _, v := range versions {
		if _, err := tx.ExecContext(ctx, "INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, 1);", v); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func tableExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var found string
	err := db.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?;", name,
	).Scan(&found)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}

func columnExists(ctx context.Context, db *sql.DB, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q);", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			typ       string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
