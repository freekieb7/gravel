package migration

import (
	"context"
)

func init() {
	panic(RegisterMigration(historyMigration{}))
}

type historyMigration struct {
}

func (migration historyMigration) Version() string {
	return "0_create_migration_history"
}

func (migration historyMigration) Description() string {
	return "Create migration history table"
}

func (migration historyMigration) Up(ctx context.Context) (string, []any) {
	a, ok := ctx.Value(CtxSqlDriverType).(SqlDriverType)
	if !ok {
		panic("migration: ctx sql driver type of unknown type")
	}

	var query string
	switch a {
	case SqlDriverTypeSqlite:
		{
			query = `
				CREATE TABLE migration_history (
					version TEXT NOT NULL,
					performed_at INTEGER NOT NULL
				);
			`
		}
	case SqlDriverTypePostgresql:
		{
			query = `
				CREATE TABLE migration_history (
					rowid SERIAL NOT NULL,
					version VARCHAR(255) NOT NULL,
					performed_at timestamp NOT NULL,
					PRIMARY KEY (rowid)
				);
			`
		}
	case SqlDriverTypeMysql:
		{
			query = `
				CREATE TABLE migration_history (
					rowid INT UNSIGNED NOT NULL AUTO_INCREMENT,
					version VARCHAR(255) NOT NULL,
					performed_at timestamp NOT NULL,
					PRIMARY KEY (rowid)
				);
			`
		}
	default:
		{
			panic("migration: unknown driver type")
		}
	}

	return query, make([]any, 0)
}

func (migration historyMigration) Down(ctx context.Context) (string, []any) {
	return `DROP TABLE migration_history;`, make([]any, 0)
}
