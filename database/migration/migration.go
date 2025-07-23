package migration

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"strings"
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

var (
	migrationsMutex sync.RWMutex
	migrations      []Migration = make([]Migration, 0)
)

func RegisterMigration(migration Migration) error {
	migrationsMutex.Lock()
	defer migrationsMutex.Unlock()

	// Validate migration
	if migration.Version() == "" {
		return fmt.Errorf("migration version cannot be empty")
	}
	if migration.Description() == "" {
		return fmt.Errorf("migration description cannot be empty")
	}

	// Check for duplicates
	for _, existing := range migrations {
		if existing.Version() == migration.Version() {
			return fmt.Errorf("migration with version %s already registered", migration.Version())
		}
	}

	migrations = append(migrations, migration)
	return nil
}

type Migration interface {
	Version() string
	Description() string // Add description
	Up(ctx context.Context) (string, []any)
	Down(ctx context.Context) (string, []any)
}

type MigrationStatus struct {
	Version     string
	Description string
	AppliedAt   *time.Time
	Status      string // "applied", "pending", "failed"
}

type Migrator interface {
	Up(ctx context.Context) error
	Down(ctx context.Context, target string) error
	DryRun(ctx context.Context) ([]string, error)
	Status(ctx context.Context) ([]MigrationStatus, error)
	CurrentVersion(ctx context.Context) (string, error)
	UpWithProgress(ctx context.Context, callback ProgressCallback) error
	Verify(ctx context.Context) error
}

type MigratorConfig struct {
	LockTimeout     time.Duration
	AutoRollback    bool // Rollback on failure
	VerifyChecksums bool // Verify migration content hasn't changed
	MaxRetries      int  // Retry failed migrations
}

type migrator struct {
	driver SqlDriverType
	db     *sql.DB
	config MigratorConfig
}

func NewMigrator(driver SqlDriverType, db *sql.DB) Migrator {
	return &migrator{
		driver: driver,
		db:     db,
		config: MigratorConfig{
			LockTimeout:  30 * time.Second,
			AutoRollback: false, // Default: don't auto-rollback
		},
	}
}

func NewMigratorWithConfig(driver SqlDriverType, db *sql.DB, config MigratorConfig) Migrator {
	if config.LockTimeout == 0 {
		config.LockTimeout = 30 * time.Second
	}

	return &migrator{
		driver: driver,
		db:     db,
		config: config,
	}
}

