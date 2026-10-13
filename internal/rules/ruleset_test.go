package rules

import (
	"strings"
	"testing"
)

func TestDefaults_LegacyRulesetDerivesTechniquesAndDefaults(t *testing.T) {
	sc := ScoringRule{TechniquePoints: 1, KnockdownPoints: 2, FallPoints: 3, RoundDuration: 120, TotalRounds: 3}

	tech := sc.EffectiveTechniques()
	if len(tech) != 3 || tech[2].Type != "fall" || tech[2].Points != 3 {
		t.Fatalf("teknik turunan salah: %+v", tech)
	}
	if p, ok := sc.Penalty("teguran_2"); !ok || p.Points != 2 {
		t.Fatalf("teguran_2 bawaan harus -2: %+v", p)
	}
	if p, ok := sc.Penalty("peringatan_3"); !ok || p.Action != ActionDisqualify {
		t.Fatalf("peringatan_3 bawaan harus diskualifikasi: %+v", p)
	}
	if c := sc.EffectiveConsensus(); c != (ConsensusRule{Judges: 3, Majority: 2, WindowMs: 3000}) {
		t.Fatalf("konsensus bawaan salah: %+v", c)
	}
	if err := sc.Validate(); err != nil {
		t.Fatalf("ruleset lama harus valid: %v", err)
	}
}

