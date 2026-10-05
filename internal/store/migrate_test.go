package store

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/herros27/digital-scoring/internal/config"
	"github.com/herros27/digital-scoring/migrations"
	"github.com/pressly/goose/v3"
)

func TestEmbeddedMigrations(t *testing.T) {
	// 1. Verify FS has all 16 migration files
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		t.Fatalf("failed to read embedded migrations dir: %v", err)
	}

	sqlCount := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sql") {
			sqlCount++
			content, err := fs.ReadFile(migrations.FS, entry.Name())
			if err != nil {
				t.Fatalf("failed to read file %s: %v", entry.Name(), err)
			}
			if len(content) == 0 {
				t.Errorf("migration file %s is empty", entry.Name())
			}
			if !strings.HasPrefix(string(content), "-- +goose Up") {
				t.Errorf("migration file %s does not start with '-- +goose Up'", entry.Name())
			}
		}
	}

	if sqlCount != 17 {
		t.Fatalf("expected 17 SQL migration files, found %d", sqlCount)
	}

	// 2. Verify goose can collect and sequence all 17 migrations
	goose.SetBaseFS(migrations.FS)
	migrationsList, err := goose.CollectMigrations(".", 0, goose.MaxVersion)
	if err != nil {
		t.Fatalf("failed to collect migrations: %v", err)
	}

	if len(migrationsList) != 17 {
		t.Fatalf("expected 17 collected migrations, got %d", len(migrationsList))
	}

	for i, m := range migrationsList {
		expectedVersion := int64(i + 1)
		if m.Version != expectedVersion {
			t.Errorf("migration at index %d has version %d, expected %d", i, m.Version, expectedVersion)
		}
	}
}

func TestRunMigrations_NilDB(t *testing.T) {
	// Should return an error and not panic when DB is nil
	err := RunMigrations(nil)
	if err == nil {
		t.Fatal("expected error with nil db, got nil")
	}
}

func TestConfig_AutoMigrateDefault(t *testing.T) {
	cfg := config.Load()
	if !cfg.AutoMigrate {
		t.Errorf("expected AutoMigrate to default to true, got false")
	}
}