func (m *migrator) acquireLock(ctx context.Context) (func(), error) {
	var lockSQL string
	var releaseLockSQL string

	switch m.driver {
	case SqlDriverTypeMysql:
		lockSQL = "SELECT GET_LOCK('migration_lock', ?)"
		releaseLockSQL = "SELECT RELEASE_LOCK('migration_lock')"
	case SqlDriverTypePostgresql:
		lockSQL = "SELECT pg_advisory_lock(123456789)"
		releaseLockSQL = "SELECT pg_advisory_unlock(123456789)"
	case SqlDriverTypeSqlite:
		// SQLite doesn't support advisory locks, use table-based locking
		lockSQL = `INSERT OR IGNORE INTO migration_lock (id, acquired_at) VALUES (1, ?)`
		releaseLockSQL = `DELETE FROM migration_lock WHERE id = 1`

		// Ensure lock table exists
		if _, err := m.db.ExecContext(ctx, `
            CREATE TABLE IF NOT EXISTS migration_lock (
                id INTEGER PRIMARY KEY,
                acquired_at DATETIME NOT NULL
            )`); err != nil {
			return nil, fmt.Errorf("failed to create lock table: %w", err)
		}
	}

	// Acquire lock with timeout
	lockCtx, cancel := context.WithTimeout(ctx, m.config.LockTimeout)
	defer cancel()

	switch m.driver {
	case SqlDriverTypeMysql:
		var lockResult int
		if err := m.db.QueryRowContext(lockCtx, lockSQL, int(m.config.LockTimeout.Seconds())).Scan(&lockResult); err != nil {
			return nil, fmt.Errorf("failed to acquire migration lock: %w", err)
		}
		if lockResult != 1 {
			return nil, fmt.Errorf("could not acquire migration lock (another migration may be running)")
		}
	case SqlDriverTypePostgresql:
		if _, err := m.db.ExecContext(lockCtx, lockSQL); err != nil {
			return nil, fmt.Errorf("failed to acquire migration lock: %w", err)
		}
	case SqlDriverTypeSqlite:
		result, err := m.db.ExecContext(lockCtx, lockSQL, time.Now().UTC())
		if err != nil {
			return nil, fmt.Errorf("failed to acquire migration lock: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("failed to check lock result: %w", err)
		}
		if affected == 0 {
			return nil, fmt.Errorf("could not acquire migration lock (another migration may be running)")
		}
	}

	// Return release function
	return func() {
		if _, err := m.db.ExecContext(context.Background(), releaseLockSQL); err != nil {
			slog.Error("failed to release migration lock", "error", err)
		}
	}, nil
}

func (m *migrator) getMigrations() []Migration {
	migrationsMutex.RLock()
	defer migrationsMutex.RUnlock()
	result := make([]Migration, len(migrations))
	copy(result, migrations)
	return result
}

func (m *migrator) Up(ctx context.Context) error {
	releaseLock, err := m.acquireLock(ctx)
	if err != nil {
		return err
	}
	defer releaseLock()

	ctx = context.WithValue(ctx, CtxSqlDriverType, m.driver)

	if err := m.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	allMigrations := m.getMigrations() // Use thread-safe getter
	if len(allMigrations) == 0 {
		slog.Info("No migrations to apply")
		return nil
	}

	appliedVersions, err := m.getAppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}

	// Sort migrations by version (ascending for up)
	sortedMigrations := make([]Migration, len(allMigrations))
	copy(sortedMigrations, allMigrations)
	slices.SortFunc(sortedMigrations, func(a, b Migration) int {
		return strings.Compare(a.Version(), b.Version())
	})

	transaction, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if rollbackErr := transaction.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			slog.Error("rollback failed", "error", rollbackErr)
		}
	}()

	appliedCount := 0
	appliedInThisRun := []string{}

	for _, migration := range sortedMigrations {
		if appliedVersions[migration.Version()] {
			continue // Already applied
		}

		slog.Info("Applying migration", "version", migration.Version(), "description", migration.Description())

		query, arguments := migration.Up(ctx)
		if _, err := transaction.ExecContext(ctx, query, arguments...); err != nil {
			slog.Error("Migration failed", "version", migration.Version(), "error", err)

			// Rollback this transaction
			if rollbackErr := transaction.Rollback(); rollbackErr != nil {
				slog.Error("Failed to rollback transaction", "error", rollbackErr)
			}

			// Optionally: auto-rollback successfully applied migrations from this run
			if len(appliedInThisRun) > 0 {
				slog.Info("Rolling back migrations applied in this run", "count", len(appliedInThisRun))
				m.rollbackRecentMigrations(ctx, appliedInThisRun)
			}

			return fmt.Errorf("failed to apply migration %s: %w", migration.Version(), err)
		}

		// Record successful migration
		if _, err := transaction.ExecContext(ctx,
			`INSERT INTO migration_history (version, description, performed_at, direction) VALUES (?, ?, ?, ?)`,
			migration.Version(), migration.Description(), time.Now().UTC(), "up"); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migration.Version(), err)
		}

		appliedInThisRun = append(appliedInThisRun, migration.Version())
		appliedCount++
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("failed to commit migrations: %w", err)
	}

	slog.Info("Migrations completed", "applied", appliedCount)
	return nil
}

type ProgressCallback func(current, total int, migration Migration, err error)

func (m *migrator) UpWithProgress(ctx context.Context, callback ProgressCallback) error {
	releaseLock, err := m.acquireLock(ctx)
	if err != nil {
		return err
	}
	defer releaseLock()

	ctx = context.WithValue(ctx, CtxSqlDriverType, m.driver)

	if err := m.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	allMigrations := m.getMigrations() // Use thread-safe getter
	if len(allMigrations) == 0 {
		slog.Info("No migrations to apply")
		return nil
	}

	appliedVersions, err := m.getAppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}

	// Sort migrations by version (ascending for up)
	sortedMigrations := make([]Migration, len(allMigrations))
	copy(sortedMigrations, allMigrations)
	slices.SortFunc(sortedMigrations, func(a, b Migration) int {
		return strings.Compare(a.Version(), b.Version())
	})

	transaction, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if rollbackErr := transaction.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			slog.Error("rollback failed", "error", rollbackErr)
		}
	}()

	appliedCount := 0
	appliedInThisRun := []string{}

	for i, migration := range sortedMigrations {
		if appliedVersions[migration.Version()] {
			continue // Already applied
		}

		slog.Info("Applying migration", "version", migration.Version(), "description", migration.Description())

		query, arguments := migration.Up(ctx)
		_, err := transaction.ExecContext(ctx, query, arguments...)

		if callback != nil {
			callback(i+1, len(sortedMigrations), migration, err)
		}

		if err != nil {
			slog.Error("Migration failed", "version", migration.Version(), "error", err)

			// Rollback this transaction
			if rollbackErr := transaction.Rollback(); rollbackErr != nil {
				slog.Error("Failed to rollback transaction", "error", rollbackErr)
			}

			// Optionally: auto-rollback successfully applied migrations from this run
			if len(appliedInThisRun) > 0 {
				slog.Info("Rolling back migrations applied in this run", "count", len(appliedInThisRun))
				m.rollbackRecentMigrations(ctx, appliedInThisRun)
			}

			return fmt.Errorf("failed to apply migration %s: %w", migration.Version(), err)
		}

		// Record successful migration
		if _, err := transaction.ExecContext(ctx,
			`INSERT INTO migration_history (version, description, performed_at, direction) VALUES (?, ?, ?, ?)`,
			migration.Version(), migration.Description(), time.Now().UTC(), "up"); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migration.Version(), err)
		}

		appliedInThisRun = append(appliedInThisRun, migration.Version())
		appliedCount++
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("failed to commit migrations: %w", err)
	}

	slog.Info("Migrations completed", "applied", appliedCount)
	return nil
}

