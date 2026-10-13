package handler

import (
	"fmt"
	"net/http"

	"github.com/herros27/digital-scoring/internal/cache"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

type BracketHandler struct {
	Service         *service.BracketService
	AthleteService  *service.AthleteService
	CategoryService *service.CategoryService
	Cache           *cache.Cache
}

// redrawBlocked menjelaskan mengapa bagan kategori tidak boleh dibentuk ulang atau direset ("" = boleh).
// Draw yang dikunci panitia tidak boleh diubah (ditegakkan di server, bukan hanya di tombol UI), dan bagan yang
// sudah mulai dimainkan tidak boleh dibentuk ulang sama sekali (akan memutus partai dari slotnya).
func (h *BracketHandler) redrawBlocked(compID string, cat *domain.Category) (status int, code, msg string) {
	if cat.BracketLocked {
		return http.StatusConflict, "BRACKET_LOCKED", "draw bagan kategori ini terkunci; buka kunci draw terlebih dahulu"
	}
	started, err := h.Service.CategoryBracketStarted(compID, cat.ID)
	if err != nil {
		return http.StatusInternalServerError, "INTERNAL_ERROR", err.Error()
	}
	if started {
		return http.StatusConflict, "BRACKET_IN_PROGRESS", "bagan kategori ini sudah mulai dimainkan dan tidak bisa dibentuk ulang atau direset"
	}
	return 0, "", ""
}

// DELETE /api/v1/competitions/{comp_id}/categories/{id}/bracket
// Menghapus bagan satu kategori beserta partai idle-nya (peserta tetap terdaftar) agar bisa dibuat ulang dari nol,
// mis. setelah ada peserta mundur. Ditolak bila draw terkunci atau sudah ada partai yang dimulai.
func (h *BracketHandler) Reset(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	cat, err := h.CategoryService.GetCategoryByID(r.PathValue("id"))
	if err != nil || cat.CompetitionID != compID {
		Fail(w, http.StatusNotFound, "NOT_FOUND", "category not found in this competition", nil)
		return
	}
	if status, code, msg := h.redrawBlocked(compID, cat); code != "" {
		Fail(w, status, code, msg, nil)
		return
	}
	if err := h.Service.ResetCategoryBracket(compID, cat.ID); err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	if h.Cache != nil {
		h.Cache.InvalidateCompetition(compID)
	}
	Ok(w, map[string]any{"competition_id": compID, "category_id": cat.ID, "reset": true})
}

// AutoJoin dipanggil setelah peserta lolos verifikasi: bila kategorinya sudah punya bagan dan draw belum dikunci/dimainkan,
// bagan dibentuk ulang agar peserta baru ikut. Mengembalikan catatan untuk panitia ("" = tidak ada yang perlu dilaporkan).
// Kategori tanpa bagan dibiarkan: generate pertama tetap manual.
func (h *BracketHandler) AutoJoin(compID, categoryID string) string {
	if categoryID == "" || h.CategoryService == nil || h.AthleteService == nil {
		return ""
	}
	cat, err := h.CategoryService.GetCategoryByID(categoryID)
	if err != nil || cat.CompetitionID != compID {
		return ""
	}
	if tree, err := h.Service.GetBracketTreeByCategory(compID, categoryID); err != nil || len(tree) == 0 {
		return ""
	}
	if _, code, _ := h.redrawBlocked(compID, cat); code != "" {
		return "Bagan kategori ini terkunci atau sudah berjalan, peserta belum masuk bagan. Buka kunci atau reset bagan terlebih dahulu."
	}
	parts, err := h.AthleteService.ListParticipants(compID, "verified")
	if err != nil {
		return ""
	}
	var ids []string
	for _, p := range parts {
		if p.CategoryID == categoryID {
			ids = append(ids, p.ID)
		}
	}
	if _, err := h.Service.GenerateCategoryBracket(compID, categoryID, cat.WeightClass, cat.Gender, cat.BracketMode, ids); err != nil {
		return "Peserta lolos, tetapi bagan belum bisa dibuat ulang otomatis: " + err.Error()
	}
	if h.Cache != nil {
		h.Cache.InvalidateCompetition(compID)
	}
	return "Bagan kategori ini dibentuk ulang otomatis dan peserta baru sudah masuk."
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
		if status, code, msg := h.redrawBlocked(compID, cat); code != "" {
			Fail(w, status, code, msg, nil)
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
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", fmt.Sprintf("bagan butuh minimal 2 peserta terverifikasi pada kategori ini (saat ini %d); verifikasi peserta atau pilih kategori lain", len(req.ParticipantIDs)), nil)
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
