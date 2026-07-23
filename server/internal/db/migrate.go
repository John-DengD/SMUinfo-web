package db

import (
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies all up migrations found in dir against the given DSN.
// The DSN is a standard "postgres://" connection string (shared with pgxpool);
// golang-migrate's pgx/v5 driver registers under the "pgx5://" scheme, so we
// rewrite the scheme before handing it to migrate.
func RunMigrations(dsn, dir string) error {
	migrateDSN := dsn
	if strings.HasPrefix(migrateDSN, "postgres://") {
		migrateDSN = "pgx5://" + strings.TrimPrefix(migrateDSN, "postgres://")
	} else if strings.HasPrefix(migrateDSN, "postgresql://") {
		migrateDSN = "pgx5://" + strings.TrimPrefix(migrateDSN, "postgresql://")
	}
	m, err := migrate.New("file://"+dir, migrateDSN)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
