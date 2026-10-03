package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// NewDB initializes the PostgreSQL connection pool.
// Accepts an optional DSN string; otherwise falls back to DB_URL environment variable.
func NewDB(dsnOverride ...string) *sql.DB {
	dsn := ""
	if len(dsnOverride) > 0 && dsnOverride[0] != "" {
		dsn = dsnOverride[0]
	} else {
		dsn = os.Getenv("DB_URL")
	}

	if dsn == "" {
		dsn = "postgres://scoring_user@localhost:5432/digital_scoring?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

	// Production connection pool tuning
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}

	fmt.Println("✅ DB connected")
	return db
}