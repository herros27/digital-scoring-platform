package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func NewDB() *sql.DB{
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		dsn = "postgres://scoring_user@localhost:5432/digital_scoring?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil{
		log.Fatalf("failed to open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}

	fmt.Println("✅ DB connected")
	return db
}