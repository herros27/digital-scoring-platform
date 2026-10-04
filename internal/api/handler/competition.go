package handler

import (
	"errors"
	"net/http"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
)

type CompetitionHandler struct {
	Service *service.CompetitionService
}

// GET /api/v1/competitions?page=1&limit=20&status=active
func (h *CompetitionHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit, _ := ParsePagination(r)
	status := r.URL.Query().Get("status")

	list, total, err := h.Service.GetCompetitionsPaged(status, page, limit)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	OkPaged(w, list, page, limit, total)
}

// GET /api/v1/competitions/{id}
func (h *CompetitionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := h.Service.GetCompetitionByID(id)
	if err != nil {
		if errors.Is(err, domain.ErrCompetitionNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	Ok(w, c)
}

// POST /api/v1/competitions
func (h *CompetitionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var c domain.Competition
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
		if errors.Is(err, domain.ErrCompetitionNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrAlreadyActive) {
			Fail(w, http.StatusBadRequest, "ALREADY_ACTIVE", err.Error(), nil)
			return
		}
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, map[string]string{"status": "active"})
}

// PUT /api/v1/competitions/{id}
func (h *CompetitionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req domain.Competition
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	updated, err := h.Service.UpdateCompetition(id, &req)
	if err != nil {
		if errors.Is(err, domain.ErrCompetitionNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	Ok(w, updated)
}

// POST /api/v1/competitions/{id}/pause
func (h *CompetitionHandler) Pause(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.Service.PauseCompetition(id); err != nil {
		if errors.Is(err, domain.ErrCompetitionNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, map[string]string{"status": "paused"})
}

// POST /api/v1/competitions/{id}/finish
func (h *CompetitionHandler) Finish(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.Service.FinishCompetition(id); err != nil {
		if errors.Is(err, domain.ErrCompetitionNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, map[string]string{"status": "completed"})
}

// POST /api/v1/competitions/{id}/cancel
func (h *CompetitionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.Service.CancelCompetition(id); err != nil {
		if errors.Is(err, domain.ErrCompetitionNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, map[string]string{"status": "cancelled"})
}
