package config

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Port        string `env:"PORT" envDefault:"8080"`
	Env         string `env:"ENV" envDefault:"production"`
	DBURL       string `env:"DB_URL" envDefault:"postgres://scoring_user@localhost:5432/digital_scoring?sslmode=disable"`
	JWTSecret   string `env:"JWT_SECRET" envDefault:"dev-secret-change-in-prod"`
	AutoMigrate bool   `env:"AUTO_MIGRATE" envDefault:"true"`
	SeedData    bool   `env:"SEED_DATA" envDefault:"false"`
	// Admin pertama (produksi tanpa seed demo): dibuat saat start bila belum ada admin. Lihat OfficialService.EnsureAdmin.
	AdminEmail    string `env:"ADMIN_EMAIL" envDefault:""`
	AdminPassword string `env:"ADMIN_PASSWORD" envDefault:""`
	AdminName     string `env:"ADMIN_NAME" envDefault:"Administrator"`
	// MetricsToken: bila diisi, /metrics wajib header Authorization: Bearer <token>; kosong = hanya loopback.
	MetricsToken string `env:"METRICS_TOKEN" envDefault:""`
}

// Load loads configuration from optional .env files and environment variables into Config.
// ponytail: godotenv loads .env file if present, env.Parse unmarshals with defaults and type coercion.
func Load(filenames ...string) *Config {
	if len(filenames) > 0 {
		_ = godotenv.Load(filenames...)
	} else {
		// Ignore error if .env doesn't exist (e.g. in docker/production where env is injected)
		_ = godotenv.Load()
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		log.Fatalf("failed to parse environment configuration: %v", err)
	}

	return cfg
}

// MinProductionSecretLen panjang minimal JWT_SECRET pada mode production.
const MinProductionSecretLen = 32

// placeholderSecrets nilai contoh yang tercantum di repo (kode, .env.example, docker-compose) dan karena itu
// dapat ditebak siapa pun.
var placeholderSecrets = map[string]bool{
	"dev-secret-change-in-prod":                true,
	"change-this-in-production-secret-key-123": true,
}

// placeholderDBPasswords password contoh di .env.example; terlihat publik, jadi tidak boleh dipakai di produksi.
var placeholderDBPasswords = map[string]bool{
	"dev_password_placeholder": true,
}

// Validate memeriksa konfigurasi yang berbahaya bila dibiarkan di luar development. Hanya ENV=development (persis)
// yang boleh memakai kunci contoh dan seed; ENV kosong, "prod", atau "Production" diperlakukan sebagai produksi,
// sehingga salah ketik tidak membuka jalan bagi kunci yang tercantum di repo.
// JWT_SECRET kosong atau nilai contoh berarti siapa pun dapat memalsukan token login (termasuk token admin).
// SEED_DATA membuat akun demo (admin@silat.id / DemoPass_2026 dst., tertulis di docs/CREDENTIALS.md) dan SETIAP start
// mengembalikan password akun itu ke nilai publik tersebut.
func (c *Config) Validate() error {
	if c.Env == "development" {
		return nil
	}
	if dbURLWithoutPassword(c.DBURL) {
		return fmt.Errorf("DB_URL tidak berisi password database (ENV=%q); isi POSTGRES_PASSWORD di .env.prod, atau pakai PGPASSWORD/password= di DB_URL", c.Env)
	}
	if dbURLUsesPlaceholderPassword(c.DBURL) {
		return fmt.Errorf("DB_URL memakai password contoh dari .env.example; isi POSTGRES_PASSWORD dengan nilai acak milik Anda sendiri (ENV=%q)", c.Env)
	}
	if c.SeedData {
		return fmt.Errorf("SEED_DATA=true menanam akun demo berpassword publik dan hanya boleh dengan ENV=development (ENV=%q)", c.Env)
	}
	switch {
	case c.JWTSecret == "":
		return fmt.Errorf("JWT_SECRET kosong (ENV=%q; kunci bawaan hanya untuk ENV=development)", c.Env)
	case placeholderSecrets[c.JWTSecret]:
		return fmt.Errorf("JWT_SECRET masih nilai contoh dari repo; isi dengan nilai acak milik Anda sendiri (ENV=%q; kunci contoh hanya untuk ENV=development)", c.Env)
	case len(c.JWTSecret) < MinProductionSecretLen:
		return fmt.Errorf("JWT_SECRET terlalu pendek (%d karakter); minimal %d karakter (ENV=%q; kunci pendek hanya untuk ENV=development)", len(c.JWTSecret), MinProductionSecretLen, c.Env)
	}
	return nil
}

// dbURLWithoutPassword true bila DB_URL berbentuk URL dengan host tetapi tanpa password (di userinfo, parameter
// `password=`, maupun PGPASSWORD). Bentuk lain (soket Unix tanpa host, string keyword/value) tidak dinilai.
func dbURLWithoutPassword(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	if _, ok := u.User.Password(); ok {
		return false
	}
	return u.Query().Get("password") == "" && os.Getenv("PGPASSWORD") == ""
}

func dbURLUsesPlaceholderPassword(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	pw, _ := u.User.Password()
	return placeholderDBPasswords[pw]
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
