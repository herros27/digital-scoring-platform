package rules

import (
	"fmt"
	"regexp"
	"sort"
	"time"
)

// ScoringRule parameter pertandingan. Bagian `Penalties`, `Techniques`, dan `Consensus`
// bersifat opsional: bila kosong, nilai turunan/bawaan dipakai (lihat Effective*), sehingga
// ruleset lama yang hanya berisi nilai poin dan durasi tetap berjalan.
type ScoringRule struct {
	TechniquePoints  int `json:"technique_points"`
	KnockdownPoints  int `json:"knockdown_points"`
	FallPoints       int `json:"fall_points"`
	MaxPenalties     int `json:"max_penalties"`
	PenaltyThreshold int `json:"penalty_threshold"` // tiap N pelanggaran = 1 poin lawan
	RoundDuration    int `json:"round_duration"`    // detik
	BreakDuration    int `json:"break_duration"`    // detik
	TotalRounds      int `json:"total_rounds"`

	Penalties  []PenaltyRule   `json:"penalties,omitempty"`
	Techniques []TechniqueRule `json:"techniques,omitempty"`
	Consensus  *ConsensusRule  `json:"consensus,omitempty"`
	Tiebreak   *TiebreakRule   `json:"tiebreak,omitempty"`
	Outcomes   []OutcomeRule   `json:"outcomes,omitempty"`
}

// Jenis hasil khusus yang dapat dinyatakan wasit di luar skor, dan kapan boleh dinyatakan.
const (
	OutcomeKO         = "ko"         // Menang Mutlak: lawan tidak bangkit setelah serangan sah
	OutcomeTKO        = "tko"        // Menang Teknik: lawan tidak dapat melanjutkan
	OutcomeWalkover   = "walkover"   // Menang WO: lawan tidak hadir setelah panggilan
	OutcomeWithdrawal = "withdrawal" // Undur Diri: lawan/manajer mengundurkan diri

	PhaseBeforeStart = "before_start" // hanya sebelum ronde pertama dimulai
	PhaseInMatch     = "in_match"     // hanya setelah pertandingan dimulai
	PhaseAny         = "any"
)

// OutcomeRule satu jenis hasil khusus. `Type` menjadi `result_type` hasil pertandingan.
type OutcomeRule struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Phase string `json:"phase"`
}

// DefaultOutcomes hasil khusus bawaan menurut peraturan pertandingan.
func DefaultOutcomes() []OutcomeRule {
	return []OutcomeRule{
		{Type: OutcomeKO, Label: "Menang Mutlak (KO)", Phase: PhaseInMatch},
		{Type: OutcomeTKO, Label: "Menang Teknik (TKO)", Phase: PhaseInMatch},
		{Type: OutcomeWalkover, Label: "Menang WO", Phase: PhaseBeforeStart},
		{Type: OutcomeWithdrawal, Label: "Undur Diri", Phase: PhaseAny},
	}
}

// EffectiveOutcomes hasil khusus yang berlaku (bawaan bila ruleset tidak mendefinisikan).
func (s ScoringRule) EffectiveOutcomes() []OutcomeRule {
	if len(s.Outcomes) > 0 {
		return s.Outcomes
	}
	return DefaultOutcomes()
}

// Outcome mencari hasil khusus menurut kode.
func (s ScoringRule) Outcome(t string) (OutcomeRule, bool) {
	for _, o := range s.EffectiveOutcomes() {
		if o.Type == t {
			return o, true
		}
	}
	return OutcomeRule{}, false
}

// Langkah seri otomatis (dihitung server dari data pertandingan) dan metode manual (diputuskan wasit).
const (
	StepFewestPenalties   = "fewest_penalties"   // total potongan sanksi paling sedikit menang
	StepTechniquePriority = "technique_priority" // jumlah teknik terbanyak menurut urutan prioritas
	MethodRefereeDecision = "referee_decision"
	MethodExtraRound      = "extra_round"
	MethodLighterWeight   = "lighter_weight"
	MethodCoinToss        = "coin_toss"
)

// TiebreakRule aturan bila skor bersih seri setelah ronde terakhir.
// `Steps` dijalankan berurutan sampai ada pemenang. `TechniquePriority` = kode teknik dari yang
// paling bernilai; sudut dengan jumlah lebih banyak pada teknik paling awal yang berbeda menang.
// Bila semua langkah otomatis seri, wasit memilih satu dari `ManualMethods` dan menentukan pemenang.
type TiebreakRule struct {
	Steps             []string `json:"steps,omitempty"`
	TechniquePriority []string `json:"technique_priority,omitempty"`
	ManualMethods     []string `json:"manual_methods,omitempty"`
}

// ActionDisqualify: sanksi ini langsung mendiskualifikasi sudut yang dikenai.
const ActionDisqualify = "disqualify"

