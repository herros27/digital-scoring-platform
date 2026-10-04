package handler

import (
	"errors"
	"net/http"

	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/cache"
	"github.com/herros27/digital-scoring/internal/consensus"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

type MatchHandler struct {
	Service    *service.MatchService
	HubManager *hub.Manager
	Cache      *cache.Cache
}

// GET /api/v1/competitions/{comp_id}/matches?page=1&limit=20&status=running
func (h *MatchHandler) List(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	page, limit, _ := ParsePagination(r)
	status := r.URL.Query().Get("status")

	matches, total, err := h.Service.GetMatchesByCompetitionPaged(compID, status, page, limit)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	OkPaged(w, matches, page, limit, total)
}

// GET /api/v1/competitions/{comp_id}/matches/{id}
func (h *MatchHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	m, err := h.Service.GetMatchByID(id)
	if err != nil {
		Fail(w, http.StatusNotFound, "NOT_FOUND", "match not found", nil)
		return
	}
	Ok(w, m)
}

// POST /api/v1/competitions/{comp_id}/matches
func (h *MatchHandler) Create(w http.ResponseWriter, r *http.Request) {
	var m store.Match
	if err := Decode(r, &m); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	m.CompetitionID = r.PathValue("comp_id")
	m.Status = "idle"
	m.CurrentRound = 1
	if err := h.Service.CreateMatch(&m); err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
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
		Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing token", nil)
		return
	}
	claims, err := auth.ParseToken(tokenStr)
	if err != nil {
		Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return
	}
	// pastikan token scoped ke arena yang benar
	if claims.ArenaID != arenaID {
		Fail(w, http.StatusForbidden, "FORBIDDEN", "token arena mismatch", nil)
		return
	}

	h.HubManager.GetOrCreate(arenaID).HandleWSWithSeat(w, r, claims.JudgePinID, hub.Role(claims.Role), claims.SeatNumber)
}

// POST /api/v1/competitions/{comp_id}/matches/{id}/load-arena
// Body: { "arena_id": "uuid", "ruleset_path": "rulesets/ipsi_2024.json" }
func (h *MatchHandler) LoadArena(w http.ResponseWriter, r *http.Request) {
	matchID := r.PathValue("id")

	var req struct {
		ArenaID     string `json:"arena_id" validate:"required"`
		RulesetPath string `json:"ruleset_path"`
	}

	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	if err := h.Service.LoadMatchToArena(matchID, req.ArenaID, req.RulesetPath); err != nil {
		if errors.Is(err, domain.ErrArenaBusy) {
			Fail(w, http.StatusConflict, "ARENA_BUSY", err.Error(), nil)
			return
		}
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	Ok(w, map[string]string{
		"match_id": matchID,
		"arena_id": req.ArenaID,
		"status":   "ready",
	})
}

// POST /api/v1/competitions/{comp_id}/matches/{id}/start
func (h *MatchHandler) Start(w http.ResponseWriter, r *http.Request) {
	matchID := r.PathValue("id")
	if err := h.Service.StartMatch(matchID); err != nil {
		Fail(w, http.StatusBadRequest, "INVALID_STATE_TRANSITION", err.Error(), nil)
		return
	}
	Ok(w, map[string]string{"status": "running"})
}

// POST /api/v1/competitions/{comp_id}/matches/{id}/pause
func (h *MatchHandler) Pause(w http.ResponseWriter, r *http.Request) {
	matchID := r.PathValue("id")
	if err := h.Service.PauseMatch(matchID); err != nil {
		Fail(w, http.StatusBadRequest, "INVALID_STATE_TRANSITION", err.Error(), nil)
		return
	}
	Ok(w, map[string]string{"status": "paused"})
}

// POST /api/v1/competitions/{comp_id}/matches/{id}/resume
func (h *MatchHandler) Resume(w http.ResponseWriter, r *http.Request) {
	matchID := r.PathValue("id")
	if err := h.Service.ResumeMatch(matchID); err != nil {
		Fail(w, http.StatusBadRequest, "INVALID_STATE_TRANSITION", err.Error(), nil)
		return
	}
	Ok(w, map[string]string{"status": "running"})
}

// POST /api/v1/competitions/{comp_id}/matches/{id}/finish
func (h *MatchHandler) Finish(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	matchID := r.PathValue("id")
	if err := h.Service.FinishMatch(matchID); err != nil {
		Fail(w, http.StatusBadRequest, "INVALID_STATE_TRANSITION", err.Error(), nil)
		return
	}
	if h.Cache != nil {
		h.Cache.InvalidateCompetition(compID)
	}
	Ok(w, map[string]string{"status": "finished"})
}

// POST /api/v1/competitions/{comp_id}/matches/{id}/cancel
func (h *MatchHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	matchID := r.PathValue("id")
	if err := h.Service.CancelMatch(matchID); err != nil {
		Fail(w, http.StatusBadRequest, "INVALID_STATE_TRANSITION", err.Error(), nil)
		return
	}
	if h.Cache != nil {
		h.Cache.InvalidateCompetition(compID)
	}
	Ok(w, map[string]string{"status": "cancelled"})
}

// GET /api/v1/competitions/{comp_id}/matches/{id}/scores
func (h *MatchHandler) GetScores(w http.ResponseWriter, r *http.Request) {
	matchID := r.PathValue("id")
	detail, err := h.Service.GetMatchScores(matchID)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, detail)
}

// GET /api/v1/competitions/{comp_id}/matches/{id}/events
func (h *MatchHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	matchID := r.PathValue("id")
	events, err := h.Service.GetMatchEvents(matchID)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, events)
}

// POST /api/v1/competitions/{comp_id}/matches/{id}/scores
func (h *MatchHandler) SubmitScore(w http.ResponseWriter, r *http.Request) {
	matchID := r.PathValue("id")
	idempotencyKey := r.Header.Get("Idempotency-Key")

	var req struct {
		EventID        string `json:"event_id"`
		IdempotencyKey string `json:"idempotency_key"`
		JudgeID        string `json:"judge_id" validate:"required"`
		ScoreType      string `json:"score_type" validate:"required"`
		Target         string `json:"target" validate:"required"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	if idempotencyKey == "" {
		idempotencyKey = req.IdempotencyKey
	}
	if idempotencyKey == "" {
		idempotencyKey = req.EventID
	}

	input := consensus.JudgeInput{
		JudgeID:   req.JudgeID,
		ScoreType: consensus.ScoreType(req.ScoreType),
		Target:    consensus.Target(req.Target),
	}

	res, isDuplicate, err := h.Service.SubmitScoreInput(matchID, input, idempotencyKey)
	if err != nil {
		Fail(w, http.StatusBadRequest, "SCORING_ERROR", err.Error(), nil)
		return
	}

	if isDuplicate {
		Ok(w, map[string]any{
			"status":    "duplicate_ignored",
			"duplicate": true,
		})
		return
	}

	Ok(w, map[string]any{
		"status": "accepted",
		"result": res,
	})
}
