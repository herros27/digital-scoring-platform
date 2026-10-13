package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/herros27/digital-scoring/internal/logger"
	"github.com/herros27/digital-scoring/migrations"
	"github.com/pressly/goose/v3"
)

// RunMigrations applies all pending database migrations using embedded SQL files.
// ponytail: pressly/goose Provider runs directly on *sql.DB with embed.FS, zero external CLI needed.
func RunMigrations(db *sql.DB) error {
	var opts []goose.ProviderOption
	if logger.Log != nil {
		opts = append(opts, goose.WithSlog(logger.Log))
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS, opts...)
	if err != nil {
		return fmt.Errorf("failed to initialize goose provider: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if logger.Log != nil {
		logger.Log.Info("checking database migrations...")
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	if len(results) == 0 {
		if logger.Log != nil {
			logger.Log.Info("✅ database schema is already up to date")
		}
		return nil
	}

	if logger.Log != nil {
		for _, res := range results {
			logger.Log.Info("applied migration",
				"file", res.Source.Path,
				"version", res.Source.Version,
				"duration", res.Duration.String(),
			)
		}
		logger.Log.Info("✅ database migrations applied successfully", "applied_count", len(results))
	}

	return nil
}
