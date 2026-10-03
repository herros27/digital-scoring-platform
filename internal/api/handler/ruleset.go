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
