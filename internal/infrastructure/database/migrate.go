package database

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
)

type Migrator struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewMigrator(db *sql.DB, logger *zap.Logger) *Migrator {
	return &Migrator{
		db:     db,
		logger: logger,
	}
}

func (m *Migrator) RunMigrations(migrationsPath string) error {
	m.logger.Info("Running database migrations", zap.String("path", migrationsPath))

	driver, err := postgres.WithInstance(m.db, &postgres.Config{})
	if err != nil {
		m.logger.Error("Failed to create migration driver", zap.Error(err))
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	migration, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		m.logger.Error("Failed to create migration instance")
		return fmt.Errorf("failed to create migration isntance: %w", err)
	}

	if err := migration.Up(); err != nil {
		if err == migrate.ErrNoChange {
			m.logger.Info("No new migrations to apply")
			return nil
		}
		m.logger.Error("Migration failed", zap.Error(err))
		return fmt.Errorf("migration failed: %w", err)
	}

	version, dirty, err := migration.Version()
	if err != nil {
		m.logger.Warn("Failed to get migration version", zap.Error(err))
	} else {
		m.logger.Info(
			"Migrations applied successfully",
			zap.Uint("version", version),
			zap.Bool("dirty", dirty),
		)
	}

	return nil
}

func (m *Migrator) RollbackMigration(migrationsPath string) error {
	m.logger.Info("Rolling back last migration", zap.String("path", migrationsPath))

	driver, err := postgres.WithInstance(m.db, &postgres.Config{})
	if err != nil {
		m.logger.Error("Failed to create migration driver", zap.Error(err))
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	migration, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		m.logger.Error("Failed to create migration instance")
		return fmt.Errorf("failed to create migration isntance: %w", err)
	}

	if err := migration.Steps(-1); err != nil {
		m.logger.Error("Rollback failed", zap.Error(err))
		return fmt.Errorf("rollback failed: %w", err)
	}

	m.logger.Info("Rollback complete successfully")
	return nil
}
