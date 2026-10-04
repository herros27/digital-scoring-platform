package handler

import (
	"net/http"

	"github.com/herros27/digital-scoring/internal/service"
)

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
		Password string `json:"password" validate:"required,min=6"`
		Role     string `json:"role" validate:"required,oneof=admin dewan wasit juri operator"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	official, err := h.Service.CreateOfficial(req.Name, req.Email, req.Password, req.Role)
	if err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
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

	official, err := h.Service.UpdateOfficial(id, req.Name, req.Role)
	if err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	Ok(w, official)
}

// DELETE /api/v1/officials/{id}
func (h *OfficialHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.Service.DeleteOfficial(id); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	Ok(w, map[string]string{"status": "deleted"})
}
