package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/cache"
	"github.com/herros27/digital-scoring/internal/consensus"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/orchestrator"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

type MatchHandler struct {
	Service    *service.MatchService
	HubManager *hub.Manager
	Cache      *cache.Cache
}

// arenaAllowed memastikan token PIN (terikat satu gelanggang) hanya mengontrol gelanggangnya sendiri,
// sama seperti handshake WebSocket. Login email admin/dewan/wasit tidak terikat gelanggang dan boleh di mana saja.
// Mengembalikan false setelah menulis respons 403.
func arenaAllowed(w http.ResponseWriter, r *http.Request, arenaID string) bool {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil || claims.ArenaID == "" || claims.ArenaID == arenaID {
		return true
	}
	Fail(w, http.StatusForbidden, "FORBIDDEN", "token terikat ke gelanggang lain", nil)
	return false
}

// matchArenaAllowed seperti arenaAllowed untuk partai yang ditentukan oleh id pada path.
func (h *MatchHandler) matchArenaAllowed(w http.ResponseWriter, r *http.Request, matchID string) bool {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil || claims.ArenaID == "" {
		return true
	}
	// partai yang tidak diketahui dibiarkan lolos: handler memberi galat "tidak dimuat/tidak ditemukan" sendiri
	arenaID := h.Service.ArenaForMatch(matchID)
	if arenaID == "" {
		return true
	}
	return arenaAllowed(w, r, arenaID)
}

// GET /api/v1/competitions/{comp_id}/matches?page=1&limit=20&status=running
func (h *MatchHandler) List(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	page, limit, _ := ParsePagination(r)
	status := r.URL.Query().Get("status")

	var (
		matches []store.Match
		total   int
		err     error
	)
	if arenaID := r.URL.Query().Get("arena_id"); arenaID != "" {
		matches, total, err = h.Service.GetMatchesByArenaPaged(compID, arenaID, status, page, limit)
	} else {
		matches, total, err = h.Service.GetMatchesByCompetitionPaged(compID, status, page, limit)
	}
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
		if errors.Is(err, domain.ErrInvalidInput) {
			Fail(w, http.StatusBadRequest, "INVALID_INPUT", strings.TrimPrefix(err.Error(), domain.ErrInvalidInput.Error()+": "), nil)
			return
		}
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	if h.Cache != nil {
		h.Cache.InvalidateCompetition(m.CompetitionID)
	}
	w.WriteHeader(http.StatusCreated)
	Ok(w, m)
}

