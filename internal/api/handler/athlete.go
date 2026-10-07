package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/cache"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
)

type AthleteHandler struct {
	Service *service.AthleteService
	Cache   *cache.Cache
	// Competition opsional: tanggal mulai kejuaraan dipakai sebagai tanggal acuan hitung umur.
	Competition *service.CompetitionService
}

// ageReferenceDate tanggal acuan golongan umur: hari pertama kejuaraan, fallback hari ini.
func (h *AthleteHandler) ageReferenceDate(compID string) time.Time {
	if h.Competition != nil {
		if comp, err := h.Competition.GetCompetitionByID(compID); err == nil && comp != nil && comp.StartDate != "" {
			if len(comp.StartDate) >= 10 {
				if t, err := time.Parse("2006-01-02", comp.StartDate[:10]); err == nil {
					return t
				}
			}
		}
	}
	return time.Now()
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
		BirthDate   string `json:"birth_date"`
		NIK         string `json:"nik" validate:"omitempty,max=50"`
		CategoryID  string `json:"category_id"`
		WeightClass string `json:"weight_class" validate:"required_without=CategoryID"`
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
			BirthDate:    req.BirthDate,
			NIK:          req.NIK,
		}
		if req.CategoryID != "" && req.BirthDate == "" {
			Fail(w, http.StatusBadRequest, "BAD_REQUEST", "birth_date wajib diisi untuk pendaftaran ke kategori golongan umur", nil)
			return
		}
		if req.CategoryID != "" {
			// Validasi umur/gender sebelum atlet dibuat agar tidak ada atlet yatim bila ditolak.
			if _, err := h.Service.CheckCategoryEligibility(compID, req.CategoryID, &newAthlete, h.ageReferenceDate(compID)); err != nil {
				Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
				return
			}
		}
		if err := h.Service.CreateAthlete(&newAthlete); err != nil {
			Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
			return
		}
		athleteID = newAthlete.ID
		gender = newAthlete.Gender
	}

	weightClass := req.WeightClass
	if req.CategoryID != "" {
		cat, err := h.Service.RegisterParticipantToCategory(compID, athleteID, contingentID, req.CategoryID, status, h.ageReferenceDate(compID))
		if err != nil {
			Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
			return
		}
		weightClass = cat.WeightClass
	} else if err := h.Service.RegisterParticipant(compID, athleteID, contingentID, req.WeightClass, gender, status); err != nil {
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
		"category_id":    req.CategoryID,
		"weight_class":   weightClass,
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
		Status       string          `json:"status" validate:"required,oneof=verified rejected"`
		ActualWeight *float64        `json:"actual_weight"`
		DocsVerified map[string]bool `json:"docs_verified"`
	}
	if err := Decode(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	res, err := h.Service.VerifyParticipantDetailed(participantID, service.VerifyInput{
		Status:       req.Status,
		ActualWeight: req.ActualWeight,
		Docs:         req.DocsVerified,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}

	if h.Cache != nil {
		h.Cache.InvalidateCompetition(compID)
	}

	Ok(w, map[string]any{
		"participant_id": participantID,
		"status":         res.Status,
		"reason":         res.Reason,
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
