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

// GET /api/v1/live/arenas  — no auth, daftar gelanggang yang sedang punya pertandingan aktif.
func (h *LiveHandler) Arenas(w http.ResponseWriter, r *http.Request) {
	if h.Matches == nil {
		Ok(w, []map[string]any{})
		return
	}
	Ok(w, h.Matches.ListLiveArenas())
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
	resp := map[string]string{"status": "no active match"}
	if h.Matches != nil {
		resp["arena_name"], resp["competition_name"] = h.Matches.ArenaContext(arenaID)
	}
	Ok(w, resp)
}
