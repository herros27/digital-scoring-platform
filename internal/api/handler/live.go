package handler

import (
	"net/http"

	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

type LiveHandler struct {
	ScoreStore *store.MatchScoreStore
	MatchStore *store.MatchStore
	Matches    *service.MatchService
}

// GET /api/v1/live/scoreboard/{arena_id}  — no auth, untuk display publik
// Mengembalikan keadaan pertandingan aktif di arena, atau {"status": "no active match"}.
func (h *LiveHandler) Scoreboard(w http.ResponseWriter, r *http.Request) {
	arenaID := r.PathValue("arena_id")
	if h.Matches != nil {
		if snap, ok := h.Matches.GetLiveSnapshot(arenaID); ok {
			Ok(w, snap)
			return
		}
	}
	Ok(w, map[string]string{"status": "no active match"})
}
