package handler

import (
	"net/http"

	"github.com/herros27/digital-scoring/internal/cache"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

type BracketHandler struct {
	Service         *service.BracketService
	AthleteService  *service.AthleteService
	CategoryService *service.CategoryService
	Cache           *cache.Cache
}

// POST /api/v1/competitions/{comp_id}/brackets/generate
// Body: { "weight_class": "A", "gender": "male", "participant_ids": ["uuid-1", "uuid-2", ...] }
func (h *BracketHandler) Generate(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")

	var req struct {
		CategoryID     string   `json:"category_id"`
		WeightClass    string   `json:"weight_class"`
		Gender         string   `json:"gender" validate:"omitempty,oneof=male female putra putri"`
		ParticipantIDs []string `json:"participant_ids"`
	}

	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	// Bagan terisolasi per kategori: weight_class/gender/mode diambil dari kategori itu sendiri.
	bracketMode := ""
	if req.CategoryID != "" {
		if h.CategoryService == nil {
			Fail(w, http.StatusBadRequest, "BAD_REQUEST", "category brackets are not enabled", nil)
			return
		}
		cat, err := h.CategoryService.GetCategoryByID(req.CategoryID)
		if err != nil || cat.CompetitionID != compID {
			Fail(w, http.StatusNotFound, "NOT_FOUND", "category not found in this competition", nil)
			return
		}
		// Draw yang dikunci panitia tidak boleh diacak ulang (ditegakkan di server, bukan hanya di tombol UI), dan
		// bagan yang sudah mulai dimainkan tidak boleh dibentuk ulang sama sekali (akan memutus partai dari slotnya).
		if cat.BracketLocked {
			Fail(w, http.StatusConflict, "BRACKET_LOCKED", "draw bagan kategori ini terkunci; buka kunci draw terlebih dahulu", nil)
			return
		}
		if started, err := h.Service.CategoryBracketStarted(compID, cat.ID); err != nil {
			Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
			return
		} else if started {
			Fail(w, http.StatusConflict, "BRACKET_IN_PROGRESS", "bagan kategori ini sudah mulai dimainkan dan tidak bisa dibentuk ulang", nil)
			return
		}
		req.WeightClass = cat.WeightClass
		req.Gender = cat.Gender
		bracketMode = cat.BracketMode
	}

	// Auto-populate from verified participants if participant_ids omitted
	if len(req.ParticipantIDs) == 0 && h.AthleteService != nil {
		parts, err := h.AthleteService.ListParticipants(compID, "verified")
		if err == nil {
			for _, p := range parts {
				if req.CategoryID != "" {
					if p.CategoryID == req.CategoryID {
						req.ParticipantIDs = append(req.ParticipantIDs, p.ID)
					}
				} else if p.WeightClass == req.WeightClass && p.Gender == req.Gender && p.CategoryID == "" {
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
				if st != "pending" && st != "rejected" && st != "overweight" && st != "underweight" {
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

	var slots []store.BracketSlot
	var err error
	if req.CategoryID != "" {
		slots, err = h.Service.GenerateCategoryBracket(compID, req.CategoryID, req.WeightClass, req.Gender, bracketMode, req.ParticipantIDs)
	} else {
		slots, err = h.Service.GenerateBracket(compID, req.WeightClass, req.Gender, req.ParticipantIDs)
	}
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

	if categoryID := r.URL.Query().Get("category_id"); categoryID != "" {
		tree, err := h.Service.GetBracketTreeByCategory(compID, categoryID)
		if err != nil {
			Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
			return
		}
		Ok(w, tree)
		return
	}

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
