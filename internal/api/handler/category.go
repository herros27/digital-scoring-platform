package handler

import (
	"errors"
	"net/http"
	"strings"

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
		if errors.Is(err, domain.ErrConflict) {
			Fail(w, http.StatusConflict, "CONFLICT", strings.TrimPrefix(err.Error(), domain.ErrConflict.Error()+": "), nil)
			return
		}
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

// DELETE /api/v1/competitions/{comp_id}/categories
// Reset seluruh kategori kejuaraan (untuk mengganti standar PB IPSI <-> PERSILAT 2026).
func (h *CategoryHandler) Reset(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	if compID == "" {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", "competition id is required", nil)
		return
	}
	n, err := h.Service.ResetCategories(compID)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			Fail(w, http.StatusConflict, "CONFLICT", "kategori masih dipakai peserta terdaftar; pindahkan atau hapus peserta tersebut terlebih dahulu", nil)
			return
		}
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, map[string]any{"competition_id": compID, "deleted": n})
}

// PUT /api/v1/competitions/{comp_id}/categories/ruleset
// Body: {"assignments":[{"age_group":"usia_dini_2","ruleset_id":"<uuid>"}, {"category_id":"<uuid>","ruleset_id":null}]}
// Menetapkan ruleset khusus per golongan umur / kategori; ruleset_id null = ikut ruleset kejuaraan.
func (h *CategoryHandler) AssignRulesets(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	var req struct {
		Assignments []domain.CategoryRulesetAssignment `json:"assignments" validate:"required,min=1"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	n, err := h.Service.AssignRulesets(compID, req.Assignments)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrConflict):
			Fail(w, http.StatusConflict, "CONFLICT", "ruleset kategori tidak dapat diganti karena sudah ada pertandingan yang berjalan atau selesai", nil)
		case errors.Is(err, domain.ErrInvalidInput):
			Fail(w, http.StatusBadRequest, "INVALID_INPUT", strings.TrimPrefix(err.Error(), domain.ErrInvalidInput.Error()+": "), nil)
		default:
			Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		}
		return
	}
	Ok(w, map[string]any{"competition_id": compID, "updated": n})
}

// PUT /api/v1/competitions/{comp_id}/categories/{id}/bracket-lock
// Body: {"locked": true|false}. Mengunci draw bagan kategori: setelah dikunci, brackets/generate untuk kategori itu
// ditolak (409 BRACKET_LOCKED) sampai dibuka lagi. Hanya admin/dewan.
func (h *CategoryHandler) SetBracketLock(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Locked *bool `json:"locked" validate:"required"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	cat, err := h.Service.SetBracketLock(r.PathValue("comp_id"), r.PathValue("id"), *req.Locked)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", "kategori tidak ditemukan pada kejuaraan ini", nil)
			return
		}
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, cat)
}

// PUT /api/v1/competitions/{comp_id}/categories/{id}
// Mengubah nama, gender, kelas/batas bobot, golongan umur, dan mode bagan satu kategori. Ditolak 409 bila standar
// berbeda atau kategori sudah punya peserta (saat itu hanya nama yang boleh berubah).
func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name           string   `json:"name" validate:"required,min=2,max=100"`
		Gender         string   `json:"gender" validate:"required,oneof=male female putra putri"`
		WeightClass    string   `json:"weight_class" validate:"required"`
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
	cat, err := h.Service.UpdateCategory(r.PathValue("comp_id"), r.PathValue("id"), &domain.Category{
		Name: req.Name, Gender: req.Gender, WeightClass: req.WeightClass, AgeGroup: req.AgeGroup,
		PresetStandard: req.PresetStandard, MinWeight: req.MinWeight, MaxWeight: req.MaxWeight, BracketMode: req.BracketMode,
	})
	if err != nil {
		failCategory(w, err)
		return
	}
	Ok(w, cat)
}

// PUT /api/v1/competitions/{comp_id}/categories/{id}/home-arena
// Mengubah gelanggang asal kategori. Partai kategori yang belum dimuat ikut pindah; 409 bila sudah ada yang berjalan.
func (h *CategoryHandler) SetHomeArena(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ArenaID string `json:"arena_id" validate:"required"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	cat, err := h.Service.SetHomeArena(r.PathValue("comp_id"), r.PathValue("id"), req.ArenaID)
	if err != nil {
		failCategory(w, err)
		return
	}
	Ok(w, cat)
}

// DELETE /api/v1/competitions/{comp_id}/categories/{id}
// Menghapus satu kategori. 409 bila masih ada peserta atau partainya sudah berjalan/selesai.
func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.Service.DeleteCategory(r.PathValue("comp_id"), r.PathValue("id")); err != nil {
		failCategory(w, err)
		return
	}
	Ok(w, map[string]string{"status": "deleted"})
}

func failCategory(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		Fail(w, http.StatusNotFound, "NOT_FOUND", "kategori tidak ditemukan pada kejuaraan ini", nil)
	case errors.Is(err, domain.ErrConflict):
		Fail(w, http.StatusConflict, "CONFLICT", strings.TrimPrefix(err.Error(), domain.ErrConflict.Error()+": "), nil)
	case errors.Is(err, domain.ErrInvalidInput):
		Fail(w, http.StatusBadRequest, "INVALID_INPUT", strings.TrimPrefix(err.Error(), domain.ErrInvalidInput.Error()+": "), nil)
	default:
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}