// GET /ws/arena/{arena_id}[?token=eyJ...]
// WS upgrade — auth via query param (WS tidak support custom header saat handshake).
//
//   - Tanpa token: penonton publik (scoreboard TV), role "scoreboard". Hanya bisa
//     menerima siaran dan sync_request; semua perintah ditolak orchestrator.
//     Hub tidak dibuat untuk penonton, sehingga arena yang belum punya pertandingan
//     aktif dijawab 404 (klien akan mencoba lagi).
//   - Dengan token PIN: arena pada token harus sama dengan arena yang dituju.
//   - Dengan token login email (tanpa arena): hanya admin, dewan, dan wasit.
func (h *MatchHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	arenaID := r.PathValue("arena_id")

	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		spectatorHub, err := h.HubManager.Get(arenaID)
		if err != nil {
			Fail(w, http.StatusNotFound, "NOT_FOUND", "no active match in this arena", nil)
			return
		}
		spectatorHub.HandleWSWithSeat(w, r, "scoreboard-"+uuid.NewString(), hub.RoleScoreboard, 0)
		return
	}

	claims, err := auth.ParseToken(tokenStr)
	if err != nil {
		Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return
	}
	if auth.Revoked(claims) {
		Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "sesi dicabut, silakan login ulang", nil)
		return
	}
	// Token PIN selalu terikat ke satu arena dan harus cocok. Token login email
	// (tanpa arena) hanya diterima untuk petugas: admin, dewan, wasit.
	unscopedOperator := claims.ArenaID == "" && hub.IsOperatorClaim(claims.Role)
	if claims.ArenaID != arenaID && !unscopedOperator {
		Fail(w, http.StatusForbidden, "FORBIDDEN", "token arena mismatch", nil)
		return
	}

	// Token login email (admin/dewan/wasit) tidak membawa pin_id; beri ID unik per koneksi
	clientID := claims.JudgePinID
	if clientID == "" {
		clientID = "official-" + uuid.NewString()
	}

	h.HubManager.GetOrCreate(arenaID).HandleWSWithSeat(w, r, clientID, hub.RoleFromClaim(claims.Role), claims.SeatNumber)
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

	if !arenaAllowed(w, r, req.ArenaID) {
		return
	}
	if err := h.Service.LoadMatchToArena(matchID, req.ArenaID, req.RulesetPath); err != nil {
		if errors.Is(err, domain.ErrArenaBusy) {
			Fail(w, http.StatusConflict, "ARENA_BUSY", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			Fail(w, http.StatusForbidden, "FORBIDDEN", strings.TrimPrefix(err.Error(), domain.ErrForbidden.Error()+": "), nil)
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
	if !h.matchArenaAllowed(w, r, matchID) {
		return
	}
	if err := h.Service.StartMatch(matchID); err != nil {
		Fail(w, http.StatusBadRequest, "INVALID_STATE_TRANSITION", err.Error(), nil)
		return
	}
	Ok(w, map[string]string{"status": "running"})
}

// POST /api/v1/competitions/{comp_id}/matches/{id}/pause
func (h *MatchHandler) Pause(w http.ResponseWriter, r *http.Request) {
	matchID := r.PathValue("id")
	if !h.matchArenaAllowed(w, r, matchID) {
		return
	}
	if err := h.Service.PauseMatch(matchID); err != nil {
		Fail(w, http.StatusBadRequest, "INVALID_STATE_TRANSITION", err.Error(), nil)
		return
	}
	Ok(w, map[string]string{"status": "paused"})
}

// POST /api/v1/competitions/{comp_id}/matches/{id}/resume
func (h *MatchHandler) Resume(w http.ResponseWriter, r *http.Request) {
	matchID := r.PathValue("id")
	if !h.matchArenaAllowed(w, r, matchID) {
		return
	}
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
	if !h.matchArenaAllowed(w, r, matchID) {
		return
	}

	// Body opsional: {"winner":"red"|"blue"} = keputusan wasit (wajib bila skor seri)
	var req struct {
		Winner string `json:"winner" validate:"omitempty,oneof=red blue"`
		// Method dasar keputusan seri: extra_round | lighter_weight | coin_toss (kosong = keputusan wasit)
		Method string `json:"method" validate:"omitempty,oneof=referee_decision extra_round lighter_weight coin_toss"`
	}
	if r.ContentLength != 0 {
		if err := Decode(r, &req); err != nil {
			Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
			return
		}
	}

	if err := h.Service.FinishMatchWithDecision(matchID, req.Winner, req.Method); err != nil {
		code := "INVALID_STATE_TRANSITION"
		if errors.Is(err, orchestrator.ErrTieNeedsDecision) {
			code = "TIE_NEEDS_DECISION"
		}
		Fail(w, http.StatusBadRequest, code, err.Error(), nil)
		return
	}
	if h.Cache != nil {
		h.Cache.InvalidateCompetition(compID)
	}
	Ok(w, map[string]string{"status": "finished"})
}

// POST /api/v1/competitions/{comp_id}/matches/{id}/outcome
// Body: {"outcome":"ko|tko|walkover|withdrawal","winner":"red|blue","note":"..."}
func (h *MatchHandler) Outcome(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	matchID := r.PathValue("id")
	if !h.matchArenaAllowed(w, r, matchID) {
		return
	}
	var req struct {
		Outcome string `json:"outcome" validate:"required,oneof=ko tko walkover withdrawal"`
		Winner  string `json:"winner" validate:"required,oneof=red blue"`
		Note    string `json:"note" validate:"omitempty,max=200"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	if err := h.Service.DeclareOutcome(matchID, req.Outcome, req.Winner, req.Note); err != nil {
		Fail(w, http.StatusBadRequest, "INVALID_STATE_TRANSITION", err.Error(), nil)
		return
	}
	if h.Cache != nil {
		h.Cache.InvalidateCompetition(compID)
	}
	Ok(w, map[string]string{"status": "finished", "result_type": req.Outcome})
}

// POST /api/v1/competitions/{comp_id}/matches/{id}/cancel
func (h *MatchHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	matchID := r.PathValue("id")
	if !h.matchArenaAllowed(w, r, matchID) {
		return
	}
	if err := h.Service.CancelMatch(matchID); err != nil {
		Fail(w, http.StatusBadRequest, "INVALID_STATE_TRANSITION", err.Error(), nil)
		return
	}
	if h.Cache != nil {
		h.Cache.InvalidateCompetition(compID)
	}
	Ok(w, map[string]string{"status": "cancelled"})
}

// POST /api/v1/competitions/{comp_id}/matches/{id}/reschedule
func (h *MatchHandler) Reschedule(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	matchID := r.PathValue("id")
	if !h.matchArenaAllowed(w, r, matchID) {
		return
	}
	if err := h.Service.RescheduleMatch(matchID); err != nil {
		status, code := http.StatusBadRequest, "INVALID_STATE_TRANSITION"
		if errors.Is(err, domain.ErrMatchNotFound) {
			status, code = http.StatusNotFound, "NOT_FOUND"
		}
		Fail(w, status, code, err.Error(), nil)
		return
	}
	if h.Cache != nil {
		h.Cache.InvalidateCompetition(compID)
	}
	Ok(w, map[string]string{"status": "idle"})
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
// Fallback REST untuk juri (WebSocket tetap jalur utama). Hanya token PIN juri (dijaga router) dan
// identitas juri SELALU diambil dari token, bukan dari body: kalau tidak, siapa pun yang login bisa
// mengirim suara atas nama juri lain dan mencapai konsensus. Token juga harus milik gelanggang tempat
// partai ini dimuat, sama seperti aturan handshake WebSocket.
func (h *MatchHandler) SubmitScore(w http.ResponseWriter, r *http.Request) {
	matchID := r.PathValue("id")
	idempotencyKey := r.Header.Get("Idempotency-Key")

	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil || claims.Role != "judge" || claims.JudgePinID == "" {
		Fail(w, http.StatusForbidden, "FORBIDDEN", "hanya juri yang boleh mengirim skor", nil)
		return
	}

	var req struct {
		EventID        string `json:"event_id"`
		IdempotencyKey string `json:"idempotency_key"`
		// JudgeID diabaikan (dipertahankan agar klien lama tetap valid); identitas dari token.
		JudgeID   string `json:"judge_id"`
		ScoreType string `json:"score_type" validate:"required"`
		Target    string `json:"target" validate:"required"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	orch, err := h.Service.GetOrchestrator(matchID)
	if err != nil {
		Fail(w, http.StatusBadRequest, "SCORING_ERROR", err.Error(), nil)
		return
	}
	if orch.GetArenaID() != claims.ArenaID {
		Fail(w, http.StatusForbidden, "FORBIDDEN", "token juri bukan untuk gelanggang partai ini", nil)
		return
	}

	if idempotencyKey == "" {
		idempotencyKey = req.IdempotencyKey
	}
	if idempotencyKey == "" {
		idempotencyKey = req.EventID
	}

	input := consensus.JudgeInput{
		JudgeID:    claims.JudgePinID,
		Role:       consensus.RoleJudge,
		SeatNumber: claims.SeatNumber,
		ScoreType:  consensus.ScoreType(req.ScoreType),
		Target:     consensus.Target(req.Target),
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
