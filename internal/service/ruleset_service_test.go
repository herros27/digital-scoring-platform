package service

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/store"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeRulesetRepo struct{ items []store.Ruleset }

func (f *fakeRulesetRepo) Create(r *store.Ruleset) error  { f.items = append(f.items, *r); return nil }
func (f *fakeRulesetRepo) Update(r *store.Ruleset) error  { return nil }
func (f *fakeRulesetRepo) Delete(id string) error         { return nil }
func (f *fakeRulesetRepo) List() ([]store.Ruleset, error) { return f.items, nil }
func (f *fakeRulesetRepo) GetByID(id string) (*store.Ruleset, error) {
	for i := range f.items {
		if f.items[i].ID == id {
			return &f.items[i], nil
		}
	}
	return nil, sql.ErrNoRows
}
func (f *fakeRulesetRepo) GetActive() (*store.Ruleset, error) { return nil, sql.ErrNoRows }

func dbRuleset(id, name string, fall int) store.Ruleset {
	return store.Ruleset{ID: id, Name: name, Version: "1", Status: "active", Scoring: rules.ScoringRule{
		TechniquePoints: 1, KnockdownPoints: 2, FallPoints: fall, RoundDuration: 90, TotalRounds: 3,
	}}
}

func TestResolveRuleset_ByIDExactNameAndShortName(t *testing.T) {
	repo := &fakeRulesetRepo{items: []store.Ruleset{
		dbRuleset("11111111-1111-1111-1111-111111111111", "IPSI 2024", 3),
		dbRuleset("22222222-2222-2222-2222-222222222222", "PERSILAT International", 4),
	}}
	svc := NewRulesetService(repo)

	for _, id := range []string{"22222222-2222-2222-2222-222222222222", "PERSILAT International", "persilat", "PERSILAT"} {
		rs, err := svc.ResolveRuleset(id)
		if err != nil || rs.Scoring.FallPoints != 4 || rs.Scoring.RoundDuration != 90 {
			t.Fatalf("%q -> %+v, %v (ruleset basis data harus dipakai, bukan bawaan Go)", id, rs, err)
		}
	}
	if rs, err := svc.ResolveRuleset("ipsi"); err != nil || rs.Name != "IPSI 2024" {
		t.Fatalf("nama pendek ipsi: %+v, %v", rs, err)
	}
}

func TestCreateRuleset_ValidatesScoringSections(t *testing.T) {
	svc := NewRulesetService(&fakeRulesetRepo{})
	_, err := svc.CreateRuleset(CreateRulesetInput{Name: "Uji", Version: "1", Scoring: rules.ScoringRule{
		RoundDuration: 120, TotalRounds: 3,
		Techniques: []rules.TechniqueRule{{Type: "undo", Label: "Undo", Points: 1}},
	}})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("kode teknik dicadangkan harus ditolak sebagai input tidak valid: %v", err)
	}
	if _, err := svc.CreateRuleset(CreateRulesetInput{Name: "Uji", Version: "1", Scoring: rules.PresetScoring()}); err != nil {
		t.Fatalf("preset harus valid: %v", err)
	}
}

func TestValidRulesetID(t *testing.T) {
	empty, good, bad := "", "11111111-1111-1111-1111-111111111111", "ipsi"
	for _, ok := range []*string{nil, &empty, &good} {
		if err := validRulesetID(ok); err != nil {
			t.Errorf("harus valid: %v", err)
		}
	}
	if err := validRulesetID(&bad); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("non-UUID harus ditolak: %v", err)
	}
}

func TestMapRulesetDBError(t *testing.T) {
	dup := mapRulesetDBError(&pgconn.PgError{Code: "23505"})
	inUse := mapRulesetDBError(&pgconn.PgError{Code: "23503"})
	other := errors.New("boom")
	if !errors.Is(dup, domain.ErrConflict) || !errors.Is(inUse, domain.ErrConflict) {
		t.Fatalf("pelanggaran constraint harus menjadi ErrConflict: %v / %v", dup, inUse)
	}
	if mapRulesetDBError(other) != other {
		t.Error("galat lain tidak boleh diubah")
	}
}
