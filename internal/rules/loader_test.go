package rules

import (
	"testing"
)

func TestRules_LoaderAndScoring(t *testing.T) {
	// 1. Load IPSI
	rsIPSI, err := LoadRuleset("ipsi")
	if err != nil {
		t.Fatalf("failed to load IPSI ruleset: %v", err)
	}
	if rsIPSI.Name != "IPSI" {
		t.Errorf("expected name IPSI, got %s", rsIPSI.Name)
	}
	if rsIPSI.Scoring.TechniquePoints != 1 || rsIPSI.Scoring.KnockdownPoints != 2 || rsIPSI.Scoring.FallPoints != 2 {
		t.Errorf("unexpected scoring points configuration: %+v", rsIPSI.Scoring)
	}

	// 2. Load PERSILAT
	rsPersilat, err := LoadRuleset("persilat")
	if err != nil {
		t.Fatalf("failed to load PERSILAT ruleset: %v", err)
	}
	if rsPersilat.Name != "PERSILAT" {
		t.Errorf("expected name PERSILAT, got %s", rsPersilat.Name)
	}

	// 3. Unknown ruleset
	_, errUnknown := LoadRuleset("unknown-ruleset-xyz")
	if errUnknown == nil {
		t.Errorf("expected error loading non-existent ruleset, got nil")
	}
}
