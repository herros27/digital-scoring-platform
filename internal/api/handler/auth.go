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
		CompetitionID string `json:"competition_id" validate:"required"`
		ArenaID       string `json:"arena_id" validate:"required"`
		PIN           string `json:"pin" validate:"required,min=4,max=10"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	token, record, err := h.Pins.LoginWithPIN(req.CompetitionID, req.ArenaID, req.PIN)
	if err != nil {
		Fail(w, http.StatusUnauthorized, "INVALID_PIN", "pin not found or expired", nil)
		return
	}

	res := map[string]any{
		"token":          token,
		"role":           record.Role,
		"arena_id":       record.ArenaID,
		"competition_id": record.CompetitionID,
	}
	if record.SeatNumber > 0 {
		res["seat_number"] = record.SeatNumber
	}
	Ok(w, res)
}

// POST /api/v1/auth/create-pin  (admin only — untuk assign PIN ke juri)
// Body: { "competition_id", "arena_id", "role", "seat_number" (opsional), "pin" (opsional), "ttl_hours" (opsional) }
func (h *AuthHandler) CreatePIN(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CompetitionID string `json:"competition_id" validate:"required"`
		ArenaID       string `json:"arena_id" validate:"required"`
		Role          string `json:"role" validate:"required,oneof=wasit juri dewan admin operator"`
		SeatNumber    int    `json:"seat_number" validate:"min=0,max=10"`
		PIN           string `json:"pin" validate:"omitempty,min=4,max=10"`
		TTLHours      int    `json:"ttl_hours" validate:"min=0,max=720"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	pin := req.PIN
	if pin == "" {
		var err error
		pin, err = auth.GenerateRandomPIN()
		if err != nil {
			Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate pin", nil)
			return
		}
	}

	if req.TTLHours <= 0 {
		req.TTLHours = 12 // default 12 jam
	}

	ttl := time.Duration(req.TTLHours) * time.Hour
	record, err := h.Pins.CreatePINWithSeat(req.CompetitionID, req.ArenaID, req.Role, req.SeatNumber, pin, ttl)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	// Kembalikan plain PIN dan expires_at agar admin/operator dapat membagikannya kepada juri
	res := map[string]any{
		"id":             record.ID,
		"competition_id": record.CompetitionID,
		"arena_id":       record.ArenaID,
		"role":           record.Role,
		"pin":            pin,
		"expires_at":     record.ExpiresAt,
	}
	if record.SeatNumber > 0 {
		res["seat_number"] = record.SeatNumber
	}
	Ok(w, res)
}
