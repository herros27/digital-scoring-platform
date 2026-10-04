package config

import (
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

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
