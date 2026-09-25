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

// POST /api/v1/competitions/{comp_id}/arenas
func (h *ArenaHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req struct{
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