// PenaltyRule satu jenis sanksi: `Points` adalah potongan nilai bagi sudut yang dikenai.
type PenaltyRule struct {
	Type   string `json:"type"`
	Label  string `json:"label"`
	Points int    `json:"points"`
	Action string `json:"action,omitempty"`
}

// TechniqueRule satu jenis teknik yang dapat dinilai juri (`Type` = nilai `score_type`).
type TechniqueRule struct {
	Type   string `json:"type"`
	Label  string `json:"label"`
	Points int    `json:"points"`
}

// ConsensusRule aturan kesepakatan juri: `Majority` dari `Judges` juri harus menekan teknik
// yang sama pada sudut yang sama dalam `WindowMs`.
type ConsensusRule struct {
	Judges   int `json:"judges"`
	Majority int `json:"majority"`
	WindowMs int `json:"window_ms"`
}

type Ruleset struct {
	Name    string      `json:"name"`
	Version string      `json:"version"`
	Scoring ScoringRule `json:"scoring"`
}

// DefaultConsensus format juri 3 dengan mayoritas 2 dalam 3 detik (perilaku sebelum aturan ini dapat diatur).
func DefaultConsensus() ConsensusRule {
	return ConsensusRule{Judges: 3, Majority: 2, WindowMs: 3000}
}

// DefaultPenalties tabel sanksi bawaan: binaan tanpa potongan, teguran I -1, teguran II -2,
// peringatan I -5, peringatan II -10, peringatan III diskualifikasi. Cocokkan dengan regulasi event.
func DefaultPenalties() []PenaltyRule {
	return []PenaltyRule{
		{Type: "binaan_1", Label: "Binaan 1", Points: 0},
		{Type: "binaan_2", Label: "Binaan 2", Points: 0},
		{Type: "teguran_1", Label: "Teguran 1", Points: 1},
		{Type: "teguran_2", Label: "Teguran 2", Points: 2},
		{Type: "peringatan_1", Label: "Peringatan 1", Points: 5},
		{Type: "peringatan_2", Label: "Peringatan 2", Points: 10},
		{Type: "peringatan_3", Label: "Peringatan 3", Points: 0, Action: ActionDisqualify},
	}
}

// EffectivePenalties daftar sanksi yang berlaku (bawaan bila ruleset tidak mendefinisikan).
func (s ScoringRule) EffectivePenalties() []PenaltyRule {
	if len(s.Penalties) > 0 {
		return s.Penalties
	}
	return DefaultPenalties()
}

// EffectiveTechniques daftar teknik yang berlaku. Bila tidak didefinisikan, diturunkan dari
// nilai Pukulan/Tendangan/Jatuhan agar ruleset lama tetap bekerja.
func (s ScoringRule) EffectiveTechniques() []TechniqueRule {
	if len(s.Techniques) > 0 {
		return s.Techniques
	}
	return []TechniqueRule{
		{Type: "technique", Label: "Pukulan", Points: s.TechniquePoints},
		{Type: "knockdown", Label: "Tendangan", Points: s.KnockdownPoints},
		{Type: "fall", Label: "Jatuhan", Points: s.FallPoints},
	}
}

// EffectiveConsensus aturan konsensus yang berlaku; nilai yang tidak diisi memakai bawaan.
func (s ScoringRule) EffectiveConsensus() ConsensusRule {
	c := DefaultConsensus()
	if s.Consensus == nil {
		return c
	}
	if s.Consensus.Judges > 0 {
		c.Judges = s.Consensus.Judges
	}
	if s.Consensus.Majority > 0 {
		c.Majority = s.Consensus.Majority
	}
	if s.Consensus.WindowMs > 0 {
		c.WindowMs = s.Consensus.WindowMs
	}
	return c
}

// EffectiveTiebreak aturan seri yang berlaku. Bawaan: sanksi paling sedikit, lalu teknik terbanyak
// (urut dari nilai tertinggi), lalu keputusan manual wasit (babak tambahan, timbang ulang, undian koin).
func (s ScoringRule) EffectiveTiebreak() TiebreakRule {
	t := TiebreakRule{}
	if s.Tiebreak != nil {
		t = *s.Tiebreak
	}
	if len(t.Steps) == 0 {
		t.Steps = []string{StepFewestPenalties, StepTechniquePriority}
	}
	if len(t.TechniquePriority) == 0 {
		techs := append([]TechniqueRule(nil), s.EffectiveTechniques()...)
		sort.SliceStable(techs, func(i, j int) bool { return techs[i].Points > techs[j].Points })
		for _, tc := range techs {
			t.TechniquePriority = append(t.TechniquePriority, tc.Type)
		}
	}
	if len(t.ManualMethods) == 0 {
		t.ManualMethods = []string{MethodExtraRound, MethodLighterWeight, MethodCoinToss}
	}
	return t
}

// Window durasi window konsensus.
func (c ConsensusRule) Window() time.Duration { return time.Duration(c.WindowMs) * time.Millisecond }

