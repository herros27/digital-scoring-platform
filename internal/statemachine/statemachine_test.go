package statemachine

import (
	"testing"
)

func TestStateMachine_NormalLifecycleFlow(t *testing.T) {
	sm := New("match-lifecycle")

	// 1. Initial state
	if sm.Current() != StateIdle {
		t.Fatalf("expected StateIdle, got: %s", sm.Current())
	}
	if sm.IsInRound() {
		t.Errorf("idle should not be in round")
	}
	if sm.IsFinished() {
		t.Errorf("idle should not be finished")
	}

	// 2. Transition to Ready
	if err := sm.Transition(EventStart); err != nil {
		t.Fatalf("failed transition to ready: %v", err)
	}
	if sm.Current() != StateReady {
		t.Errorf("expected StateReady, got %s", sm.Current())
	}

	// 3. Transition to Round 1
	if err := sm.Transition(EventStart); err != nil {
		t.Fatalf("failed transition to round_1: %v", err)
	}
	if sm.Current() != StateRound1 {
		t.Errorf("expected StateRound1, got %s", sm.Current())
	}
	if !sm.IsInRound() {
		t.Errorf("round_1 should be in round")
	}
	if sm.CurrentRoundNumber() != 1 {
		t.Errorf("expected round number 1, got %d", sm.CurrentRoundNumber())
	}

	// 4. Round 1 End -> Break 1
	if err := sm.Transition(EventRoundEnd); err != nil {
		t.Fatalf("failed transition to break_1: %v", err)
	}
	if sm.Current() != StateBreak1 {
		t.Errorf("expected StateBreak1, got %s", sm.Current())
	}
	if sm.IsInRound() {
		t.Errorf("break_1 should not be in round")
	}

	// 5. Break 1 End -> Round 2
	if err := sm.Transition(EventBreakEnd); err != nil {
		t.Fatalf("failed transition to round_2: %v", err)
	}
	if sm.Current() != StateRound2 {
		t.Errorf("expected StateRound2, got %s", sm.Current())
	}
	if sm.CurrentRoundNumber() != 2 {
		t.Errorf("expected round number 2, got %d", sm.CurrentRoundNumber())
	}

	// 6. Round 2 End -> Break 2
	if err := sm.Transition(EventRoundEnd); err != nil {
		t.Fatalf("failed transition to break_2: %v", err)
	}
	if sm.Current() != StateBreak2 {
		t.Errorf("expected StateBreak2, got %s", sm.Current())
	}

	// 7. Break 2 End -> Round 3
	if err := sm.Transition(EventBreakEnd); err != nil {
		t.Fatalf("failed transition to round_3: %v", err)
	}
	if sm.Current() != StateRound3 {
		t.Errorf("expected StateRound3, got %s", sm.Current())
	}
	if sm.CurrentRoundNumber() != 3 {
		t.Errorf("expected round number 3, got %d", sm.CurrentRoundNumber())
	}

	// 8. Round 3 End -> Finished
	if err := sm.Transition(EventRoundEnd); err != nil {
		t.Fatalf("failed transition to finished: %v", err)
	}
	if sm.Current() != StateFinished {
		t.Errorf("expected StateFinished, got %s", sm.Current())
	}
	if !sm.IsFinished() {
		t.Errorf("finished state should return IsFinished == true")
	}

	// 9. Reset -> Idle
	if err := sm.Transition(EventReset); err != nil {
		t.Fatalf("failed reset to idle: %v", err)
	}
	if sm.Current() != StateIdle {
		t.Errorf("expected StateIdle after reset, got %s", sm.Current())
	}
}

func TestStateMachine_InvalidTransitions(t *testing.T) {
	sm := New("match-invalid")

	// In Idle, cannot execute EventRoundEnd
	if err := sm.Transition(EventRoundEnd); err == nil {
		t.Errorf("expected error transitioning Idle with EventRoundEnd, got nil")
	}

	// In Idle, cannot execute EventBreakEnd
	if err := sm.Transition(EventBreakEnd); err == nil {
		t.Errorf("expected error transitioning Idle with EventBreakEnd, got nil")
	}
}

func TestStateMachine_EarlyFinishAndCancel(t *testing.T) {
	// Early finish in Round 1 (e.g. KO or 30 point difference)
	sm1 := NewWithState("match-ko", StateRound1)
	if err := sm1.Transition(EventFinish); err != nil {
		t.Fatalf("expected early finish to succeed: %v", err)
	}
	if sm1.Current() != StateFinished {
		t.Errorf("expected StateFinished, got %s", sm1.Current())
	}

	// Cancel from Ready state
	sm2 := NewWithState("match-cancel", StateReady)
	if err := sm2.Transition(EventCancel); err != nil {
		t.Fatalf("expected cancel from ready to succeed: %v", err)
	}
	if sm2.Current() != StateCancelled {
		t.Errorf("expected StateCancelled, got %s", sm2.Current())
	}
	if !sm2.IsFinished() {
		t.Errorf("cancelled match should be considered finished")
	}
}
