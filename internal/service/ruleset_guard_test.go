package service

import (
	"errors"
	"testing"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/store"
)

type guardCompRepo struct {
	comp    *domain.Competition
	started bool
}

func (g *guardCompRepo) Create(*domain.Competition) error { return nil }
func (g *guardCompRepo) GetByID(string) (*domain.Competition, error) {
	c := *g.comp
	return &c, nil
}
func (g *guardCompRepo) List() ([]domain.Competition, error) { return nil, nil }
func (g *guardCompRepo) UpdateStatus(string, string) error   { return nil }
func (g *guardCompRepo) Update(c *domain.Competition) error  { g.comp = c; return nil }
func (g *guardCompRepo) HasStartedMatches(string) (bool, error) {
	return g.started, nil
}

func ptr(s string) *string { return &s }

const rsA = "11111111-1111-1111-1111-111111111111"
const rsB = "22222222-2222-2222-2222-222222222222"

func TestUpdateCompetition_RulesetLockedAfterMatchesStart(t *testing.T) {
	repo := &guardCompRepo{comp: &domain.Competition{ID: "c1", Name: "Kejurda", RulesetID: ptr(rsA), RulesetName: "IPSI"}}
	svc := NewCompetitionService(repo)

	// Belum ada partai berjalan: boleh ganti.
	if _, err := svc.UpdateCompetition("c1", &domain.Competition{RulesetID: ptr(rsB), RulesetName: "PERSILAT"}); err != nil {
		t.Fatalf("change before start: %v", err)
	}

	// Sudah ada partai berjalan: ganti ruleset ditolak (409), tapi mengisi ulang nilai yang sama tetap boleh.
	repo.started = true
	_, err := svc.UpdateCompetition("c1", &domain.Competition{RulesetID: ptr(rsA)})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
	if _, err := svc.UpdateCompetition("c1", &domain.Competition{RulesetID: ptr(rsB), Name: "Kejurda 2026"}); err != nil {
		t.Fatalf("same ruleset + other fields must pass: %v", err)
	}
}

type guardRulesetRepo struct{ users []string }

func (g *guardRulesetRepo) Create(*store.Ruleset) error            { return nil }
func (g *guardRulesetRepo) Update(*store.Ruleset) error            { return nil }
func (g *guardRulesetRepo) Delete(string) error                    { return nil }
func (g *guardRulesetRepo) List() ([]store.Ruleset, error)         { return nil, nil }
func (g *guardRulesetRepo) GetByID(string) (*store.Ruleset, error) { return nil, nil }
func (g *guardRulesetRepo) GetActive() (*store.Ruleset, error)     { return nil, nil }
func (g *guardRulesetRepo) CompetitionsUsing(string) ([]string, error) {
	return g.users, nil
}

func TestDeleteRuleset_BlockedWhileInUse(t *testing.T) {
	repo := &guardRulesetRepo{users: []string{"Kejurda 2026"}}
	svc := NewRulesetService(repo)
	err := svc.DeleteRuleset(rsA)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
	repo.users = nil
	if err := svc.DeleteRuleset(rsA); err != nil {
		t.Fatalf("unused ruleset must delete: %v", err)
	}
}

type fakeAssignRepo struct {
	mockCategoryRepo
	got []domain.CategoryRulesetAssignment
}

func (f *fakeAssignRepo) AssignRulesets(_ string, items []domain.CategoryRulesetAssignment) (int64, error) {
	f.got = items
	return int64(len(items)), nil
}

func TestAssignRulesets_Validation(t *testing.T) {
	repo := &fakeAssignRepo{mockCategoryRepo: mockCategoryRepo{categories: map[string]*domain.Category{}}}
	exists := map[string]bool{rsA: true}
	svc := NewCategoryService(repo).WithRulesetLookup(func(id string) bool { return exists[id] })

	if _, err := svc.AssignRulesets("c1", nil); err == nil {
		t.Error("empty assignments must fail")
	}
	// harus tepat satu pemilih
	if _, err := svc.AssignRulesets("c1", []domain.CategoryRulesetAssignment{{RulesetID: ptr(rsA)}}); err == nil {
		t.Error("missing selector must fail")
	}
	if _, err := svc.AssignRulesets("c1", []domain.CategoryRulesetAssignment{{AgeGroup: "usia_dini_2", CategoryID: "x", RulesetID: ptr(rsA)}}); err == nil {
		t.Error("both selectors must fail")
	}
	if _, err := svc.AssignRulesets("c1", []domain.CategoryRulesetAssignment{{AgeGroup: "bukan_golongan", RulesetID: ptr(rsA)}}); err == nil {
		t.Error("unknown age group must fail")
	}
	if _, err := svc.AssignRulesets("c1", []domain.CategoryRulesetAssignment{{AgeGroup: "usia_dini_2", RulesetID: ptr("bukan-uuid")}}); err == nil {
		t.Error("non-uuid ruleset must fail")
	}
	if _, err := svc.AssignRulesets("c1", []domain.CategoryRulesetAssignment{{AgeGroup: "usia_dini_2", RulesetID: ptr(rsB)}}); err == nil {
		t.Error("unknown ruleset must fail")
	}
	// valid + string kosong = kembali ke ruleset kejuaraan
	if _, err := svc.AssignRulesets("c1", []domain.CategoryRulesetAssignment{
		{AgeGroup: "usia_dini_2", RulesetID: ptr(rsA)},
		{AgeGroup: "usia_dini_1", RulesetID: ptr("")},
	}); err != nil {
		t.Fatalf("valid: %v", err)
	}
	if repo.got[1].RulesetID != nil {
		t.Error("empty ruleset_id must become nil (follow competition ruleset)")
	}
}
