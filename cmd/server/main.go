package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/herros27/digital-scoring/internal/api"
	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/config"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/logger"
	"github.com/herros27/digital-scoring/internal/store"
)

func main() {
	// 1. Centralized Configuration
	cfg := config.Load()

	// 2. Inisialisasi Logger & Auth JWT Secret
	logger.Init(cfg.IsProduction())
	if err := cfg.Validate(); err != nil {
		logger.Log.Error("konfigurasi tidak aman, server tidak dijalankan", "error", err,
			"petunjuk", "buat nilai acak: openssl rand -base64 48, lalu isi JWT_SECRET di .env")
		os.Exit(1)
	}
	auth.SetJWTSecret(cfg.JWTSecret)

	// 3. Hubungkan Database & Jalankan Migrasi Otomatis
	db := store.NewDB(cfg.DBURL)
	defer db.Close()

	if cfg.AutoMigrate {
		if err := store.RunMigrations(db); err != nil {
			logger.Log.Error("database migration failed", "error", err)
			os.Exit(1)
		}
	}

	if cfg.SeedData {
		if summary, err := store.SeedData(db); err != nil {
			logger.Log.Error("database seed failed", "error", err)
		} else {
			logger.Log.Info("🌱 database seed applied successfully",
				"officials", summary.Officials,
				"contingents", summary.Contingents,
				"athletes", summary.Athletes,
				"participants", summary.Participants,
				"matches", summary.Matches,
				"judge_pins", summary.JudgePins,
			)
		}
	}

	// 4. Hub Manager & Router
	hubManager := hub.NewManager()
	router := api.NewRouter(db, hubManager)

	// 5. Production HTTP Server Configuration
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 6. Graceful Shutdown listener
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Log.Info("🚀 Server starting", "port", ":"+cfg.Port, "env", cfg.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for SIGINT / SIGTERM signal
	<-ctx.Done()
	logger.Log.Info("🛑 Shutting down server gracefully...")

	// Beri waktu 10 detik untuk menyelesaikan request & WebSocket connection aktif
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("server forced to shutdown", "error", err)
	}

	logger.Log.Info("✅ Server stopped cleanly")
}
