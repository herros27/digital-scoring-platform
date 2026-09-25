package handler

import (
	"net/http"

	"github.com/herros27/digital-scoring/internal/service"
)

type BracketHandler struct {
	Service *service.BracketService
}

// POST /api/v1/competitions/{comp_id}/brackets/generate
// Body: { "weight_class": "A", "gender": "male", "participant_ids": ["uuid-1", "uuid-2", ...] }
func (h *BracketHandler) Generate(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")

	var req struct {
		WeightClass    string   `json:"weight_class"`
		Gender         string   `json:"gender"`
		ParticipantIDs []string `json:"participant_ids"`
	}

	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	if req.WeightClass == "" || req.Gender == "" || len(req.ParticipantIDs) < 2 {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", "weight_class, gender, and at least 2 participants required")
		return
	}

	slots, err := h.Service.GenerateBracket(compID, req.WeightClass, req.Gender, req.ParticipantIDs)
	if err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	Ok(w, slots)
}

// GET /api/v1/competitions/{comp_id}/brackets?weight_class=A&gender=male
func (h *BracketHandler) GetTree(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	weightClass := r.URL.Query().Get("weight_class")
	gender := r.URL.Query().Get("gender")

	if weightClass == "" || gender == "" {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", "weight_class and gender query parameters required")
		return
	}

	tree, err := h.Service.GetBracketTree(compID, weightClass, gender)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	Ok(w, tree)
}