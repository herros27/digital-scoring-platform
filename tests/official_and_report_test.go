package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/api/handler"
	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

type mockOfficialRepo struct {
	officials map[string]*domain.Official
	passwords map[string]string
}

func newMockOfficialRepo() *mockOfficialRepo {
	return &mockOfficialRepo{
		officials: make(map[string]*domain.Official),
		passwords: make(map[string]string),
	}
}

func (m *mockOfficialRepo) Create(o *domain.Official, passwordHash string) error {
	if o.ID == "" {
		o.ID = "off-1"
	}
	m.officials[o.ID] = o
	m.passwords[o.ID] = passwordHash
	return nil
}

func (m *mockOfficialRepo) GetByID(id string) (*domain.Official, error) {
	if o, ok := m.officials[id]; ok {
		return o, nil
	}
	return nil, nil
}

func (m *mockOfficialRepo) GetByEmail(email string) (*domain.Official, string, error) {
	for _, o := range m.officials {
		if o.Email == email {
			return o, m.passwords[o.ID], nil
		}
	}
	return nil, "", nil
}

func (m *mockOfficialRepo) List() ([]domain.Official, error) {
	var list []domain.Official
	for _, o := range m.officials {
		list = append(list, *o)
	}
	return list, nil
}

func (m *mockOfficialRepo) Update(o *domain.Official) error {
	m.officials[o.ID] = o
	return nil
}

func (m *mockOfficialRepo) Delete(id string) error {
	delete(m.officials, id)
	delete(m.passwords, id)
	return nil
}

func TestOfficialService_CRUDAndLogin(t *testing.T) {
	repo := newMockOfficialRepo()
	svc := service.NewOfficialService(repo)

	// 1. Create Official
	created, err := svc.CreateOfficial("Dewan 1", "dewan@silat.id", "mock_secret_1", "dewan")
	if err != nil {
		t.Fatalf("CreateOfficial failed: %v", err)
	}
	if created.Email != "dewan@silat.id" || created.Role != "dewan" {
		t.Fatalf("unexpected official fields: %+v", created)
	}

	// 2. Duplicate email check
	_, errDup := svc.CreateOfficial("Duplicate", "dewan@silat.id", "pass", "admin")
	if errDup != domain.ErrConflict {
		t.Errorf("expected ErrConflict on duplicate email, got %v", errDup)
	}

	// 3. Login success
	token, off, err := svc.Login("dewan@silat.id", "mock_secret_1")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if token == "" || off.ID != created.ID {
		t.Errorf("invalid token or user returned: token=%s user=%+v", token, off)
	}

	// Verify token claims using auth package
	claims, err := auth.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken failed on generated JWT: %v", err)
	}
	if claims.Role != "dewan" || claims.JudgePinID != created.ID {
		t.Errorf("claims mismatch: role=%s pinID=%s", claims.Role, claims.JudgePinID)
	}

	// 4. Login wrong password
	_, _, errWrongPass := svc.Login("dewan@silat.id", "wrongpassword")
	if errWrongPass == nil {
		t.Errorf("expected error on wrong password, got nil")
	}

	// 5. Login non-existent email
	_, _, errNotFound := svc.Login("unknown@silat.id", "mock_secret_1")
	if errNotFound == nil {
		t.Errorf("expected error on unknown email, got nil")
	}

	// 6. GetByID & List
	fetched, err := svc.GetOfficialByID(created.ID)
	if err != nil || fetched.ID != created.ID {
		t.Errorf("GetOfficialByID failed: %v", err)
	}
	list, err := svc.ListOfficials()
	if err != nil || len(list) != 1 {
		t.Errorf("ListOfficials failed: len=%d err=%v", len(list), err)
	}

	// 7. Update Official
	updated, err := svc.UpdateOfficial(created.ID, "Dewan Utama", "admin")
	if err != nil {
		t.Fatalf("UpdateOfficial failed: %v", err)
	}
	if updated.Name != "Dewan Utama" || updated.Role != "admin" {
		t.Errorf("updated fields mismatch: %+v", updated)
	}

	// 8. Delete Official
	if err := svc.DeleteOfficial(created.ID); err != nil {
		t.Fatalf("DeleteOfficial failed: %v", err)
	}
	_, errDel := svc.GetOfficialByID(created.ID)
	if errDel != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound after deletion, got %v", errDel)
	}
}

type mockReportMatchRepo struct {
	matches []store.Match
}

func (m *mockReportMatchRepo) ListByCompetition(competitionID string) ([]store.Match, error) {
	var res []store.Match
	for _, match := range m.matches {
		if match.CompetitionID == competitionID {
			res = append(res, match)
		}
	}
	return res, nil
}

