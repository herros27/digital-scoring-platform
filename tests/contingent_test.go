package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/api/handler"
	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

// In-memory mock for ContingentRepository
type mockContingentRepo struct {
	contingents map[string]*domain.Contingent
	passwords   map[string]string
}

func newMockContingentRepo() *mockContingentRepo {
	return &mockContingentRepo{
		contingents: make(map[string]*domain.Contingent),
		passwords:   make(map[string]string),
	}
}

func (m *mockContingentRepo) Create(c *domain.Contingent, hash string) error {
	if c.ID == "" {
		c.ID = "cont-" + c.Email
	}
	c.CreatedAt = time.Now()
	m.contingents[c.ID] = c
	m.passwords[c.Email] = hash
	return nil
}

func (m *mockContingentRepo) GetByID(id string) (*domain.Contingent, error) {
	if c, ok := m.contingents[id]; ok {
		return c, nil
	}
	return nil, nil
}

func (m *mockContingentRepo) GetByEmail(email string) (*domain.Contingent, string, error) {
	for _, c := range m.contingents {
		if c.Email == email {
			return c, m.passwords[email], nil
		}
	}
	return nil, "", nil
}

func (m *mockContingentRepo) UpdatePassword(id, hash string) error {
	c, ok := m.contingents[id]
	if !ok {
		return sql.ErrNoRows
	}
	m.passwords[c.Email] = hash
	return nil
}

func (m *mockContingentRepo) List() ([]domain.Contingent, error) {
	var list []domain.Contingent
	for _, c := range m.contingents {
		list = append(list, *c)
	}
	return list, nil
}

// In-memory mock for ParticipantManager and AthleteRepository
type mockAthleteParticipantRepo struct {
	athletes     map[string]*domain.Athlete
	participants map[string]*domain.Participant
}

func newMockAthleteParticipantRepo() *mockAthleteParticipantRepo {
	return &mockAthleteParticipantRepo{
		athletes:     make(map[string]*domain.Athlete),
		participants: make(map[string]*domain.Participant),
	}
}

func (m *mockAthleteParticipantRepo) Create(a *domain.Athlete) error {
	if a.ID == "" {
		a.ID = "ath-test-uuid"
	}
	a.CreatedAt = time.Now()
	m.athletes[a.ID] = a
	return nil
}

func (m *mockAthleteParticipantRepo) GetByID(id string) (*domain.Athlete, error) {
	if a, ok := m.athletes[id]; ok {
		return a, nil
	}
	return nil, nil
}

func (m *mockAthleteParticipantRepo) List() ([]domain.Athlete, error) {
	var list []domain.Athlete
	for _, a := range m.athletes {
		list = append(list, *a)
	}
	return list, nil
}

func (m *mockAthleteParticipantRepo) RegisterToCompetition(competitionID, athleteID, weightClass, gender string) error {
	p := &domain.Participant{
		ID:                 "part-" + athleteID,
		CompetitionID:      competitionID,
		AthleteID:          athleteID,
		WeightClass:        weightClass,
		Gender:             gender,
		RegistrationStatus: "registered",
		CreatedAt:          time.Now(),
	}
	m.participants[p.ID] = p
	return nil
}

func (m *mockAthleteParticipantRepo) RegisterParticipant(competitionID, athleteID, contingentID, weightClass, gender, status string) error {
	p := &domain.Participant{
		ID:                 "part-" + athleteID,
		CompetitionID:      competitionID,
		AthleteID:          athleteID,
		ContingentID:       contingentID,
		WeightClass:        weightClass,
		Gender:             gender,
		RegistrationStatus: status,
		CreatedAt:          time.Now(),
	}
	m.participants[p.ID] = p
	return nil
}

func (m *mockAthleteParticipantRepo) UpdateParticipantStatus(participantID, status string) error {
	if p, ok := m.participants[participantID]; ok {
		p.RegistrationStatus = status
		return nil
	}
	return domain.ErrNotFound
}

func (m *mockAthleteParticipantRepo) ListParticipants(competitionID, status string) ([]domain.Participant, error) {
	var list []domain.Participant
	for _, p := range m.participants {
		if p.CompetitionID == competitionID {
			if status == "" || p.RegistrationStatus == status {
				list = append(list, *p)
			}
		}
	}
	return list, nil
}

