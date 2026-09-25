package handler

import (
    "net/http"

    "github.com/herros27/digital-scoring/internal/service"
    "github.com/herros27/digital-scoring/internal/store"
)

type CompetitionHandler struct {
    Service *service.CompetitionService
}

// GET /api/v1/competitions
func (h *CompetitionHandler) List(w http.ResponseWriter, r *http.Request) {
    list, err := h.Service.GetAllCompetitions()
    if err != nil {
        Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
        return
    }
    Ok(w, list)
}

// GET /api/v1/competitions/{id}
func (h *CompetitionHandler) Get(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    c, err := h.Service.GetCompetitionByID(id)
    if err != nil {
        Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
        return
    }
    Ok(w, c)
}

// POST /api/v1/competitions
func (h *CompetitionHandler) Create(w http.ResponseWriter, r *http.Request) {
    var c store.Competition
    if err := Decode(r, &c); err != nil {
        Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
        return
    }

    // Biarkan Service yang menangani default value dan aturan bisnisnya
    if err := h.Service.CreateCompetition(&c); err != nil {
        Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
        return
    }

    w.WriteHeader(http.StatusCreated)
    Ok(w, c)
}

// PUT /api/v1/competitions/{id}/activate
func (h *CompetitionHandler) Activate(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    if err := h.Service.ActivateCompetition(id); err != nil {
        Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
        return
    }
    Ok(w, map[string]string{"status": "active"})
}