package handler

import (
	"net/http"

	"github.com/herros27/digital-scoring/internal/cache"
	"github.com/herros27/digital-scoring/internal/service"
)

type BracketHandler struct {
	Service        *service.BracketService
	AthleteService *service.AthleteService
	Cache          *cache.Cache
}

// POST /api/v1/competitions/{comp_id}/brackets/generate
// Body: { "weight_class": "A", "gender": "male", "participant_ids": ["uuid-1", "uuid-2", ...] }
func (h *BracketHandler) Generate(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")

	var req struct {
		WeightClass    string   `json:"weight_class" validate:"required"`
		Gender         string   `json:"gender" validate:"required,oneof=male female putra putri"`
		ParticipantIDs []string `json:"participant_ids"`
	}

	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	// Auto-populate from verified participants if participant_ids omitted
	if len(req.ParticipantIDs) == 0 && h.AthleteService != nil {
		parts, err := h.AthleteService.ListParticipants(compID, "verified")
		if err == nil {
			for _, p := range parts {
				if p.WeightClass == req.WeightClass && p.Gender == req.Gender {
					req.ParticipantIDs = append(req.ParticipantIDs, p.ID)
				}
			}
		}
	} else if len(req.ParticipantIDs) > 0 && h.AthleteService != nil {
		// Filter out pending or rejected participants
		parts, err := h.AthleteService.ListParticipants(compID, "")
		if err == nil && len(parts) > 0 {
			statusMap := make(map[string]string)
			for _, p := range parts {
				statusMap[p.ID] = p.RegistrationStatus
			}
			var verifiedIDs []string
			for _, pid := range req.ParticipantIDs {
				st := statusMap[pid]
				if st != "pending" && st != "rejected" {
					verifiedIDs = append(verifiedIDs, pid)
				}
			}
			req.ParticipantIDs = verifiedIDs
		}
	}

	if req.WeightClass == "" || req.Gender == "" || len(req.ParticipantIDs) < 2 {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", "weight_class, gender, and at least 2 verified participants required", nil)
		return
	}

	slots, err := h.Service.GenerateBracket(compID, req.WeightClass, req.Gender, req.ParticipantIDs)
	if err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	if h.Cache != nil {
		h.Cache.InvalidateCompetition(compID)
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
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", "weight_class and gender query parameters required", nil)
		return
	}

	tree, err := h.Service.GetBracketTree(compID, weightClass, gender)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	Ok(w, tree)
}