// Penalty mencari sanksi menurut kode.
func (s ScoringRule) Penalty(t string) (PenaltyRule, bool) {
	for _, p := range s.EffectivePenalties() {
		if p.Type == t {
			return p, true
		}
	}
	return PenaltyRule{}, false
}

// Technique mencari teknik menurut kode.
func (s ScoringRule) Technique(t string) (TechniqueRule, bool) {
	for _, tech := range s.EffectiveTechniques() {
		if tech.Type == t {
			return tech, true
		}
	}
	return TechniqueRule{}, false
}

var codeRE = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

// Validate memeriksa seluruh parameter. Dipakai saat ruleset dibuat atau diubah.
func (s ScoringRule) Validate() error {
	if s.RoundDuration <= 0 || s.TotalRounds <= 0 {
		return fmt.Errorf("round_duration and total_rounds must be greater than 0")
	}
	if s.BreakDuration < 0 {
		return fmt.Errorf("break_duration must not be negative")
	}

	seen := map[string]bool{}
	for _, p := range s.Penalties {
		if !codeRE.MatchString(p.Type) {
			return fmt.Errorf("penalties: kode %q harus huruf kecil/angka/underscore (maks 32)", p.Type)
		}
		if seen[p.Type] {
			return fmt.Errorf("penalties: kode %q ganda", p.Type)
		}
		seen[p.Type] = true
		if p.Label == "" {
			return fmt.Errorf("penalties: label %q wajib diisi", p.Type)
		}
		if p.Points < 0 {
			return fmt.Errorf("penalties: potongan %q tidak boleh negatif", p.Type)
		}
		if p.Action != "" && p.Action != ActionDisqualify {
			return fmt.Errorf("penalties: action %q tidak dikenal (hanya %q)", p.Action, ActionDisqualify)
		}
	}

	seen = map[string]bool{}
	for _, t := range s.Techniques {
		if !codeRE.MatchString(t.Type) {
			return fmt.Errorf("techniques: kode %q harus huruf kecil/angka/underscore (maks 32)", t.Type)
		}
		if t.Type == "undo" || t.Type == "penalty" {
			return fmt.Errorf("techniques: kode %q dicadangkan sistem", t.Type)
		}
		if seen[t.Type] {
			return fmt.Errorf("techniques: kode %q ganda", t.Type)
		}
		seen[t.Type] = true
		if t.Label == "" {
			return fmt.Errorf("techniques: label %q wajib diisi", t.Type)
		}
		if t.Points <= 0 {
			return fmt.Errorf("techniques: nilai %q harus lebih dari 0", t.Type)
		}
	}

	seen = map[string]bool{}
	for _, o := range s.Outcomes {
		switch o.Type {
		case OutcomeKO, OutcomeTKO, OutcomeWalkover, OutcomeWithdrawal:
		default:
			return fmt.Errorf("outcomes: jenis %q tidak dikenal (%s, %s, %s, %s)", o.Type, OutcomeKO, OutcomeTKO, OutcomeWalkover, OutcomeWithdrawal)
		}
		if seen[o.Type] {
			return fmt.Errorf("outcomes: jenis %q ganda", o.Type)
		}
		seen[o.Type] = true
		if o.Label == "" {
			return fmt.Errorf("outcomes: label %q wajib diisi", o.Type)
		}
		if o.Phase != PhaseBeforeStart && o.Phase != PhaseInMatch && o.Phase != PhaseAny {
			return fmt.Errorf("outcomes: phase %q tidak dikenal (%s, %s, %s)", o.Phase, PhaseBeforeStart, PhaseInMatch, PhaseAny)
		}
	}

	if s.Tiebreak != nil {
		for _, st := range s.Tiebreak.Steps {
			if st != StepFewestPenalties && st != StepTechniquePriority {
				return fmt.Errorf("tiebreak: langkah %q tidak dikenal (%s, %s)", st, StepFewestPenalties, StepTechniquePriority)
			}
		}
		for _, tp := range s.Tiebreak.TechniquePriority {
			if _, ok := s.Technique(tp); !ok {
				return fmt.Errorf("tiebreak: technique_priority %q bukan teknik pada ruleset", tp)
			}
		}
		for _, m := range s.Tiebreak.ManualMethods {
			if m != MethodExtraRound && m != MethodLighterWeight && m != MethodCoinToss {
				return fmt.Errorf("tiebreak: metode manual %q tidak dikenal (%s, %s, %s)", m, MethodExtraRound, MethodLighterWeight, MethodCoinToss)
			}
		}
	}

	if s.Consensus != nil {
		c := s.EffectiveConsensus()
		if c.Judges < 1 || c.Judges > 5 {
			return fmt.Errorf("consensus: judges harus 1-5")
		}
		if c.Majority < 1 || c.Majority > c.Judges {
			return fmt.Errorf("consensus: majority harus 1 sampai judges")
		}
		if c.WindowMs < 500 || c.WindowMs > 10000 {
			return fmt.Errorf("consensus: window_ms harus 500-10000")
		}
	}
	return nil
}