func TestPresets_AreValidAndIncludeCatchFall(t *testing.T) {
	for _, name := range []string{"ipsi", "persilat"} {
		rs, err := LoadRuleset(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := rs.Scoring.Validate(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		if tech, ok := rs.Scoring.Technique("catch_fall"); !ok || tech.Points != 4 {
			t.Errorf("%s: catch_fall harus 4 poin: %+v", name, tech)
		}
	}
}

func TestEffectiveConsensus_PartialOverrideKeepsDefaults(t *testing.T) {
	sc := ScoringRule{Consensus: &ConsensusRule{Judges: 5, Majority: 3}}
	c := sc.EffectiveConsensus()
	if c.Judges != 5 || c.Majority != 3 || c.WindowMs != 3000 {
		t.Fatalf("%+v", c)
	}
}

func TestValidate_Rejects(t *testing.T) {
	base := func() ScoringRule { return ScoringRule{RoundDuration: 120, TotalRounds: 3} }
	cases := map[string]func(*ScoringRule){
		"durasi nol":         func(s *ScoringRule) { s.RoundDuration = 0 },
		"kode ganda":         func(s *ScoringRule) { s.Techniques = []TechniqueRule{{"a", "A", 1}, {"a", "B", 2}} },
		"kode salah":         func(s *ScoringRule) { s.Penalties = []PenaltyRule{{Type: "Teguran 1", Label: "x"}} },
		"nilai teknik nol":   func(s *ScoringRule) { s.Techniques = []TechniqueRule{{"a", "A", 0}} },
		"kode dicadangkan":   func(s *ScoringRule) { s.Techniques = []TechniqueRule{{"undo", "Undo", 1}} },
		"potongan negatif":   func(s *ScoringRule) { s.Penalties = []PenaltyRule{{Type: "x", Label: "X", Points: -1}} },
		"action tak dikenal": func(s *ScoringRule) { s.Penalties = []PenaltyRule{{Type: "x", Label: "X", Action: "ban"}} },
		"label kosong":       func(s *ScoringRule) { s.Penalties = []PenaltyRule{{Type: "x"}} },
		"juri 6":             func(s *ScoringRule) { s.Consensus = &ConsensusRule{Judges: 6, Majority: 2, WindowMs: 3000} },
		"mayoritas > juri":   func(s *ScoringRule) { s.Consensus = &ConsensusRule{Judges: 3, Majority: 4, WindowMs: 3000} },
		"window terlalu kecil": func(s *ScoringRule) {
			s.Consensus = &ConsensusRule{Judges: 3, Majority: 2, WindowMs: 100}
		},
	}
	for name, mutate := range cases {
		sc := base()
		mutate(&sc)
		if err := sc.Validate(); err == nil {
			t.Errorf("%s: harus ditolak", name)
		} else if strings.TrimSpace(err.Error()) == "" {
			t.Errorf("%s: pesan kosong", name)
		}
	}
}

func TestEffectiveTiebreak_DefaultsFollowTechniqueValue(t *testing.T) {
	sc := ScoringRule{TechniquePoints: 1, KnockdownPoints: 2, FallPoints: 3}
	tb := sc.EffectiveTiebreak()
	if len(tb.Steps) != 2 || tb.Steps[0] != StepFewestPenalties || tb.Steps[1] != StepTechniquePriority {
		t.Fatalf("langkah bawaan salah: %+v", tb.Steps)
	}
	want := []string{"fall", "knockdown", "technique"}
	for i, w := range want {
		if tb.TechniquePriority[i] != w {
			t.Fatalf("prioritas teknik = %v, want %v", tb.TechniquePriority, want)
		}
	}
	if len(tb.ManualMethods) != 3 {
		t.Fatalf("metode manual bawaan: %v", tb.ManualMethods)
	}
}

func TestValidate_Tiebreak(t *testing.T) {
	base := ScoringRule{RoundDuration: 120, TotalRounds: 3, TechniquePoints: 1, KnockdownPoints: 2, FallPoints: 3}
	bad := []TiebreakRule{
		{Steps: []string{"undian"}},
		{TechniquePriority: []string{"tidak_ada"}},
		{ManualMethods: []string{"sulap"}},
	}
	for _, tb := range bad {
		sc := base
		sc.Tiebreak = &tb
		if err := sc.Validate(); err == nil {
			t.Errorf("%+v harus ditolak", tb)
		}
	}
	ok := base
	ok.Tiebreak = &TiebreakRule{Steps: []string{StepTechniquePriority}, TechniquePriority: []string{"fall"}, ManualMethods: []string{MethodCoinToss}}
	if err := ok.Validate(); err != nil {
		t.Errorf("konfigurasi valid ditolak: %v", err)
	}
}

func TestOutcomes_DefaultsAndValidation(t *testing.T) {
	sc := ScoringRule{RoundDuration: 120, TotalRounds: 3}
	if got := len(sc.EffectiveOutcomes()); got != 4 {
		t.Fatalf("hasil khusus bawaan = %d, want 4", got)
	}
	if o, ok := sc.Outcome(OutcomeWalkover); !ok || o.Phase != PhaseBeforeStart {
		t.Fatalf("WO harus hanya sebelum mulai: %+v", o)
	}
	if o, _ := sc.Outcome(OutcomeWithdrawal); o.Phase != PhaseAny {
		t.Fatalf("undur diri bisa kapan saja: %+v", o)
	}
	bad := [][]OutcomeRule{
		{{Type: "menyerah", Label: "Menyerah", Phase: PhaseAny}},
		{{Type: OutcomeKO, Label: "KO", Phase: "kapan_saja"}},
		{{Type: OutcomeKO, Phase: PhaseAny}},
		{{Type: OutcomeKO, Label: "KO", Phase: PhaseAny}, {Type: OutcomeKO, Label: "KO2", Phase: PhaseAny}},
	}
	for _, out := range bad {
		s := sc
		s.Outcomes = out
		if err := s.Validate(); err == nil {
			t.Errorf("%+v harus ditolak", out)
		}
	}
	ok := sc
	ok.Outcomes = []OutcomeRule{{Type: OutcomeKO, Label: "KO", Phase: PhaseInMatch}}
	if err := ok.Validate(); err != nil {
		t.Errorf("valid ditolak: %v", err)
	}
}

func TestPresets_TiebreakOrderFollowsRegulation(t *testing.T) {
	want := []string{"catch_fall", "fall", "combo_1_2", "knockdown", "combo_1_1", "technique"}
	for _, name := range []string{"ipsi", "persilat"} {
		rs, err := LoadRuleset(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := rs.Scoring.Validate(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		got := rs.Scoring.EffectiveTiebreak().TechniquePriority
		if len(got) != len(want) {
			t.Fatalf("%s: prioritas = %v, want %v", name, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: prioritas = %v, want %v", name, got, want)
			}
		}
		if c, _ := rs.Scoring.Technique("combo_1_2"); c.Points != 3 {
			t.Errorf("%s: 1+2 harus 3 poin", name)
		}
		if c, _ := rs.Scoring.Technique("combo_1_1"); c.Points != 2 {
			t.Errorf("%s: 1+1 harus 2 poin", name)
		}
	}
}

func TestEffectiveTiebreak_DerivedOrderMatchesRegulationWithoutExplicitPriority(t *testing.T) {
	// ruleset tersimpan tanpa tiebreak (preset basis data hasil migrasi): turunan dari nilai + urutan daftar
	sc := PresetScoring()
	sc.Tiebreak = nil
	want := []string{"catch_fall", "fall", "combo_1_2", "knockdown", "combo_1_1", "technique"}
	got := sc.EffectiveTiebreak().TechniquePriority
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("prioritas turunan = %v, want %v", got, want)
		}
	}
}

func TestEffectivePointGap_DefaultOffAndCustom(t *testing.T) {
	cases := []struct {
		name string
		gap  *PointGapRule
		want PointGapRule
	}{
		{"kosong = bawaan remaja/dewasa mulai babak 2", nil, PointGapRule{Margin: 30, FromRound: 2}},
		{"margin 0 = nonaktif", &PointGapRule{Margin: 0}, PointGapRule{Margin: 0, FromRound: 2}},
		{"usia dini 20", &PointGapRule{Margin: 20}, PointGapRule{Margin: 20, FromRound: 2}},
		{"babak mulai diatur", &PointGapRule{Margin: 15, FromRound: 1}, PointGapRule{Margin: 15, FromRound: 1}},
	}
	for _, c := range cases {
		if got := (ScoringRule{PointGap: c.gap}).EffectivePointGap(); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestValidate_PointGap(t *testing.T) {
	base := ScoringRule{RoundDuration: 120, BreakDuration: 60, TotalRounds: 3}
	ok := []*PointGapRule{nil, {Margin: 0}, {Margin: 30}, {Margin: 20, FromRound: 3}}
	for _, g := range ok {
		s := base
		s.PointGap = g
		if err := s.Validate(); err != nil {
			t.Errorf("%+v harus valid: %v", g, err)
		}
	}
	bad := []*PointGapRule{{Margin: -1}, {Margin: 30, FromRound: -1}, {Margin: 30, FromRound: 4}}
	for _, g := range bad {
		s := base
		s.PointGap = g
		if err := s.Validate(); err == nil {
			t.Errorf("%+v harus ditolak", g)
		}
	}
}
