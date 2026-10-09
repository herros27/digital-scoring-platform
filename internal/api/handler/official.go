package handler

import (
	"errors"
	"net/http"

	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
)

// officialActor petugas yang melakukan permintaan, dari klaim token (JudgePinID menyimpan ID akun resmi).
func officialActor(r *http.Request) (service.Actor, bool) {
	c := auth.ClaimsFromContext(r.Context())
	if c == nil || c.JudgePinID == "" {
		return service.Actor{}, false
	}
	return service.Actor{ID: c.JudgePinID, Role: c.Role}, true
}

// failOfficial memetakan galat service ke status HTTP.
func failOfficial(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	case errors.Is(err, domain.ErrNotFound):
		Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, domain.ErrConflict):
		Fail(w, http.StatusConflict, "CONFLICT", err.Error(), nil)
	case errors.Is(err, domain.ErrUnauthorized):
		Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
	default:
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
	}
}

type OfficialHandler struct {
	Service *service.OfficialService
}

// POST /api/v1/auth/login
func (h *OfficialHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=1"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	token, official, err := h.Service.Login(req.Email, req.Password)
	if err != nil {
		Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return
	}

	Ok(w, map[string]any{
		"token": token,
		"user":  official,
	})
}

// GET /api/v1/officials
func (h *OfficialHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.Service.ListOfficials()
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, list)
}

// POST /api/v1/officials
func (h *OfficialHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name" validate:"required,min=2,max=100"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8"`
		Role     string `json:"role" validate:"required,oneof=admin dewan wasit juri operator"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	actor, ok := officialActor(r)
	if !ok {
		Fail(w, http.StatusForbidden, "FORBIDDEN", "akun resmi tidak dikenali", nil)
		return
	}
	official, err := h.Service.CreateOfficialBy(actor, req.Name, req.Email, req.Password, req.Role)
	if err != nil {
		failOfficial(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	Ok(w, official)
}

// GET /api/v1/officials/{id}
func (h *OfficialHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	official, err := h.Service.GetOfficialByID(id)
	if err != nil {
		Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		return
	}
	Ok(w, official)
}

// PUT /api/v1/officials/{id}
func (h *OfficialHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name string `json:"name" validate:"required,min=2,max=100"`
		Role string `json:"role" validate:"required,oneof=admin dewan wasit juri operator"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	actor, ok := officialActor(r)
	if !ok {
		Fail(w, http.StatusForbidden, "FORBIDDEN", "akun resmi tidak dikenali", nil)
		return
	}
	official, err := h.Service.UpdateOfficialBy(actor, id, req.Name, req.Role)
	if err != nil {
		failOfficial(w, err)
		return
	}
	Ok(w, official)
}

// DELETE /api/v1/officials/{id}
func (h *OfficialHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	actor, ok := officialActor(r)
	if !ok {
		Fail(w, http.StatusForbidden, "FORBIDDEN", "akun resmi tidak dikenali", nil)
		return
	}
	if err := h.Service.DeleteOfficialBy(actor, id); err != nil {
		failOfficial(w, err)
		return
	}
	Ok(w, map[string]string{"status": "deleted"})
}

// PUT /api/v1/officials/{id}/password — admin/dewan mereset password akun lain (tanpa password lama).
func (h *OfficialHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password" validate:"required,min=8,max=72"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	actor, ok := officialActor(r)
	if !ok {
		Fail(w, http.StatusForbidden, "FORBIDDEN", "akun resmi tidak dikenali", nil)
		return
	}
	if err := h.Service.ResetPassword(actor, r.PathValue("id"), req.Password); err != nil {
		failOfficial(w, err)
		return
	}
	Ok(w, map[string]string{"status": "password_reset"})
}

// POST /api/v1/auth/change-password — petugas mengganti password sendiri (password lama wajib benar).
func (h *OfficialHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password" validate:"required,min=8,max=72"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	actor, ok := officialActor(r)
	if !ok {
		Fail(w, http.StatusForbidden, "FORBIDDEN", "akun resmi tidak dikenali", nil)
		return
	}
	if err := h.Service.ChangeOwnPassword(actor.ID, req.CurrentPassword, req.NewPassword); err != nil {
		failOfficial(w, err)
		return
	}
	Ok(w, map[string]string{"status": "password_changed"})
}
