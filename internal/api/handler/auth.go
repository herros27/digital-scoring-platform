package handler

import (
    "net/http"
	"time"

    "github.com/herros27/digital-scoring/internal/auth"
)

type AuthHandler struct {
    Pins *auth.PinStore
}

// POST /api/v1/auth/judge-login
// Body: { "competition_id", "arena_id", "pin" }
func (h *AuthHandler) JudgeLogin(w http.ResponseWriter, r *http.Request) {
    var req struct {
        CompetitionID string `json:"competition_id"`
        ArenaID       string `json:"arena_id"`
        PIN           string `json:"pin"`
    }
    if err := Decode(r, &req); err != nil {
        Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
        return
    }
    if req.CompetitionID == "" || req.ArenaID == "" || req.PIN == "" {
        Fail(w, http.StatusBadRequest, "BAD_REQUEST", "competition_id, arena_id, pin required", nil)
        return
    }

    token, record, err := h.Pins.LoginWithPIN(req.CompetitionID, req.ArenaID, req.PIN)
    if err != nil {
        Fail(w, http.StatusUnauthorized, "INVALID_PIN", "pin not found or expired", nil)
        return
    }

    Ok(w, map[string]any{
        "token":          token,
        "role":           record.Role,
        "arena_id":       record.ArenaID,
        "competition_id": record.CompetitionID,
    })
}

// POST /api/v1/auth/create-pin  (admin only — untuk assign PIN ke juri)
// Body: { "competition_id", "arena_id", "role", "pin", "ttl_hours" }
func (h *AuthHandler) CreatePIN(w http.ResponseWriter, r *http.Request) {
    var req struct {
        CompetitionID string `json:"competition_id"`
        ArenaID       string `json:"arena_id"`
        Role          string `json:"role"`
        PIN           string `json:"pin"`
        TTLHours      int    `json:"ttl_hours"`
    }
    if err := Decode(r, &req); err != nil {
        Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
        return
    }
    if req.TTLHours == 0 {
        req.TTLHours = 12 // default 12 jam
    }

    ttl := time.Duration(req.TTLHours) * time.Hour
    record, err := h.Pins.CreatePIN(req.CompetitionID, req.ArenaID, req.Role, req.PIN, ttl)
    if err != nil {
        Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
        return
    }
    Ok(w, record)
}