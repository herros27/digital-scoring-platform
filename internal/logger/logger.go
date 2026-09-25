package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

func Init(isProduction bool) {
	var handler slog.Handler

	if isProduction {
		// Production: Output JSON terstruktur (bagus untuk Caddy, Docker, Datadog/Grafana)
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		// Development: Output text biasa yang mudah dibaca di terminal
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	Log = slog.New(handler)
	slog.SetDefault(Log)
}

// Helper untuk match logging dengan konteks arena dan match
func MatchLogger(arenaID, matchID string) *slog.Logger {
	return Log.With(
		slog.String("arena_id", arenaID),
		slog.String("match_id", matchID),
	)
}