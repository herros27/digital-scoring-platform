package config

import (
	"fmt"
	"log"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Port        string `env:"PORT" envDefault:"8080"`
	Env         string `env:"ENV" envDefault:"development"`
	DBURL       string `env:"DB_URL" envDefault:"postgres://scoring_user@localhost:5432/digital_scoring?sslmode=disable"`
	JWTSecret   string `env:"JWT_SECRET" envDefault:"dev-secret-change-in-prod"`
	AutoMigrate bool   `env:"AUTO_MIGRATE" envDefault:"true"`
	SeedData    bool   `env:"SEED_DATA" envDefault:"false"`
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

// Validate memeriksa konfigurasi yang berbahaya bila dibiarkan pada mode production. Mode lain selalu lolos.
// JWT_SECRET kosong atau nilai contoh berarti siapa pun dapat memalsukan token login (termasuk token admin).
func (c *Config) Validate() error {
	if !c.IsProduction() {
		return nil
	}
	switch {
	case c.JWTSecret == "":
		return fmt.Errorf("JWT_SECRET kosong pada ENV=production")
	case placeholderSecrets[c.JWTSecret]:
		return fmt.Errorf("JWT_SECRET masih nilai contoh dari repo; isi dengan nilai acak milik Anda sendiri pada ENV=production")
	case len(c.JWTSecret) < MinProductionSecretLen:
		return fmt.Errorf("JWT_SECRET terlalu pendek (%d karakter); minimal %d karakter pada ENV=production", len(c.JWTSecret), MinProductionSecretLen)
	}
	return nil
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
