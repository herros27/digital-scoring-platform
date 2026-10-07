package handler

import (
	"errors"
	"net/http"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
)

type CategoryHandler struct {
	Service *service.CategoryService
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	if compID == "" {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", "competition id is required", nil)
		return
	}

	var req struct {
		Name        string  `json:"name" validate:"required,min=2,max=100"`
		Gender      string  `json:"gender" validate:"required,oneof=male female putra putri"`
		WeightClass string  `json:"weight_class" validate:"required"`
		RulesetID   *string `json:"ruleset_id,omitempty"`

		AgeGroup       string   `json:"age_group"`
		PresetStandard string   `json:"preset_standard"`
		MinWeight      *float64 `json:"min_weight"`
		MaxWeight      *float64 `json:"max_weight"`
		BracketMode    string   `json:"bracket_mode"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	c := domain.Category{
		CompetitionID:  compID,
		Name:           req.Name,
		Gender:         req.Gender,
		WeightClass:    req.WeightClass,
		RulesetID:      req.RulesetID,
		AgeGroup:       req.AgeGroup,
		PresetStandard: req.PresetStandard,
		MinWeight:      req.MinWeight,
		MaxWeight:      req.MaxWeight,
		BracketMode:    req.BracketMode,
	}
	if err := h.Service.CreateCategory(&c); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	w.WriteHeader(http.StatusCreated)
	Ok(w, c)
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	if compID == "" {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", "competition id is required", nil)
		return
	}

	list, err := h.Service.ListCategories(compID)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, list)
}

func (h *CategoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cat, err := h.Service.GetCategoryByID(id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	Ok(w, cat)
}