func (m *mockAthleteParticipantRepo) ListByCompetition(competitionID string) ([]domain.Athlete, error) {
	var list []domain.Athlete
	for _, p := range m.participants {
		if p.CompetitionID == competitionID {
			if a, ok := m.athletes[p.AthleteID]; ok {
				list = append(list, *a)
			}
		}
	}
	return list, nil
}

func TestContingent_RegisterAndLogin(t *testing.T) {
	repo := newMockContingentRepo()
	svc := service.NewContingentService(repo)

	// 1. Register Contingent
	c, err := svc.Register("Tapak Suci Sleman", "Budi Santoso", "tapaksuci@example.com", "081234567890", "mock_secret_1")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if c.Name != "Tapak Suci Sleman" || c.PicName != "Budi Santoso" {
		t.Fatalf("unexpected contingent fields: %+v", c)
	}

	// 2. Duplicate email rejected
	_, err = svc.Register("Other Team", "Someone", "tapaksuci@example.com", "08999", "mock_secret_1")
	if err == nil {
		t.Fatalf("expected conflict error for duplicate email")
	}

	// 3. Login wrong password
	_, _, err = svc.Login("tapaksuci@example.com", "wrongpass")
	if err == nil {
		t.Fatalf("expected error on invalid password")
	}

	// 4. Login success
	token, loggedIn, err := svc.Login("tapaksuci@example.com", "mock_secret_1")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loggedIn.ID != c.ID {
		t.Fatalf("expected id %s, got %s", c.ID, loggedIn.ID)
	}

	// 5. Parse token and check role
	claims, err := auth.ParseToken(token)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	if claims.Role != "contingent" {
		t.Fatalf("expected role 'contingent', got '%s'", claims.Role)
	}
	if claims.JudgePinID != c.ID {
		t.Fatalf("expected claim id '%s', got '%s'", c.ID, claims.JudgePinID)
	}
}

