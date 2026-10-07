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
