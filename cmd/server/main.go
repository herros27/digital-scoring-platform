package main

import (
	"log"
	"net/http"
	"os"

	"github.com/herros27/digital-scoring/internal/api"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/logger"
	"github.com/herros27/digital-scoring/internal/store"
)

func main() {
	// 1. Inisialisasi Logger
	isProd := os.Getenv("ENV") == "production"
	logger.Init(isProd)

	// 2. Hubungkan Database
	db := store.NewDB()
	defer db.Close()

	// 3. Hub Manager & Router
	hubManager := hub.NewManager()
	router := api.NewRouter(db, hubManager)

	logger.Log.Info("🚀 Server starting", "port", ":8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}