package rules

import (
	"fmt"
	"regexp"
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