func (m *migrator) Down(ctx context.Context, target string) error {
	ctx = context.WithValue(ctx, CtxSqlDriverType, m.driver)

	if err := m.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	appliedVersions, err := m.getAppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}

	// Fix: Use getMigrations() instead of direct access
	allMigrations := m.getMigrations()
	sortedMigrations := make([]Migration, len(allMigrations))
	copy(sortedMigrations, allMigrations)
	slices.SortFunc(sortedMigrations, func(a, b Migration) int {
		return strings.Compare(b.Version(), a.Version()) // Reverse order
	})

	transaction, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if rollbackErr := transaction.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			slog.Error("rollback failed", "error", rollbackErr)
		}
	}()

	rolledBackCount := 0
	for _, migration := range sortedMigrations {
		// Stop when we reach the target version
		if migration.Version() == target {
			break
		}

		// Only rollback applied migrations
		if !appliedVersions[migration.Version()] {
			continue
		}

		slog.Info("Rolling back migration", "version", migration.Version(), "description", migration.Description())

		query, arguments := migration.Down(ctx)
		if _, err := transaction.ExecContext(ctx, query, arguments...); err != nil {
			return fmt.Errorf("failed to rollback migration %s: %w", migration.Version(), err)
		}

		// Remove from migration history
		if _, err := transaction.ExecContext(ctx,
			`DELETE FROM migration_history WHERE version = ?`,
			migration.Version()); err != nil {
			return fmt.Errorf("failed to remove migration record %s: %w", migration.Version(), err)
		}

		rolledBackCount++
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollbacks: %w", err)
	}

	slog.Info("Rollback completed", "rolled_back", rolledBackCount)
	return nil
}

func (m *migrator) Status(ctx context.Context) ([]MigrationStatus, error) {
	if err := m.ensureMigrationTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to create migration table: %w", err)
	}

	appliedVersions, err := m.getAppliedVersionsWithTime(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied versions: %w", err)
	}

	var statuses []MigrationStatus
	// Fix: Use getMigrations() instead of direct access
	allMigrations := m.getMigrations()
	for _, migration := range allMigrations {
		status := MigrationStatus{
			Version:     migration.Version(),
			Description: migration.Description(),
			Status:      "pending",
		}

		if appliedAt, exists := appliedVersions[migration.Version()]; exists {
			status.AppliedAt = &appliedAt
			status.Status = "applied"
		}

		statuses = append(statuses, status)
	}

	// Sort by version
	slices.SortFunc(statuses, func(a, b MigrationStatus) int {
		return strings.Compare(a.Version, b.Version)
	})

	return statuses, nil
}

func (m *migrator) CurrentVersion(ctx context.Context) (string, error) {
	if err := m.ensureMigrationTable(ctx); err != nil {
		return "", fmt.Errorf("failed to create migration table: %w", err)
	}

	var currentVersion string
	row := m.db.QueryRowContext(ctx,
		`SELECT version FROM migration_history ORDER BY performed_at DESC LIMIT 1`)

	if err := row.Scan(&currentVersion); err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No migrations applied yet
		}
		return "", fmt.Errorf("failed to get current version: %w", err)
	}

	return currentVersion, nil
}

