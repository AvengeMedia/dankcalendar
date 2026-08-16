package repo

import (
	"context"
	"database/sql"
	"fmt"
	"runtime/debug"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankgo/log"
	_ "modernc.org/sqlite"
)

type Repo struct {
	client *ent.Client
}

func New(client *ent.Client) *Repo {
	return &Repo{client: client}
}

func (r *Repo) Client() *ent.Client { return r.client }

func (r *Repo) Close() error { return r.client.Close() }

// WithTx runs fn in a transaction, rolling back on error or panic.
func (r *Repo) WithTx(ctx context.Context, fn func(tx *ent.Tx) error) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if v := recover(); v != nil {
			log.Errorf("panic caught in WithTx: %v\n%s", v, debug.Stack())
			_ = tx.Rollback()
			panic(v)
		}
	}()
	if err := fn(tx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			err = fmt.Errorf("%w: rolling back transaction: %v", err, rerr)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

func Open(ctx context.Context, dsn string) (*ent.Client, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite at %q: %w", dsn, err)
	}

	// Hold an open connection before migrating so a shared-cache memory
	// database survives the migration connection closing.
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect sqlite at %q: %w", dsn, err)
	}

	if err := migrate(ctx, dsn); err != nil {
		db.Close()
		return nil, err
	}

	driver := entsql.OpenDB(dialect.SQLite, db)
	return ent.NewClient(ent.Driver(driver)), nil
}

// _time_format=sqlite makes the driver store timestamps in a format it can
// read back; without it time.Time values are written via String(), which is
// unparseable in zones whose offset has minutes, e.g. +0545 (issue #79).
func OpenFile(ctx context.Context, path string) (*ent.Client, error) {
	dsn := fmt.Sprintf("file:%s?cache=shared&_time_format=sqlite&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", path)
	return Open(ctx, dsn)
}

func OpenMemory(ctx context.Context) (*ent.Client, error) {
	return Open(ctx, "file:dankcal_test?mode=memory&cache=shared&_time_format=sqlite&_pragma=foreign_keys(ON)")
}
