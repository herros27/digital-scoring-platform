package handler

import (
	"net/http"
	"sort"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/store"
)

// publicCompetitionStatus: bagan hanya dibuka untuk umum selama kejuaraan berstatus aktif, yaitu setelah
// panitia mengaktifkannya. Kejuaraan draft/paused/finished/cancelled disamarkan sebagai tidak ditemukan.
const publicCompetitionStatus = "active"

// Antarmuka sempit agar handler mudah diuji; *service.XxxService memenuhinya.
type publicCompetitionReader interface {
	GetCompetitionsPaged(status string, page, limit int) ([]domain.Competition, int, error)
	GetCompetitionByID(id string) (*domain.Competition, error)
}
type publicCategoryReader interface {
	ListCategories(competitionID string) ([]domain.Category, error)
	GetCategoryByID(id string) (*domain.Category, error)
}
type publicBracketReader interface {
	GetBracketTreeByCategory(competitionID, categoryID string) ([]store.BracketSlot, error)
	ListCategoryBracketSummaries(competitionID string) ([]store.CategoryBracketSummary, error)
}
type publicParticipantReader interface {
	ListParticipants(competitionID, status string) ([]domain.Participant, error)
}

// PublicBracketHandler melayani bagan untuk penonton tanpa login. Respons dibentuk dari DTO khusus
// (bukan domain.BracketSlot/Participant) sehingga hanya field di bawah ini yang bisa keluar: nama dan
// kontingen atlet, tanpa NIK, tanggal lahir, berat badan, berkas, maupun ID peserta/atlet.
type PublicBracketHandler struct {
	Competitions publicCompetitionReader
	Categories   publicCategoryReader
	Brackets     publicBracketReader
	Participants publicParticipantReader
}

type publicCompetitionDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Location  string `json:"location"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type publicBracketListItem struct {
	CategoryID   string `json:"category_id"`
	Name         string `json:"name"`
	Gender       string `json:"gender"`
	AgeGroup     string `json:"age_group"`
	WeightClass  string `json:"weight_class"`
	Participants int    `json:"participants"`
	Rounds       int    `json:"rounds"`
	Completed    bool   `json:"completed"`
}

type publicAthleteDTO struct {
	Name string `json:"name"`
	Team string `json:"team"`
}

type publicSlotDTO struct {
	ID          string            `json:"id"`
	Round       int               `json:"round"`
	SlotIndex   int               `json:"slot_index"`
	IsBye       bool              `json:"is_bye"`
	NextSlotID  *string           `json:"next_slot_id"`
	IsWinner    bool              `json:"is_winner"`
	Participant *publicAthleteDTO `json:"participant"`
}

type publicBracketDetail struct {
	Competition publicCompetitionDTO  `json:"competition"`
	Category    publicBracketListItem `json:"category"`
	Slots       []publicSlotDTO       `json:"slots"`
}

// GET /api/v1/public/competitions — kejuaraan aktif.
func (h *PublicBracketHandler) ListCompetitions(w http.ResponseWriter, r *http.Request) {
	list, _, err := h.Competitions.GetCompetitionsPaged(publicCompetitionStatus, 1, 50)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", "gagal memuat kejuaraan", nil)
		return
	}
	out := make([]publicCompetitionDTO, 0, len(list))
	for _, c := range list {
		out = append(out, toPublicCompetition(c))
	}
	Ok(w, out)
}

// GET /api/v1/public/competitions/{comp_id}/brackets — daftar bagan (satu per kategori) yang sudah dibuat.
func (h *PublicBracketHandler) ListBrackets(w http.ResponseWriter, r *http.Request) {
	comp, ok := h.activeCompetition(w, r.PathValue("comp_id"))
	if !ok {
		return
	}
	items, err := h.bracketItems(comp.ID)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", "gagal memuat bagan", nil)
		return
	}
	Ok(w, items)
}

// GET /api/v1/public/competitions/{comp_id}/brackets/{category_id} — satu bagan beserta nama atlet.
func (h *PublicBracketHandler) GetBracket(w http.ResponseWriter, r *http.Request) {
	comp, ok := h.activeCompetition(w, r.PathValue("comp_id"))
	if !ok {
		return
	}
	categoryID := r.PathValue("category_id")

	items, err := h.bracketItems(comp.ID)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", "gagal memuat bagan", nil)
		return
	}
	var item *publicBracketListItem
	for i := range items {
		if items[i].CategoryID == categoryID {
			item = &items[i]
			break
		}
	}
	if item == nil { // kategori tidak ada, milik kejuaraan lain, atau belum punya bagan
		Fail(w, http.StatusNotFound, "NOT_FOUND", "bagan tidak ditemukan", nil)
		return
	}

	slots, err := h.Brackets.GetBracketTreeByCategory(comp.ID, categoryID)
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", "gagal memuat bagan", nil)
		return
	}
	parts, err := h.Participants.ListParticipants(comp.ID, "")
	if err != nil {
		Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", "gagal memuat bagan", nil)
		return
	}
	// Hanya nama & kontingen yang disalin; sisa data peserta tidak pernah meninggalkan fungsi ini.
	athletes := make(map[string]publicAthleteDTO, len(parts))
	for _, p := range parts {
		athletes[p.ID] = publicAthleteDTO{Name: p.Name, Team: p.Club}
	}

	out := make([]publicSlotDTO, 0, len(slots))
	for _, s := range slots {
		dto := publicSlotDTO{ID: s.ID, Round: s.Round, SlotIndex: s.SlotIndex, IsBye: s.IsBye, NextSlotID: s.NextSlotID}
		if s.ParticipantID != nil {
			if a, ok := athletes[*s.ParticipantID]; ok {
				dto.Participant = &a
			}
			dto.IsWinner = s.WinnerID != nil && *s.WinnerID == *s.ParticipantID
		}
		out = append(out, dto)
	}
	Ok(w, publicBracketDetail{Competition: toPublicCompetition(*comp), Category: *item, Slots: out})
}

// activeCompetition memuat kejuaraan dan menolak (404) bila tidak ada atau belum/tidak lagi aktif.
func (h *PublicBracketHandler) activeCompetition(w http.ResponseWriter, id string) (*domain.Competition, bool) {
	comp, err := h.Competitions.GetCompetitionByID(id)
	// Semua kegagalan (tidak ada, id salah, belum aktif) dijawab sama agar penonton tidak bisa
	// membedakan kejuaraan draft dari yang tidak ada.
	if err != nil || comp == nil || comp.Status != publicCompetitionStatus {
		Fail(w, http.StatusNotFound, "NOT_FOUND", "kejuaraan tidak ditemukan", nil)
		return nil, false
	}
	return comp, true
}

// bracketItems menggabungkan kategori dengan ringkasan bagannya; kategori tanpa bagan tidak ditampilkan.
func (h *PublicBracketHandler) bracketItems(competitionID string) ([]publicBracketListItem, error) {
	cats, err := h.Categories.ListCategories(competitionID)
	if err != nil {
		return nil, err
	}
	sums, err := h.Brackets.ListCategoryBracketSummaries(competitionID)
	if err != nil {
		return nil, err
	}
	byCat := make(map[string]store.CategoryBracketSummary, len(sums))
	for _, s := range sums {
		byCat[s.CategoryID] = s
	}
	items := make([]publicBracketListItem, 0, len(sums))
	for _, c := range cats {
		s, ok := byCat[c.ID]
		if !ok {
			continue
		}
		items = append(items, publicBracketListItem{
			CategoryID: c.ID, Name: c.Name, Gender: c.Gender, AgeGroup: c.AgeGroup, WeightClass: c.WeightClass,
			Participants: s.Participants, Rounds: s.Rounds, Completed: s.Completed,
		})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func toPublicCompetition(c domain.Competition) publicCompetitionDTO {
	return publicCompetitionDTO{ID: c.ID, Name: c.Name, Location: c.Location, StartDate: dateOnly(c.StartDate), EndDate: dateOnly(c.EndDate)}
}

// dateOnly memotong timestamp ISO ("2026-06-01T00:00:00Z") menjadi tanggal ("2026-06-01").
func dateOnly(s string) string {
	if len(s) > 10 {
		return s[:10]
	}
	return s
}
