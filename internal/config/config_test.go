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
	if cfg.Env != "production" {
		t.Errorf("expected default Env=production (aman bila ENV lupa diisi), got %s", cfg.Env)
	}
	if !cfg.IsProduction() {
		t.Errorf("expected IsProduction=true by default")
	}
	if cfg.Validate() == nil {
		t.Errorf("default (ENV kosong + kunci contoh) harus ditolak Validate")
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
	// variabel opsional boleh tampil sebagai contoh yang dikomentari
	for _, key := range []string{"ADMIN_EMAIL", "ADMIN_PASSWORD", "ADMIN_NAME"} {
		if !regexp.MustCompile(`(?m)^#?` + key + `=`).Match(data) {
			t.Errorf(".env.example tidak memuat %s", key)
		}
	}
}

func TestLoad_AdminBootstrapVariables(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "ketua@lomba.id")
	t.Setenv("ADMIN_PASSWORD", "test_pass_valid_1")
	t.Setenv("ADMIN_NAME", "Ketua Panitia")
	cfg := Load("tidak-ada.env")
	if cfg.AdminEmail != "ketua@lomba.id" || cfg.AdminPassword != "test_pass_valid_1" || cfg.AdminName != "Ketua Panitia" {
		t.Errorf("admin bootstrap tidak terbaca: %+v", cfg)
	}
}

func TestValidate_DatabasePasswordRequiredOutsideDevelopment(t *testing.T) {
	strong := "mock-jwt-secret-key-for-test-32chars-ok!"
	t.Setenv("PGPASSWORD", "")
	cases := []struct {
		name, env, dbURL string
		ok               bool
	}{
		{"development tanpa password lolos", "development", "postgres://u@localhost:5432/db", true},
		{"production tanpa password ditolak", "production", "postgres://u@postgres:5432/db", false},
		{"ENV kosong tanpa password ditolak", "", "postgres://u@postgres:5432/db", false},
		{"production dengan password lolos", "production", "postgres://u:rahasia@postgres:5432/db", true},
		{"password lewat parameter lolos", "production", "postgres://u@postgres:5432/db?password=rahasia", true},
		{"soket Unix tanpa host tidak dinilai", "production", "postgres:///db?host=/var/run/postgresql", true},
		{"DB_URL kosong tidak dinilai", "production", "", true},
	}
	for _, c := range cases {
		err := (&Config{Env: c.env, JWTSecret: strong, DBURL: c.dbURL}).Validate()
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok=%v", c.name, err, c.ok)
		}
	}

	t.Setenv("PGPASSWORD", "dari-env")
	if err := (&Config{Env: "production", JWTSecret: strong, DBURL: "postgres://u@postgres:5432/db"}).Validate(); err != nil {
		t.Errorf("PGPASSWORD harus dihitung sebagai password: %v", err)
	}
}

func TestValidate_SeedDataOnlyInDevelopment(t *testing.T) {
	strong := "mock-jwt-secret-key-for-test-32chars-ok!"
	cases := []struct {
		env  string
		seed bool
		ok   bool
	}{
		{"development", true, true},
		{"production", true, false},
		{"", true, false},
		{"prod", true, false},
		{"production", false, true},
	}
	for _, c := range cases {
		err := (&Config{Env: c.env, JWTSecret: strong, SeedData: c.seed}).Validate()
		if (err == nil) != c.ok {
			t.Errorf("env=%q seed=%v: err = %v, want ok=%v", c.env, c.seed, err, c.ok)
		}
	}
}

func TestValidate_ProductionRejectsWeakJWTSecret(t *testing.T) {
	long := "mock-jwt-secret-key-for-test-32chars-ok!"
	cases := []struct {
		name   string
		env    string
		secret string
		ok     bool
	}{
		{"development apa pun lolos", "development", "dev-secret-change-in-prod", true},
		{"development kosong lolos", "development", "", true},
		{"production kosong ditolak", "production", "", false},
		{"production nilai contoh kode ditolak", "production", "dev-secret-change-in-prod", false},
		{"production nilai contoh compose ditolak", "production", "change-this-in-production-secret-key-123", false},
		{"production terlalu pendek ditolak", "production", "short-secret", false},
		{"production 31 karakter ditolak", "production", "0123456789012345678901234567890", false},
		{"production 32 karakter lolos", "production", "01234567890123456789012345678901", true},
		{"production acak panjang lolos", "production", long, true},
		{"ENV kosong dengan kunci contoh ditolak", "", "dev-secret-change-in-prod", false},
		{"ENV salah ketik dengan kunci contoh ditolak", "prod", "dev-secret-change-in-prod", false},
		{"ENV huruf besar dengan kunci contoh ditolak", "Production", "dev-secret-change-in-prod", false},
		{"staging dengan kunci pendek ditolak", "staging", "short-secret", false},
		{"ENV salah ketik dengan kunci kuat lolos", "prod", long, true},
	}
	for _, c := range cases {
		cfg := &Config{Env: c.env, JWTSecret: c.secret}
		err := cfg.Validate()
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}
