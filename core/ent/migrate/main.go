//go:build ignore

// Command migrate generates versioned, goose-format migration files by diffing
// the Ent schema against an in-memory SQLite dev database.
//
//	go run -mod=mod ./ent/migrate -name add_something   # create a migration
//	go run -mod=mod ./ent/migrate -checksum             # rebuild atlas.sum
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	atlasmigrate "ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqltool"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"
	"modernc.org/sqlite"

	"github.com/AvengeMedia/dankcalendar/core/ent/migrate"

	_ "ariga.io/atlas/sql/sqlite"
)

// devURL is a throwaway in-memory SQLite database Atlas replays migrations into
// to compute the diff. It is pure Go (modernc), so no external engine is needed.
const devURL = "sqlite://file?mode=memory&_fk=1"

// Atlas opens its SQLite dev database with the database/sql driver named
// "sqlite3"; alias the pure-Go modernc driver under that name so generation
// needs no CGO toolchain. Atlas passes mattn-style params modernc ignores, so a
// connection hook turns on foreign keys to satisfy Atlas's pragma check.
func init() {
	sqlite.RegisterConnectionHook(func(conn sqlite.ExecQuerierContext, _ string) error {
		_, err := conn.ExecContext(context.Background(), "PRAGMA foreign_keys=ON;", nil)
		return err
	})

	tmp, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		panic(fmt.Sprintf("alias modernc sqlite driver: %v", err))
	}
	sql.Register("sqlite3", tmp.Driver())
	_ = tmp.Close()
}

func main() {
	ctx := context.Background()

	_, thisFile, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(thisFile), "migrations")

	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		fatalf("create migrations dir: %v", err)
	}

	dir, err := sqltool.NewGooseDir(migrationsDir)
	if err != nil {
		fatalf("open migrations dir: %v", err)
	}

	name := flag.String("name", "", "name of the migration to create")
	checksum := flag.Bool("checksum", false, "rebuild atlas.sum and exit")
	flag.Parse()

	if *checksum {
		sum, err := dir.Checksum()
		if err != nil {
			fatalf("compute checksum: %v", err)
		}
		if err := atlasmigrate.WriteSumFile(dir, sum); err != nil {
			fatalf("write atlas.sum: %v", err)
		}
		fmt.Println("atlas.sum rebuilt")
		return
	}

	if *name == "" {
		fmt.Println("error: -name is required")
		flag.Usage()
		os.Exit(1)
	}

	opts := []schema.MigrateOption{
		schema.WithDir(dir),
		schema.WithMigrationMode(schema.ModeReplay),
		schema.WithDialect(dialect.SQLite),
		schema.WithDropColumn(true),
		schema.WithDropIndex(true),
		schema.WithIndent("  "),
		schema.WithFormatter(sqltool.GooseFormatter),
	}

	switch err := migrate.NamedDiff(ctx, devURL, *name, opts...); {
	case err != nil:
		fatalf("generate migration: %v", err)
	default:
		fmt.Printf("migration %q written to %s\n", *name, migrationsDir)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
