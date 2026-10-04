package handler

import (
	"errors"
	"net/http"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
)

type ReportHandler struct {
	Service *service.ReportService
}

// GET /api/v1/competitions/{comp_id}/report/summary
func (h *ReportHandler) Summary(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	summary, err := h.Service.GetSummary(compID)
	if err != nil {
		if errors.Is(err, domain.ErrCompetitionNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return
	}
	Ok(w, summary)
}

// GET /api/v1/competitions/{comp_id}/report/medals
func (h *ReportHandler) Medals(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	standings, err := h.Service.GetMedalStandings(compID)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, standings)
}

// GET /api/v1/competitions/{comp_id}/report/export
func (h *ReportHandler) Export(w http.ResponseWriter, r *http.Request) {
	compID := r.PathValue("comp_id")
	export, err := h.Service.ExportCompetition(compID)
	if err != nil {
		if errors.Is(err, domain.ErrCompetitionNotFound) {
			Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	Ok(w, export)
}