type mockAthleteRepo struct {
	athletes map[string]*domain.Athlete
}

func (m *mockAthleteRepo) Create(a *domain.Athlete) error {
	m.athletes[a.ID] = a
	return nil
}

func (m *mockAthleteRepo) GetByID(id string) (*domain.Athlete, error) {
	if a, ok := m.athletes[id]; ok {
		return a, nil
	}
	return nil, nil
}

func (m *mockAthleteRepo) List() ([]domain.Athlete, error) {
	var list []domain.Athlete
	for _, a := range m.athletes {
		list = append(list, *a)
	}
	return list, nil
}

func (m *mockAthleteRepo) RegisterToCompetition(competitionID, athleteID, weightClass, gender string) error {
	return nil
}

func (m *mockAthleteRepo) ListByCompetition(competitionID string) ([]domain.Athlete, error) {
	var list []domain.Athlete
	for _, a := range m.athletes {
		list = append(list, *a)
	}
	return list, nil
}

func TestReportService_SummaryAndMedalsAndExport(t *testing.T) {
	compMap := make(map[string]*domain.Competition)
	compRepo := &mockLifecycleCompRepo{comps: compMap}
	compID := "comp-rep-1"
	comp := &domain.Competition{
		ID:        compID,
		Name:      "Kejurnas Silat 2026",
		Location:  "Surabaya",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	_ = compRepo.Create(comp)

	arenaRepo := newMockArenaRepo()
	arena1 := &store.Arena{ID: "arena-rep-1", CompetitionID: compID, Name: "Gelanggang 1"}
	_ = arenaRepo.Create(arena1)

	categoryRepo := &mockCategoryRepo{categories: make(map[string]*domain.Category)}
	cat1 := &domain.Category{ID: "cat-rep-1", CompetitionID: compID, Name: "Tanding Dewasa A Putra", Gender: "male", WeightClass: "A"}
	_ = categoryRepo.Create(cat1)

	athleteRepo := &mockAthleteRepo{athletes: make(map[string]*store.Athlete)}
	ath1 := &store.Athlete{ID: "ath-1", Name: "Pesilat Merah", Team: "Kontingen Jatim"}
	ath2 := &store.Athlete{ID: "ath-2", Name: "Pesilat Biru", Team: "Kontingen Jabar"}
	_ = athleteRepo.Create(ath1)
	_ = athleteRepo.Create(ath2)

	matchRepo := &mockReportMatchRepo{
		matches: []store.Match{
			{
				ID:            "m-1",
				CompetitionID: compID,
				ArenaID:       "arena-rep-1",
				Status:        "finished",
				CurrentRound:  3,
				AthleteRedID:  "ath-1",
				AthleteBlueID: "ath-2",
				WinnerID:      "ath-1",
			},
			{
				ID:            "m-2",
				CompetitionID: compID,
				ArenaID:       "arena-rep-1",
				Status:        "running",
				CurrentRound:  1,
				AthleteRedID:  "ath-1",
				AthleteBlueID: "ath-2",
			},
		},
	}

	reportSvc := service.NewReportService(compRepo, arenaRepo, categoryRepo, athleteRepo, matchRepo)

	// 1. Summary
	summary, err := reportSvc.GetSummary(compID)
	if err != nil {
		t.Fatalf("GetSummary failed: %v", err)
	}
	if summary.TotalMatches != 2 || summary.MatchesByStatus["finished"] != 1 || summary.MatchesByStatus["running"] != 1 {
		t.Errorf("summary match stats unexpected: %+v", summary)
	}
	if summary.TotalArenas != 1 || summary.TotalAthletes != 2 || summary.TotalCategories != 1 {
		t.Errorf("summary counts unexpected: %+v", summary)
	}

	// 2. Medals
	medals, err := reportSvc.GetMedalStandings(compID)
	if err != nil {
		t.Fatalf("GetMedalStandings failed: %v", err)
	}
	if len(medals) != 2 {
		t.Fatalf("expected 2 teams in standings, got %d", len(medals))
	}
	// Winner is ath-1 (Kontingen Jatim) -> 1 Gold
	// Runner up is ath-2 (Kontingen Jabar) -> 1 Silver
	if medals[0].Team != "Kontingen Jatim" || medals[0].Gold != 1 {
		t.Errorf("expected Kontingen Jatim with 1 gold at rank 1, got %+v", medals[0])
	}
	if medals[1].Team != "Kontingen Jabar" || medals[1].Silver != 1 {
		t.Errorf("expected Kontingen Jabar with 1 silver at rank 2, got %+v", medals[1])
	}

	// 3. Export
	exportData, err := reportSvc.ExportCompetition(compID)
	if err != nil {
		t.Fatalf("ExportCompetition failed: %v", err)
	}
	if exportData.Competition.ID != compID {
		t.Errorf("export competition ID mismatch: %s", exportData.Competition.ID)
	}
	if len(exportData.Medals) != 2 || len(exportData.Categories) != 1 || len(exportData.Matches) != 2 {
		t.Errorf("export collection lengths unexpected: %+v", exportData)
	}
}

func TestOfficialHandler_HTTP(t *testing.T) {
	repo := newMockOfficialRepo()
	svc := service.NewOfficialService(repo)
	h := &handler.OfficialHandler{Service: svc}

	// 1. Create Official via HTTP
	createBody := strings.NewReader(`{"name":"Operator 1","email":"op1@silat.id","password":"test_password_1","role":"operator"}`)
	req := httptest.NewRequest("POST", "/api/v1/officials", createBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Login via HTTP with wrong password
	loginFailBody := strings.NewReader(`{"email":"op1@silat.id","password":"wrong"}`)
	reqFail := httptest.NewRequest("POST", "/api/v1/auth/login", loginFailBody)
	reqFail.Header.Set("Content-Type", "application/json")
	wFail := httptest.NewRecorder()
	h.Login(wFail, reqFail)

	if wFail.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", wFail.Code)
	}

	// 3. Login via HTTP success
	loginSuccessBody := strings.NewReader(`{"email":"op1@silat.id","password":"test_password_1"}`)
	reqSuccess := httptest.NewRequest("POST", "/api/v1/auth/login", loginSuccessBody)
	reqSuccess.Header.Set("Content-Type", "application/json")
	wSuccess := httptest.NewRecorder()
	h.Login(wSuccess, reqSuccess)

	if wSuccess.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", wSuccess.Code, wSuccess.Body.String())
	}

	var loginResp struct {
		Success bool `json:"success"`
		Data    struct {
			Token string           `json:"token"`
			User  *domain.Official `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wSuccess.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	if loginResp.Data.Token == "" || loginResp.Data.User.Email != "op1@silat.id" {
		t.Errorf("unexpected login response data: %+v", loginResp)
	}

	// 4. List via HTTP
	reqList := httptest.NewRequest("GET", "/api/v1/officials", nil)
	wList := httptest.NewRecorder()
	h.List(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Errorf("expected 200 OK for list, got %d", wList.Code)
	}
}

func TestReportHandler_HTTP(t *testing.T) {
	compMap := make(map[string]*domain.Competition)
	compRepo := &mockLifecycleCompRepo{comps: compMap}
	compID := "comp-http-1"
	comp := &domain.Competition{
		ID:        compID,
		Name:      "Test Kejurnas",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	_ = compRepo.Create(comp)

	arenaRepo := newMockArenaRepo()
	categoryRepo := &mockCategoryRepo{categories: make(map[string]*domain.Category)}
	athleteRepo := &mockAthleteRepo{athletes: make(map[string]*domain.Athlete)}
	matchRepo := &mockReportMatchRepo{}

	reportSvc := service.NewReportService(compRepo, arenaRepo, categoryRepo, athleteRepo, matchRepo)
	h := &handler.ReportHandler{Service: reportSvc}

	// 1. Summary
	reqSum := httptest.NewRequest("GET", "/api/v1/competitions/"+compID+"/report/summary", nil)
	reqSum.SetPathValue("comp_id", compID)
	wSum := httptest.NewRecorder()
	h.Summary(wSum, reqSum)
	if wSum.Code != http.StatusOK {
		t.Errorf("expected 200 OK for summary, got %d: %s", wSum.Code, wSum.Body.String())
	}

	// 2. Medals
	reqMedals := httptest.NewRequest("GET", "/api/v1/competitions/"+compID+"/report/medals", nil)
	reqMedals.SetPathValue("comp_id", compID)
	wMedals := httptest.NewRecorder()
	h.Medals(wMedals, reqMedals)
	if wMedals.Code != http.StatusOK {
		t.Errorf("expected 200 OK for medals, got %d: %s", wMedals.Code, wMedals.Body.String())
	}

	// 3. Export
	reqExport := httptest.NewRequest("GET", "/api/v1/competitions/"+compID+"/report/export", nil)
	reqExport.SetPathValue("comp_id", compID)
	wExport := httptest.NewRecorder()
	h.Export(wExport, reqExport)
	if wExport.Code != http.StatusOK {
		t.Errorf("expected 200 OK for export, got %d: %s", wExport.Code, wExport.Body.String())
	}
}