func (m *migrator) ensureMigrationTable(ctx context.Context) error {
	var createTableSQL string

	switch m.driver {
	case SqlDriverTypeSqlite:
		createTableSQL = `
            CREATE TABLE IF NOT EXISTS migration_history (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                version TEXT NOT NULL,
                description TEXT,
                performed_at DATETIME NOT NULL,
                direction TEXT NOT NULL DEFAULT 'up'
            )`
	case SqlDriverTypeMysql:
		createTableSQL = `
            CREATE TABLE IF NOT EXISTS migration_history (
                id INT AUTO_INCREMENT PRIMARY KEY,
                version VARCHAR(255) NOT NULL,
                description TEXT,
                performed_at TIMESTAMP NOT NULL,
                direction VARCHAR(10) NOT NULL DEFAULT 'up',
                INDEX idx_version (version),
                INDEX idx_performed_at (performed_at)
            )`
	case SqlDriverTypePostgresql:
		createTableSQL = `
            CREATE TABLE IF NOT EXISTS migration_history (
                id SERIAL PRIMARY KEY,
                version VARCHAR(255) NOT NULL,
                description TEXT,
                performed_at TIMESTAMP NOT NULL,
                direction VARCHAR(10) NOT NULL DEFAULT 'up'
            );
            CREATE INDEX IF NOT EXISTS idx_migration_version ON migration_history(version);
            CREATE INDEX IF NOT EXISTS idx_migration_performed_at ON migration_history(performed_at);`
	default:
		return fmt.Errorf("unsupported database driver: %v", m.driver)
	}

	if _, err := m.db.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	return nil
}

func (m *migrator) getAppliedVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT DISTINCT version FROM migration_history`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", "error", err)
		}
	}()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

func (m *migrator) getAppliedVersionsWithTime(ctx context.Context) (map[string]time.Time, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT version, performed_at FROM migration_history ORDER BY performed_at DESC`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", "error", err)
		}
	}()

	applied := make(map[string]time.Time)
	for rows.Next() {
		var version string
		var performedAt time.Time
		if err := rows.Scan(&version, &performedAt); err != nil {
			return nil, err
		}
		// Keep only the most recent application
		if _, exists := applied[version]; !exists {
			applied[version] = performedAt
		}
	}

	return applied, rows.Err()
}

func (m *migrator) DryRun(ctx context.Context) ([]string, error) {
	ctx = context.WithValue(ctx, CtxSqlDriverType, m.driver)

	if err := m.ensureMigrationTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to create migration table: %w", err)
	}

	allMigrations := m.getMigrations()
	appliedVersions, err := m.getAppliedVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied versions: %w", err)
	}

	var queries []string
	for _, migration := range allMigrations {
		if !appliedVersions[migration.Version()] {
			query, _ := migration.Up(ctx)
			queries = append(queries, fmt.Sprintf("-- Migration: %s - %s\n%s",
				migration.Version(), migration.Description(), query))
		}
	}

	return queries, nil
}

func (m *migrator) Verify(ctx context.Context) error {
	appliedVersions, err := m.getAppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}

	allMigrations := m.getMigrations()
	migrationMap := make(map[string]Migration)
	for _, migration := range allMigrations {
		migrationMap[migration.Version()] = migration
	}

	// Check for orphaned migrations in DB
	for version := range appliedVersions {
		if _, exists := migrationMap[version]; !exists {
			slog.Warn("Found applied migration not in codebase", "version", version)
		}
	}

	return nil
}

func (m *migrator) rollbackRecentMigrations(ctx context.Context, versions []string) {
	if !m.config.AutoRollback {
		return // Skip if auto-rollback is disabled
	}

	// Create a new transaction for rollback
	rollbackTx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("Failed to begin rollback transaction", "error", err)
		return
	}
	defer func() {
		if err := rollbackTx.Rollback(); err != nil {
			slog.Error("Failed to rollback transaction", "error", err)
		}
	}()

	allMigrations := m.getMigrations()
	migrationMap := make(map[string]Migration)
	for _, migration := range allMigrations {
		migrationMap[migration.Version()] = migration
	}

	// Rollback in reverse order
	for i := len(versions) - 1; i >= 0; i-- {
		version := versions[i]
		migration, exists := migrationMap[version]
		if !exists {
			slog.Error("Cannot find migration to rollback", "version", version)
			continue
		}

		slog.Info("Auto-rolling back migration", "version", version)

		query, arguments := migration.Down(ctx)
		if _, err := rollbackTx.ExecContext(ctx, query, arguments...); err != nil {
			slog.Error("Failed to rollback migration", "version", version, "error", err)
			continue
		}

		// Remove from migration history
		if _, err := rollbackTx.ExecContext(ctx,
			`DELETE FROM migration_history WHERE version = ?`, version); err != nil {
			slog.Error("Failed to remove migration record", "version", version, "error", err)
		}
	}

	if err := rollbackTx.Commit(); err != nil {
		slog.Error("Failed to commit rollback transaction", "error", err)
	}
}