func TestContingent_AthleteRegistrationAndVerificationWorkflow(t *testing.T) {
	athRepo := newMockAthleteParticipantRepo()
	athSvc := service.NewAthleteService(athRepo)
	athHandler := &handler.AthleteHandler{Service: athSvc}

	compID := "comp-jogja-open"

	// 1. Simulate Contingent Token
	contingentToken, err := auth.GenerateToken("cont-123", compID, "", "contingent", 2*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate contingent token: %v", err)
	}
	claims, err := auth.ParseToken(contingentToken)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	// 2. Contingent registers an athlete
	regBody, _ := json.Marshal(map[string]string{
		"name":         "Ahmad Silat",
		"gender":       "male",
		"team":         "Tapak Suci Sleman",
		"weight_class": "A",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/competitions/"+compID+"/athletes", bytes.NewReader(regBody))
	req.SetPathValue("comp_id", compID)
	req = req.WithContext(auth.ContextWithClaims(req.Context(), claims))
	w := httptest.NewRecorder()

	athHandler.RegisterToCompetition(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]any
	json.Unmarshal(w.Body.Bytes(), &res)
	data := res["data"].(map[string]any)

	// Status should automatically be 'pending' for contingent registration
	if data["status"] != "pending" {
		t.Fatalf("expected status 'pending', got '%v'", data["status"])
	}
	if data["contingent_id"] != "cont-123" {
		t.Fatalf("expected contingent_id 'cont-123', got '%v'", data["contingent_id"])
	}

	partID := "part-" + data["athlete_id"].(string)

	// 3. Verify Participant Listing with filter ?status=pending
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/competitions/"+compID+"/participants?status=pending", nil)
	reqList.SetPathValue("comp_id", compID)
	wList := httptest.NewRecorder()
	athHandler.ListParticipants(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", wList.Code)
	}

	var listRes map[string]any
	json.Unmarshal(wList.Body.Bytes(), &listRes)
	parts := listRes["data"].([]any)
	if len(parts) != 1 {
		t.Fatalf("expected 1 pending participant, got %d", len(parts))
	}

	// 4. Admin verifies the participant
	verifyBody, _ := json.Marshal(map[string]string{
		"status": "verified",
	})
	reqVerify := httptest.NewRequest(http.MethodPut, "/api/v1/competitions/"+compID+"/participants/"+partID+"/verify", bytes.NewReader(verifyBody))
	reqVerify.SetPathValue("comp_id", compID)
	reqVerify.SetPathValue("id", partID)
	wVerify := httptest.NewRecorder()

	athHandler.VerifyParticipant(wVerify, reqVerify)
	if wVerify.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", wVerify.Code, wVerify.Body.String())
	}

	// 5. Check status is now 'verified'
	partList, _ := athSvc.ListParticipants(compID, "verified")
	if len(partList) != 1 {
		t.Fatalf("expected 1 verified participant, got %d", len(partList))
	}
	if partList[0].RegistrationStatus != "verified" {
		t.Fatalf("expected status 'verified', got '%s'", partList[0].RegistrationStatus)
	}
}

type mockBracketRepo struct {
	slots map[string]*store.BracketSlot
}

func (m *mockBracketRepo) CreateBatch(slots []store.BracketSlot) error {
	for i := range slots {
		m.slots[slots[i].ID] = &slots[i]
	}
	return nil
}

func (m *mockBracketRepo) GetByCompetitionCategory(competitionID, weightClass, gender string) ([]store.BracketSlot, error) {
	var list []store.BracketSlot
	for _, s := range m.slots {
		if s.CompetitionID == competitionID && s.WeightClass == weightClass && s.Gender == gender {
			list = append(list, *s)
		}
	}
	return list, nil
}

func (m *mockBracketRepo) GetByID(id string) (*store.BracketSlot, error) {
	if s, ok := m.slots[id]; ok {
		return s, nil
	}
	return nil, nil
}

func (m *mockBracketRepo) UpdateWinner(slotID, winnerID string) error {
	if s, ok := m.slots[slotID]; ok {
		s.WinnerID = &winnerID
		return nil
	}
	return nil
}

func (m *mockBracketRepo) DeleteByCompetitionCategory(competitionID, weightClass, gender string) error {
	for k, s := range m.slots {
		if s.CompetitionID == competitionID && s.WeightClass == weightClass && s.Gender == gender {
			delete(m.slots, k)
		}
	}
	return nil
}

func (m *mockBracketRepo) SetParticipant(slotID, participantID string) error {
	if s, ok := m.slots[slotID]; ok {
		s.ParticipantID = &participantID
		return nil
	}
	return nil
}

func TestContingent_BracketGenerationSafeguard(t *testing.T) {
	athRepo := newMockAthleteParticipantRepo()
	athSvc := service.NewAthleteService(athRepo)

	bracketRepo := &mockBracketRepo{slots: make(map[string]*store.BracketSlot)}
	bracketSvc := service.NewBracketService(bracketRepo)
	bracketHandler := &handler.BracketHandler{Service: bracketSvc, AthleteService: athSvc}

	compID := "comp-safeguard"

	// Register 2 participants: 1 verified, 1 pending
	_ = athRepo.RegisterParticipant(compID, "ath-1", "cont-1", "A", "male", "verified")
	_ = athRepo.RegisterParticipant(compID, "ath-2", "cont-2", "A", "male", "pending")

	// Attempt to generate bracket with explicit list containing the pending one
	reqBody, _ := json.Marshal(map[string]any{
		"weight_class":    "A",
		"gender":          "male",
		"participant_ids": []string{"part-ath-1", "part-ath-2"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/competitions/"+compID+"/brackets/generate", bytes.NewReader(reqBody))
	req.SetPathValue("comp_id", compID)
	w := httptest.NewRecorder()

	bracketHandler.Generate(w, req)
	// Because 1 is pending, only 1 verified participant remains -> should fail (needs >= 2 verified)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request due to unverified participant, got %d", w.Code)
	}

	// Now verify the second participant
	_ = athSvc.VerifyParticipant("part-ath-2", "verified")

	// Retry bracket generation
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/competitions/"+compID+"/brackets/generate", bytes.NewReader(reqBody))
	req2.SetPathValue("comp_id", compID)
	w2 := httptest.NewRecorder()

	bracketHandler.Generate(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created after both verified, got %d: %s", w2.Code, w2.Body.String())
	}
}
