package handler

import (
	"net/http"

	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

type MatchHandler struct {
    Service      *service.MatchService
    HubManager *hub.Manager
}

// GET /api/v1/competitions/{comp_id}/matches
func (h *MatchHandler) List(w http.ResponseWriter, r *http.Request) {
    // ponytail: list by competition belum ada di store, pakai match_events nanti
    // untuk sekarang return empty
    Ok(w, []any{})
}

// GET /api/v1/competitions/{comp_id}/matches/{id}
func (h *MatchHandler) Get(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    m, err := h.Service.GetMatchByID(id)
    if err != nil {
        Fail(w, http.StatusNotFound, "NOT_FOUND", "match not found")
        return
    }
    Ok(w, m)
}

// POST /api/v1/competitions/{comp_id}/matches
func (h *MatchHandler) Create(w http.ResponseWriter, r *http.Request) {
    var m store.Match
    if err := Decode(r, &m); err != nil {
        Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
        return
    }
    m.CompetitionID = r.PathValue("comp_id")
    m.Status = "idle"
    m.CurrentRound = 1
    if err := h.Service.CreateMatch(&m); err != nil {
        Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
        return
    }
    w.WriteHeader(http.StatusCreated)
    Ok(w, m)
}

// GET /ws/arena/{arena_id}?token=eyJ...
// WS upgrade — auth via query param (WS tidak support custom header saat handshake)
func (h *MatchHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
    arenaID := r.PathValue("arena_id")

    tokenStr := r.URL.Query().Get("token")
    if tokenStr == "" {
        Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing token")
        return
    }
    claims, err := auth.ParseToken(tokenStr)
    if err != nil {
        Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
        return
    }
    // pastikan token scoped ke arena yang benar
    if claims.ArenaID != arenaID {
        Fail(w, http.StatusForbidden, "FORBIDDEN", "token arena mismatch")
        return
    }

    h.HubManager.GetOrCreate(arenaID).HandleWS(w, r, claims.JudgePinID, hub.Role(claims.Role))
}

// POST /api/v1/competitions/{comp_id}/matches/{id}/load-arena
// Body: { "arena_id": "uuid", "ruleset_path": "rulesets/ipsi_2024.json" }
func (h *MatchHandler) LoadArena(w http.ResponseWriter, r *http.Request) {
	matchID := r.PathValue("id")

	var req struct {
		ArenaID     string `json:"arena_id"`
		RulesetPath string `json:"ruleset_path"`
	}

	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	if req.ArenaID == "" {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", "arena_id is required")
		return
	}

	if err := h.Service.LoadMatchToArena(matchID, req.ArenaID, req.RulesetPath); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	Ok(w, map[string]string{
		"match_id": matchID,
		"arena_id": req.ArenaID,
		"status":   "ready",
	})
}