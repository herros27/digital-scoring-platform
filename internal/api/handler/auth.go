package handler

import (
	"errors"
	"fmt"
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
		if errors.Is(err, auth.ErrArenaNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
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

// GET /api/v1/auth/pins (admin only)
// Query: ?competition_id=...&arena_id=...
func (h *AuthHandler) ListPINs(w http.ResponseWriter, r *http.Request) {
	compID := r.URL.Query().Get("competition_id")
	arenaID := r.URL.Query().Get("arena_id")
	if compID == "" || arenaID == "" {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", "competition_id and arena_id query parameters are required", nil)
		return
	}

	records, err := h.Pins.GetByCompAndArena(compID, arenaID)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	seatMap := make(map[string]auth.PinRecord)
	for _, rec := range records {
		key := fmt.Sprintf("%s-%d", rec.Role, rec.SeatNumber)
		seatMap[key] = rec
	}

	requiredSeats := []struct {
		Role string
		Seat int
	}{
		{"judge", 1},
		{"judge", 2},
		{"judge", 3},
		{"judge", 4},
		{"judge", 5},
		{"wasit", 0},
	}

	for _, req := range requiredSeats {
		key := fmt.Sprintf("%s-%d", req.Role, req.Seat)
		rec, exists := seatMap[key]
		if !exists || rec.RawPIN == "" {
			newPin, err := auth.GenerateRandomPIN()
			if err == nil {
				created, err := h.Pins.CreatePINWithSeat(compID, arenaID, req.Role, req.Seat, newPin, 12*time.Hour)
				if err == nil && created != nil {
					seatMap[key] = *created
				}
			}
		}
	}

	type PinItem struct {
		ID            string    `json:"id"`
		CompetitionID string    `json:"competition_id"`
		ArenaID       string    `json:"arena_id"`
		Role          string    `json:"role"`
		SeatNumber    int       `json:"seat_number"`
		PIN           string    `json:"pin"`
		ExpiresAt     time.Time `json:"expires_at"`
	}

	items := make([]PinItem, 0, len(seatMap))
	// Urutkan sesuai kursi Juri 1-5 lalu Wasit
	for _, req := range requiredSeats {
		key := fmt.Sprintf("%s-%d", req.Role, req.Seat)
		if r, ok := seatMap[key]; ok {
			items = append(items, PinItem{
				ID:            r.ID,
				CompetitionID: r.CompetitionID,
				ArenaID:       r.ArenaID,
				Role:          r.Role,
				SeatNumber:    r.SeatNumber,
				PIN:           r.RawPIN,
				ExpiresAt:     r.ExpiresAt,
			})
		}
	}

	Ok(w, items)
}

// POST /api/v1/auth/logout
// ponytail: stateless JWT logout. Token destruction is client-side (clearing localStorage/Pinia).
// Upgrade path: Redis/in-memory token blacklist (JTI revocation) if immediate invalidation is required.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	Ok(w, map[string]string{
		"message": "successfully logged out",
	})
}
