package handler

import (
	"errors"
	"net/http"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
)

type RulesetHandler struct {
	Service *service.RulesetService
}

// GET /api/v1/rulesets
func (h *RulesetHandler) List(w http.ResponseWriter, r *http.Request) {
	rulesets, err := h.Service.ListRulesets()
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, rulesets)
}

// GET /api/v1/rulesets/{id}
func (h *RulesetHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rs, err := h.Service.GetRulesetByID(id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", "ruleset not found", nil)
			return
		}
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, rs)
}

// POST /api/v1/rulesets
func (h *RulesetHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateRulesetInput
	if err := Decode(r, &input); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	rs, err := h.Service.CreateRuleset(input)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			Fail(w, http.StatusConflict, "CONFLICT", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			Fail(w, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
			return
		}
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	w.WriteHeader(http.StatusCreated)
	Ok(w, rs)
}

// PUT /api/v1/rulesets/{id}
func (h *RulesetHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var input service.CreateRulesetInput
	if err := Decode(r, &input); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	rs, err := h.Service.UpdateRuleset(id, input)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			Fail(w, http.StatusConflict, "CONFLICT", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", "ruleset not found", nil)
			return
		}
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	Ok(w, rs)
}

// DELETE /api/v1/rulesets/{id}
func (h *RulesetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.Service.DeleteRuleset(id); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			Fail(w, http.StatusConflict, "CONFLICT", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", "ruleset not found", nil)
			return
		}
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, map[string]string{"message": "ruleset deleted successfully"})
}
