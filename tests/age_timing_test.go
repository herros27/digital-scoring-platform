package tests

import (
	"strings"
	"testing"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

// Waktu pertandingan per golongan umur ditentukan panitia di ruleset yang sama (bukan wajib 2x1,5): golongan yang
// dicantumkan memakai waktunya, golongan lain ikut nilai utama.
func TestMatchService_AgeTimingsFromRuleset(t *testing.T) {
	rulesetSvc := service.NewRulesetService(newMemoryRulesetRepo())
	rs, err := rulesetSvc.CreateRuleset(service.CreateRulesetInput{
		Name: "Kejurda", Version: "1",
		Scoring: rules.ScoringRule{
			TechniquePoints: 1, KnockdownPoints: 2, RoundDuration: 120, BreakDuration: 60, TotalRounds: 3,
			AgeTimings: map[string]rules.AgeTiming{
				"usia_dini_1": {RoundDuration: 120, BreakDuration: 30, TotalRounds: 2}, // 2 babak x 2 menit, atas pilihan panitia
				"usia_dini_2": {RoundDuration: 90, BreakDuration: 60, TotalRounds: 2},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	matchRepo := newMockMatchRepo()
	compRepo := &mockCompRepoWithRuleset{comps: map[string]*domain.Competition{"c": {ID: "c", RulesetID: &rs.ID}}}
	slots := map[string]string{"m-d1": "usia_dini_1", "m-d2": "usia_dini_2", "m-dw": "dewasa", "m-none": ""}
	ages := map[string]string{}
	for id, ag := range slots {
		matchRepo.matches[id] = &store.Match{ID: id, CompetitionID: "c", ArenaID: "arena-" + id, BracketSlotID: "slot-" + id, Status: "idle"}
		ages["slot-"+id] = ag
	}
	svc := service.NewMatchService(matchRepo, compRepo, hub.NewManager(), nil, nil, nil)
	svc.SetRulesetResolver(rulesetSvc)
	svc.SetCategoryRulesetLookup(fakeAgeLookup{rulesets: map[string]string{}, ages: ages})

	check := func(id string, rounds int, durMs int64) {
		t.Helper()
		if err := svc.LoadMatchToArena(id, "arena-"+id, ""); err != nil {
			t.Fatalf("load %s: %v", id, err)
		}
		orch, _ := svc.GetOrchestrator(id)
		snap := orch.Snapshot()
		if snap["total_rounds"] != rounds || snap["round_duration_ms"] != durMs {
			t.Errorf("%s: rounds=%v dur=%v, want %d/%d", id, snap["total_rounds"], snap["round_duration_ms"], rounds, durMs)
		}
	}
	check("m-d1", 2, 120000) // pengecualian bebas: 2 babak x 2 menit
	check("m-d2", 2, 90000)
	check("m-dw", 3, 120000) // golongan tanpa pengecualian ikut nilai utama
	check("m-none", 3, 120000)
}

func TestScoringRule_AgeTimingsValidation(t *testing.T) {
	base := rules.ScoringRule{TechniquePoints: 1, KnockdownPoints: 2, RoundDuration: 120, BreakDuration: 60, TotalRounds: 3}
	ok := base
	ok.AgeTimings = map[string]rules.AgeTiming{"usia_dini_1": {RoundDuration: 120, BreakDuration: 0, TotalRounds: 2}}
	if err := ok.Validate(); err != nil {
		t.Fatalf("pengecualian sah (istirahat 0 boleh): %v", err)
	}
	for name, mut := range map[string]map[string]rules.AgeTiming{
		"durasi nol":  {"usia_dini_1": {RoundDuration: 0, BreakDuration: 10, TotalRounds: 2}},
		"babak nol":   {"usia_dini_1": {RoundDuration: 90, BreakDuration: 10, TotalRounds: 0}},
		"istirahat -": {"usia_dini_1": {RoundDuration: 90, BreakDuration: -1, TotalRounds: 2}},
		"kode salah":  {"Usia Dini": {RoundDuration: 90, BreakDuration: 10, TotalRounds: 2}},
	} {
		bad := base
		bad.AgeTimings = mut
		if err := bad.Validate(); err == nil || !strings.Contains(err.Error(), "age_timings") {
			t.Errorf("%s: harus ditolak: %v", name, err)
		}
	}
}
