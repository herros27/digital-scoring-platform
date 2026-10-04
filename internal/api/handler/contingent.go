package handler

import (
	"net/http"

	"github.com/herros27/digital-scoring/internal/service"
)

type ContingentHandler struct {
	Service *service.ContingentService
}

// POST /api/v1/auth/contingent/register
func (h *ContingentHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		PicName  string `json:"pic_name"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	contingent, err := h.Service.Register(req.Name, req.PicName, req.Email, req.Phone, req.Password)
	if err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	w.WriteHeader(http.StatusCreated)
	Ok(w, contingent)
}

// POST /api/v1/auth/contingent/login
func (h *ContingentHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	token, contingent, err := h.Service.Login(req.Email, req.Password)
	if err != nil {
		Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return
	}

	Ok(w, map[string]any{
		"token":      token,
		"contingent": contingent,
	})
}

// GET /api/v1/contingents
func (h *ContingentHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.Service.List()
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, list)
}

// GET /api/v1/contingents/{id}
func (h *ContingentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := h.Service.GetByID(id)
	if err != nil {
		Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		return
	}
	Ok(w, c)
}
