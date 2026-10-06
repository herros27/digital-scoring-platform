package config

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestConfig_Defaults(t *testing.T) {
	// Clear env vars to test defaults
	os.Unsetenv("PORT")
	os.Unsetenv("ENV")
	os.Unsetenv("DB_URL")
	os.Unsetenv("JWT_SECRET")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("expected default Port=8080, got %s", cfg.Port)
	}
	if cfg.Env != "development" {
		t.Errorf("expected default Env=development, got %s", cfg.Env)
	}
	if cfg.IsProduction() {
		t.Errorf("expected IsProduction=false for development")
	}
	if cfg.JWTSecret != "dev-secret-change-in-prod" {
		t.Errorf("expected default JWTSecret, got %s", cfg.JWTSecret)
	}
	if cfg.DBURL != "postgres://scoring_user@localhost:5432/digital_scoring?sslmode=disable" {
		t.Errorf("expected default DBURL, got %s", cfg.DBURL)
	}
}

func TestConfig_EnvOverride(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("ENV", "production")
	t.Setenv("DB_URL", "postgres://custom:5432/scoring")
	t.Setenv("JWT_SECRET", "super-secret-key-123")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("expected Port=9090, got %s", cfg.Port)
	}
	if cfg.Env != "production" {
		t.Errorf("expected Env=production, got %s", cfg.Env)
	}
	if !cfg.IsProduction() {
		t.Errorf("expected IsProduction=true")
	}
	if cfg.DBURL != "postgres://custom:5432/scoring" {
		t.Errorf("expected overridden DBURL, got %s", cfg.DBURL)
	}
	if cfg.JWTSecret != "super-secret-key-123" {
		t.Errorf("expected overridden JWTSecret, got %s", cfg.JWTSecret)
	}
}

func TestConfig_DotEnvFile(t *testing.T) {
	// Create temporary .env file
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	content := "PORT=7070\nENV=staging\nJWT_SECRET=from-dotenv-file\n"
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test .env file: %v", err)
	}

	os.Unsetenv("PORT")
	os.Unsetenv("ENV")
	os.Unsetenv("JWT_SECRET")

	cfg := Load(envPath)

	if cfg.Port != "7070" {
		t.Errorf("expected Port=7070 from .env file, got %s", cfg.Port)
	}
	if cfg.Env != "staging" {
		t.Errorf("expected Env=staging from .env file, got %s", cfg.Env)
	}
	if cfg.JWTSecret != "from-dotenv-file" {
		t.Errorf("expected JWTSecret from .env file, got %s", cfg.JWTSecret)
	}
}

// .env.example harus memuat setiap variabel yang dibaca Config, agar dokumentasi konfigurasi tidak basi.
func TestEnvExample_ListsEveryConfigVariable(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".env.example"))
	if err != nil {
		t.Fatalf(".env.example tidak ditemukan: %v", err)
	}
	for _, key := range []string{"PORT", "ENV", "DB_URL", "JWT_SECRET", "AUTO_MIGRATE", "SEED_DATA", "POSTGRES_USER", "POSTGRES_DB"} {
		if !regexp.MustCompile(`(?m)^` + key + `=`).Match(data) {
			t.Errorf(".env.example tidak memuat %s", key)
		}
	}
}
