package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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
//
// Migrations run on their own connection with foreign keys disabled: goose
// wraps each migration in a transaction, where SQLite silently ignores
// foreign_keys PRAGMAs, so a table-rebuild migration's DROP TABLE would
// otherwise fire ON DELETE CASCADE and wipe child rows (issue #76).
func migrate(ctx context.Context, dsn string) error {
	configureGoose()

	db, err := sql.Open("sqlite", migrationDSN(dsn))
	if err != nil {
		return fmt.Errorf("open migration connection: %w", err)
	}
	defer db.Close()

	if err := baselineLegacyDB(ctx, db); err != nil {
		return fmt.Errorf("baseline database: %w", err)
	}
	if err := goose.UpContext(ctx, db, migrationsDir); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	if err := repairLocalizedTimestamps(ctx, db); err != nil {
		return fmt.Errorf("repair timestamps: %w", err)
	}
	return nil
}

// Timestamps written before _time_format=sqlite was set on the connection used
// time.Time's String() format, e.g.
//
//	2026-08-16 12:34:56.789012345 +0545 +0545 m=+0.001234567
//
// which the driver cannot read back in zones whose offset has minutes,
// breaking every query on the row (issue #79). Rewrite such values (any
// datetime holding more than two space-separated fields) to "date time+hh:mm"
// by keeping the first two fields and reformatting the offset field. Runs on
// every startup; already-normalized values hold at most one space and are
// never matched.
func repairLocalizedTimestamps(ctx context.Context, db *sql.DB) error {
	const repairStmt = "UPDATE `%[1]s` SET `%[2]s` = " +
		"substr(`%[2]s`, 1, 10) || ' ' || " +
		"substr(`%[2]s`, 12, instr(substr(`%[2]s`, 12), ' ') - 1) || " +
		"substr(`%[2]s`, 12 + instr(substr(`%[2]s`, 12), ' '), 3) || ':' || " +
		"substr(`%[2]s`, 15 + instr(substr(`%[2]s`, 12), ' '), 2) " +
		"WHERE `%[2]s` LIKE '%% %% %%';"

	tables, err := listTables(ctx, db)
	if err != nil {
		return err
	}
	for _, table := range tables {
		columns, err := timeColumns(ctx, db, table)
		if err != nil {
			return err
		}
		for _, column := range columns {
			if _, err := db.ExecContext(ctx, fmt.Sprintf(repairStmt, table, column)); err != nil {
				return fmt.Errorf("repair %s.%s: %w", table, column, err)
			}
		}
	}
	return nil
}

func listTables(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%';")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func timeColumns(ctx context.Context, db *sql.DB, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(`%s`);", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var (
			cid         int
			name, typ   string
			notNull, pk int
			defaultVal  sql.NullString
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultVal, &pk); err != nil {
			return nil, err
		}
		switch strings.ToUpper(typ) {
		case "DATETIME", "TIMESTAMP", "DATE":
			columns = append(columns, name)
		}
	}
	return columns, rows.Err()
}

// migrationDSN replaces any foreign_keys pragma with foreign_keys(OFF). The
// driver does not honor a later duplicate pragma, so the original must be
// dropped rather than overridden.
func migrationDSN(dsn string) string {
	base, query, found := strings.Cut(dsn, "?")
	if !found {
		return dsn + "?_pragma=foreign_keys(OFF)"
	}

	params := strings.Split(query, "&")
	kept := params[:0]
	for _, param := range params {
		if strings.HasPrefix(param, "_pragma=foreign_keys") {
			continue
		}
		kept = append(kept, param)
	}
	kept = append(kept, "_pragma=foreign_keys(OFF)")
	return base + "?" + strings.Join(kept, "&")
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
