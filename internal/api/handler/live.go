package handler

import (
	"net/http"

	"github.com/herros27/digital-scoring/internal/store"
)

type LiveHandler struct {
	ScoreStore *store.MatchScoreStore
	MatchStore *store.MatchStore
}

// GET /api/v1/live/scoreboard/{arena_id}  — no auth, untuk display publik
func (h *LiveHandler) Scoreboard(w http.ResponseWriter, r *http.Request) {
	arenaID := r.PathValue("arena_id")
	_ = arenaID
	// ponytail: untuk sekarang return placeholder
	// nanti ambil match aktif di arena → scores-nya
	Ok(w, map[string]string{"status": "no active match"})
}
