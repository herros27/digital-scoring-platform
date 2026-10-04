package handler

import (
	"net/http"

	"github.com/herros27/digital-scoring/internal/service"
)

type ArenaHandler struct {
	Service *service.ArenaService
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
		Name string `json:"name"`
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
		Name string `json:"name"`
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