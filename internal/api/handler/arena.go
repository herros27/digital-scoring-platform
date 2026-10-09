package handler

import (
	"net/http"
	"sort"

	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/service"
)

type ArenaHandler struct {
	Service *service.ArenaService
	// Hubs sumber kehadiran perangkat (WebSocket) per gelanggang; nil bila tidak tersedia (semua dianggap offline).
	Hubs *hub.Manager
}

// arenaPresenceDTO kehadiran perangkat di satu gelanggang saat ini, dihitung dari koneksi WebSocket yang hidup.
type arenaPresenceDTO struct {
	ArenaID     string `json:"arena_id"`
	JudgeSeats  []int  `json:"judge_seats"` // nomor kursi juri yang online, terurut
	Referees    int    `json:"referees"`
	Scoreboards int    `json:"scoreboards"`
	Admins      int    `json:"admins"`
}

// GET /api/v1/competitions/{comp_id}/arenas/{id}/presence
// Perangkat yang benar-benar terhubung ke gelanggang (juri per kursi, wasit, scoreboard, admin). Gelanggang tanpa hub
// aktif (belum ada yang pernah terhubung) menghasilkan semua nol, bukan galat.
func (h *ArenaHandler) Presence(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	arena, err := h.Service.GetArenaByID(id)
	if err != nil || arena == nil || arena.CompetitionID != r.PathValue("comp_id") {
		Fail(w, http.StatusNotFound, "NOT_FOUND", "gelanggang tidak ditemukan pada kejuaraan ini", nil)
		return
	}
	out := arenaPresenceDTO{ArenaID: id, JudgeSeats: []int{}}
	if h.Hubs != nil {
		if hb, err := h.Hubs.Get(id); err == nil {
			p := hb.Presence()
			for seat, online := range p.JudgeSeats {
				if online {
					out.JudgeSeats = append(out.JudgeSeats, seat)
				}
			}
			sort.Ints(out.JudgeSeats)
			out.Referees, out.Scoreboards, out.Admins = p.Referees, p.Scoreboards, p.Admins
		}
	}
	Ok(w, out)
}

// GET /api/v1/competitions/{comp_id}/arenas
func (h *ArenaHandler) List(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	list, err := h.Service.GetArenasByCompetition(compID)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, list)
}

// GET /api/v1/competitions/{comp_id}/arenas/{id}
func (h *ArenaHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	arena, err := h.Service.GetArenaByID(id)
	if err != nil {
		Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		return
	}
	Ok(w, arena)
}

// POST /api/v1/competitions/{comp_id}/arenas
func (h *ArenaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name" validate:"required,min=1,max=100"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	compID := r.PathValue("comp_id")
	arena, err := h.Service.CreateArena(compID, req.Name)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	w.WriteHeader(http.StatusCreated)
	Ok(w, arena)
}

// PUT /api/v1/competitions/{comp_id}/arenas/{id}
func (h *ArenaHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name string `json:"name" validate:"required,min=1,max=100"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	arena, err := h.Service.UpdateArena(id, req.Name)
	if err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	Ok(w, arena)
}

// DELETE /api/v1/competitions/{comp_id}/arenas/{id}
func (h *ArenaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.Service.DeleteArena(id); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	Ok(w, map[string]string{"status": "deleted"})
}
