package migration

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"slices"
	"sync"
	"time"
)

type key string

const CtxSqlDriverType key = "sql_driver_type"

type SqlDriverType int

const (
	SqlDriverTypeSqlite SqlDriverType = iota + 1
	SqlDriverTypeMysql
	SqlDriverTypePostgresql
)

var migrationsMutex sync.RWMutex
var migrations []Migration = make([]Migration, 0)

func RegisterMigration(migration Migration) {
	migrationsMutex.Lock()
	defer migrationsMutex.Unlock()
	migrations = append(migrations, migration)
}

type Migration interface {
	Version() string
	Up(ctx context.Context) (string, []any)
	Down(ctx context.Context) (string, []any)
}

type Migrator interface {
	Up(ctx context.Context) error
	Down(ctx context.Context) error
}

type migrator struct {
	driver SqlDriverType
	db     *sql.DB
}

func NewMigrator(driver SqlDriverType, db *sql.DB) Migrator {
	return &migrator{
		driver: driver,
		db:     db,
	}
}

func (migrator *migrator) Up(ctx context.Context) error {
	ctx = context.WithValue(ctx, CtxSqlDriverType, migrator.driver)

	// Nothing to do
	if len(migrations) == 0 {
		return nil
	}

	current, err := migrator.CurrentVersion(ctx)
	if err != nil {
		return err
	}

	slices.SortFunc(migrations, func(a, b Migration) int {
		if a.Version() > b.Version() {
			return 1
		}
		return -1
	})

	transaction, err := migrator.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if rollbackErr := transaction.Rollback(); rollbackErr != nil {
			log.Printf("rollback failed: %v", err)
		}
	}()

	for _, migration := range migrations {
		if current >= migration.Version() {
			continue
		}

		query, arguments := migration.Up(ctx)
		if _, err := transaction.ExecContext(ctx, query, arguments...); err != nil {
			return nil
		}

		if _, err := transaction.ExecContext(ctx, `INSERT INTO migration_history (version, performed_at) VALUES (?,?)`, migration.Version(), time.Now().UTC()); err != nil {
			return nil
		}
	}

	return transaction.Commit()
}

func (migrator *migrator) Down(ctx context.Context) error {
	ctx = context.WithValue(ctx, CtxSqlDriverType, migrator.driver)

	// Nothing to do
	if len(migrations) == 0 {
		return nil
	}

	current, err := migrator.CurrentVersion(ctx)
	if err != nil {
		return err
	}

	slices.SortFunc(migrations, func(a, b Migration) int {
		if a.Version() > b.Version() {
			return 1
		}
		return -1
	})

	transaction, err := migrator.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if rollbackErr := transaction.Rollback(); rollbackErr != nil {
			slog.Error("rollback failed", "error", rollbackErr)
		}
	}()

	for _, migration := range migrations {
		if current >= migration.Version() {
			continue
		}

		query, arguments := migration.Down(ctx)
		if _, err := transaction.ExecContext(ctx, query, arguments...); err != nil {
			return nil
		}

		if _, err := transaction.ExecContext(ctx, `'DELETE FROM migration_history WHERE version = ?`, migration.Version()); err != nil {
			return nil
		}

	}

	return transaction.Commit()
}

func (migrator *migrator) CurrentVersion(ctx context.Context) (string, error) {
	var currentVersion string
	row := migrator.db.QueryRowContext(ctx, `SELECT version FROM migration_history WHERE performed_at = MAX(performed_at) LIMIT 1`)
	if err := row.Scan(&currentVersion); err != nil {
		return currentVersion, err
	}

	return currentVersion, nil
}
