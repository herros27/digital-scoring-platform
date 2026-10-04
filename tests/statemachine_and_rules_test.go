package tests

import (
	"testing"

	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/statemachine"
)

func TestStateMachine_NormalLifecycleFlow(t *testing.T) {
	sm := statemachine.New("match-lifecycle")

	// 1. Initial state
	if sm.Current() != statemachine.StateIdle {
		t.Fatalf("expected StateIdle, got: %s", sm.Current())
	}
	if sm.IsInRound() {
		t.Errorf("idle should not be in round")
	}
	if sm.IsFinished() {
		t.Errorf("idle should not be finished")
	}

	// 2. Transition to Ready
	if err := sm.Transition(statemachine.EventStart); err != nil {
		t.Fatalf("failed transition to ready: %v", err)
	}
	if sm.Current() != statemachine.StateReady {
		t.Errorf("expected StateReady, got %s", sm.Current())
	}

	// 3. Transition to Round 1
	if err := sm.Transition(statemachine.EventStart); err != nil {
		t.Fatalf("failed transition to round_1: %v", err)
	}
	if sm.Current() != statemachine.StateRound1 {
		t.Errorf("expected StateRound1, got %s", sm.Current())
	}
	if !sm.IsInRound() {
		t.Errorf("round_1 should be in round")
	}
	if sm.CurrentRoundNumber() != 1 {
		t.Errorf("expected round number 1, got %d", sm.CurrentRoundNumber())
	}

	// 4. Round 1 End -> Break 1
	if err := sm.Transition(statemachine.EventRoundEnd); err != nil {
		t.Fatalf("failed transition to break_1: %v", err)
	}
	if sm.Current() != statemachine.StateBreak1 {
		t.Errorf("expected StateBreak1, got %s", sm.Current())
	}
	if sm.IsInRound() {
		t.Errorf("break_1 should not be in round")
	}

	// 5. Break 1 End -> Round 2
	if err := sm.Transition(statemachine.EventBreakEnd); err != nil {
		t.Fatalf("failed transition to round_2: %v", err)
	}
	if sm.Current() != statemachine.StateRound2 {
		t.Errorf("expected StateRound2, got %s", sm.Current())
	}
	if sm.CurrentRoundNumber() != 2 {
		t.Errorf("expected round number 2, got %d", sm.CurrentRoundNumber())
	}

	// 6. Round 2 End -> Break 2
	if err := sm.Transition(statemachine.EventRoundEnd); err != nil {
		t.Fatalf("failed transition to break_2: %v", err)
	}
	if sm.Current() != statemachine.StateBreak2 {
		t.Errorf("expected StateBreak2, got %s", sm.Current())
	}

	// 7. Break 2 End -> Round 3
	if err := sm.Transition(statemachine.EventBreakEnd); err != nil {
		t.Fatalf("failed transition to round_3: %v", err)
	}
	if sm.Current() != statemachine.StateRound3 {
		t.Errorf("expected StateRound3, got %s", sm.Current())
	}
	if sm.CurrentRoundNumber() != 3 {
		t.Errorf("expected round number 3, got %d", sm.CurrentRoundNumber())
	}

	// 8. Round 3 End -> Finished
	if err := sm.Transition(statemachine.EventRoundEnd); err != nil {
		t.Fatalf("failed transition to finished: %v", err)
	}
	if sm.Current() != statemachine.StateFinished {
		t.Errorf("expected StateFinished, got %s", sm.Current())
	}
	if !sm.IsFinished() {
		t.Errorf("finished state should return IsFinished == true")
	}

	// 9. Reset -> Idle
	if err := sm.Transition(statemachine.EventReset); err != nil {
		t.Fatalf("failed reset to idle: %v", err)
	}
	if sm.Current() != statemachine.StateIdle {
		t.Errorf("expected StateIdle after reset, got %s", sm.Current())
	}
}

func TestStateMachine_InvalidTransitions(t *testing.T) {
	sm := statemachine.New("match-invalid")

	// In Idle, cannot execute EventRoundEnd
	if err := sm.Transition(statemachine.EventRoundEnd); err == nil {
		t.Errorf("expected error transitioning Idle with EventRoundEnd, got nil")
	}

	// In Idle, cannot execute EventBreakEnd
	if err := sm.Transition(statemachine.EventBreakEnd); err == nil {
		t.Errorf("expected error transitioning Idle with EventBreakEnd, got nil")
	}
}

func TestStateMachine_EarlyFinishAndCancel(t *testing.T) {
	// Early finish in Round 1 (e.g. KO or 30 point difference)
	sm1 := statemachine.NewWithState("match-ko", statemachine.StateRound1)
	if err := sm1.Transition(statemachine.EventFinish); err != nil {
		t.Fatalf("expected early finish to succeed: %v", err)
	}
	if sm1.Current() != statemachine.StateFinished {
		t.Errorf("expected StateFinished, got %s", sm1.Current())
	}

	// Cancel from Ready state
	sm2 := statemachine.NewWithState("match-cancel", statemachine.StateReady)
	if err := sm2.Transition(statemachine.EventCancel); err != nil {
		t.Fatalf("expected cancel from ready to succeed: %v", err)
	}
	if sm2.Current() != statemachine.StateCancelled {
		t.Errorf("expected StateCancelled, got %s", sm2.Current())
	}
	if !sm2.IsFinished() {
		t.Errorf("cancelled match should be considered finished")
	}
}

func TestRules_LoaderAndScoring(t *testing.T) {
	// 1. Load IPSI
	rsIPSI, err := rules.LoadRuleset("ipsi")
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
	rsPersilat, err := rules.LoadRuleset("persilat")
	if err != nil {
		t.Fatalf("failed to load PERSILAT ruleset: %v", err)
	}
	if rsPersilat.Name != "PERSILAT" {
		t.Errorf("expected name PERSILAT, got %s", rsPersilat.Name)
	}

	// 3. Unknown ruleset
	_, errUnknown := rules.LoadRuleset("unknown-ruleset-xyz")
	if errUnknown == nil {
		t.Errorf("expected error loading non-existent ruleset, got nil")
	}
}
