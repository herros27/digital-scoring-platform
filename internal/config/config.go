package config

import (
	"os"
)

type Config struct {
	Port      string
	Env       string
	DBURL     string
	JWTSecret string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://scoring_user@localhost:5432/digital_scoring?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-prod" // ponytail: override via env in production
	}

	return &Config{
		Port:      port,
		Env:       env,
		DBURL:     dbURL,
		JWTSecret: jwtSecret,
	}
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
