package rules

import (
	"encoding/json"
	"fmt"
	"os"
)

// presetScoring aturan lengkap preset resmi: nilai teknik (termasuk tangkapan+jatuhan 1+3 = 4),
// tabel sanksi, dan konsensus 2 dari 3 juri. Cocokkan dengan peraturan pertandingan event.
func presetScoring() ScoringRule {
	return ScoringRule{
		TechniquePoints:  1,
		KnockdownPoints:  2,
		FallPoints:       3,
		MaxPenalties:     5,
		PenaltyThreshold: 3,
		RoundDuration:    120,
		BreakDuration:    60,
		TotalRounds:      3,
		Techniques: []TechniqueRule{
			{Type: "technique", Label: "Pukulan", Points: 1},
			{Type: "knockdown", Label: "Tendangan", Points: 2},
			{Type: "fall", Label: "Jatuhan", Points: 3},
			{Type: "catch_fall", Label: "Tangkapan + Jatuhan", Points: 4},
		},
		Penalties: DefaultPenalties(),
		Consensus: &ConsensusRule{Judges: 3, Majority: 2, WindowMs: 3000},
	}
}

// PresetScoring mengembalikan salinan aturan lengkap preset bawaan ("ipsi" atau "persilat").
func PresetScoring() ScoringRule { return presetScoring() }

var defaults = map[string]func() Ruleset{
	"ipsi":     func() Ruleset { return Ruleset{Name: "IPSI", Version: "2024", Scoring: presetScoring()} },
	"persilat": func() Ruleset { return Ruleset{Name: "PERSILAT", Version: "2024", Scoring: presetScoring()} },
}

func LoadRuleset(name string) (*Ruleset, error) {
	// coba dari file JSON dulu
	path := fmt.Sprintf("rulesets/%s.json", name)
	if data, err := os.ReadFile(path); err == nil {
		var rs Ruleset
		if err := json.Unmarshal(data, &rs); err != nil {
			return nil, fmt.Errorf("failed to parse ruleset %s: %w", name, err)
		}
		return &rs, nil
	}

	// fallback ke default hardcoded
	if mk, ok := defaults[name]; ok {
		rs := mk()
		return &rs, nil
	}

	return nil, fmt.Errorf("ruleset '%s' not found", name)
}
