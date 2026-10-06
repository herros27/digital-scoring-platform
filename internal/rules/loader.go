package rules

import (
	"encoding/json"
	"fmt"
	"os"
)

var defaults = map[string]Ruleset{
	"ipsi": {
		Name:    "IPSI",
		Version: "2024",
		Scoring: ScoringRule{
			TechniquePoints:  1,
			KnockdownPoints:  2,
			FallPoints:       3,
			MaxPenalties:     5,
			PenaltyThreshold: 3,
			RoundDuration:    120,
			BreakDuration:    60,
			TotalRounds:      3,
		},
	},
	"persilat": {
		Name:    "PERSILAT",
		Version: "2024",
		Scoring: ScoringRule{
			TechniquePoints:  1,
			KnockdownPoints:  2,
			FallPoints:       3,
			MaxPenalties:     5,
			PenaltyThreshold: 3,
			RoundDuration:    120,
			BreakDuration:    60,
			TotalRounds:      3,
		},
	},
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
	if rs, ok := defaults[name]; ok {
		return &rs, nil
	}

	return nil, fmt.Errorf("ruleset '%s' not found", name)
}
