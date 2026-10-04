package handler

import (
	"errors"
	"net/http"

	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/cache"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
)

type AthleteHandler struct {
	Service *service.AthleteService
	Cache   *cache.Cache
}

func (h *AthleteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var a domain.Athlete
	if err := Decode(r, &a); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	if err := h.Service.CreateAthlete(&a); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	w.WriteHeader(http.StatusCreated)
	Ok(w, a)
}

// GET /api/v1/athletes?page=1&limit=20&gender=male
func (h *AthleteHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit, _ := ParsePagination(r)
	gender := r.URL.Query().Get("gender")

	list, total, err := h.Service.GetAthletesPaged(gender, page, limit)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	OkPaged(w, list, page, limit, total)
}

func (h *AthleteHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	athlete, err := h.Service.GetAthleteByID(id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	Ok(w, athlete)
}

// RegisterToCompetition registers an existing athlete or creates and links them to competition
func (h *AthleteHandler) RegisterToCompetition(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	if compID == "" {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", "competition id is required", nil)
		return
	}

	type RegisterRequest struct {
		AthleteID   string `json:"athlete_id"`
		Name        string `json:"name"`
		Gender      string `json:"gender"`
		Team        string `json:"team"`
		WeightClass string `json:"weight_class" validate:"required"`
	}

	var req RegisterRequest
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	athleteID := req.AthleteID
	gender := req.Gender

	// Determine status and contingent_id based on caller claims
	status := "registered"
	contingentID := ""
	if claims := auth.ClaimsFromContext(r.Context()); claims != nil {
		if claims.Role == "contingent" {
			contingentID = claims.JudgePinID
			status = "pending"
		}
	}

	// If no athlete_id provided, create new athlete record first
	if athleteID == "" {
		newAthlete := domain.Athlete{
			ContingentID: contingentID,
			Name:         req.Name,
			Gender:       req.Gender,
			Team:         req.Team,
		}
		if err := h.Service.CreateAthlete(&newAthlete); err != nil {
			Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
			return
		}
		athleteID = newAthlete.ID
		gender = newAthlete.Gender
	}

	if err := h.Service.RegisterParticipant(compID, athleteID, contingentID, req.WeightClass, gender, status); err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	if h.Cache != nil {
		h.Cache.InvalidateCompetition(compID)
	}

	w.WriteHeader(http.StatusCreated)
	Ok(w, map[string]string{
		"competition_id": compID,
		"athlete_id":     athleteID,
		"contingent_id":  contingentID,
		"weight_class":   req.WeightClass,
		"status":         status,
	})
}

// PUT /api/v1/competitions/{comp_id}/participants/{id}/verify
func (h *AthleteHandler) VerifyParticipant(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	participantID := r.PathValue("id")
	if participantID == "" {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", "participant id is required", nil)
		return
	}

	var req struct {
		Status string `json:"status" validate:"required,oneof=verified rejected"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	if err := h.Service.VerifyParticipant(participantID, req.Status); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	if h.Cache != nil {
		h.Cache.InvalidateCompetition(compID)
	}

	Ok(w, map[string]string{
		"participant_id": participantID,
		"status":         req.Status,
	})
}

// GET /api/v1/competitions/{comp_id}/participants?status=pending
func (h *AthleteHandler) ListParticipants(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	if compID == "" {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", "competition id is required", nil)
		return
	}
	status := r.URL.Query().Get("status")
	list, err := h.Service.ListParticipants(compID, status)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, list)
}

func (h *AthleteHandler) ListByCompetition(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	if compID == "" {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", "competition id is required", nil)
		return
	}

	list, err := h.Service.ListAthletesByCompetition(compID)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, list)
}
