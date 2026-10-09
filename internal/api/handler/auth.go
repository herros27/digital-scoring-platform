package handler

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/herros27/digital-scoring/internal/auth"
)

type AuthHandler struct {
	Pins *auth.PinStore

	// Guard (opsional) mengunci penebak PIN secara berlapis: per perangkat, per IP, dan global. ClientIP menentukan
	// IP klien (bawaan: alamat sambungan). Lihat auth.PinGuard untuk alasan tiap lapisan.
	Guard    *auth.PinGuard
	ClientIP func(*http.Request) string
}

// deviceIDPattern bentuk ID perangkat yang diterima dari header X-Device-Id (UUID acak yang dibuat tablet sekali).
// ID ini hanya untuk keadilan antar tablet di IP yang sama, bukan batas keamanan (klien bisa menggantinya).
var deviceIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)

func deviceID(r *http.Request) string {
	if id := r.Header.Get("X-Device-Id"); deviceIDPattern.MatchString(id) {
		return id
	}
	return ""
}

func (h *AuthHandler) clientIP(r *http.Request) string {
	if h.ClientIP != nil {
		return h.ClientIP(r)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// POST /api/v1/auth/judge-login
// Body: { "pin" } — turnamen, gelanggang, role, dan kursi ditentukan server dari PIN.
func (h *AuthHandler) JudgeLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PIN string `json:"pin" validate:"required,numeric,min=6,max=8"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	var ip, device string
	if h.Guard != nil {
		ip, device = h.clientIP(r), deviceID(r)
		if locked, retryAfter, scope := h.Guard.Check(ip, device); locked {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())+1))
			msg := "terlalu banyak PIN salah dari perangkat ini, coba lagi nanti"
			if scope != "device" {
				msg = "terlalu banyak PIN salah dari jaringan ini, coba lagi nanti"
			}
			Fail(w, http.StatusTooManyRequests, "TOO_MANY_ATTEMPTS", msg, map[string]any{"scope": scope})
			return
		}
	}

	token, record, err := h.Pins.LoginWithPIN(req.PIN)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidPIN) {
			if h.Guard != nil {
				h.Guard.Fail(ip, device)
			}
			Fail(w, http.StatusUnauthorized, "INVALID_PIN", "pin not found or expired", nil)
			return
		}
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", "login gagal", nil)
		return
	}
	if h.Guard != nil {
		h.Guard.Succeed(ip, device)
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
		PIN           string `json:"pin" validate:"omitempty,numeric,min=6,max=8"`
		TTLHours      int    `json:"ttl_hours" validate:"min=0,max=720"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	if req.TTLHours <= 0 {
		req.TTLHours = 12 // default 12 jam
	}
	ttl := time.Duration(req.TTLHours) * time.Hour

	// PIN kosong -> server membuat PIN acak yang unik; PIN kustom divalidasi (digit, panjang per role)
	var record *auth.PinRecord
	var err error
	if req.PIN == "" {
		record, err = h.Pins.CreateWithRandomPIN(req.CompetitionID, req.ArenaID, req.Role, req.SeatNumber, ttl)
	} else {
		role, _ := auth.NormalizeRoleAndSeat(req.Role, req.SeatNumber)
		if verr := auth.ValidatePIN(role, req.PIN); verr != nil {
			Fail(w, http.StatusBadRequest, "BAD_REQUEST", verr.Error(), nil)
			return
		}
		record, err = h.Pins.CreatePINWithSeat(req.CompetitionID, req.ArenaID, req.Role, req.SeatNumber, req.PIN, ttl)
	}
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrArenaNotFound):
			Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		case errors.Is(err, auth.ErrPinTaken):
			Fail(w, http.StatusConflict, "PIN_TAKEN", err.Error(), nil)
		default:
			Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		}
		return
	}
	pin := record.RawPIN

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
			created, err := h.Pins.CreateWithRandomPIN(compID, arenaID, req.Role, req.Seat, 12*time.Hour)
			if err == nil && created != nil {
				seatMap[key] = *created
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
