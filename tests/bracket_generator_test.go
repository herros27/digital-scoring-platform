package tests

import (
	"fmt"
	"testing"

	"github.com/herros27/digital-scoring/internal/bracket"
	"github.com/herros27/digital-scoring/internal/store"
)

func TestBracketGenerator_FewerThanTwoParticipants(t *testing.T) {
	// 0 participants
	slots0 := bracket.GenerateSlots("comp-1", "A", "male", []string{})
	if slots0 != nil {
		t.Fatalf("expected nil for 0 participants, got %d slots", len(slots0))
	}

	// 1 participant
	slots1 := bracket.GenerateSlots("comp-1", "A", "male", []string{"p1"})
	if slots1 != nil {
		t.Fatalf("expected nil for 1 participant, got %d slots", len(slots1))
	}
}

func TestBracketGenerator_TwoParticipants(t *testing.T) {
	participants := []string{"p1", "p2"}
	slots := bracket.GenerateSlots("comp-1", "A", "male", participants)

	// 2 participants = 1 round (Final) with 2 slots
	if len(slots) != 2 {
		t.Fatalf("expected 2 slots for 2 participants, got %d", len(slots))
	}

	for _, s := range slots {
		if s.Round != 1 {
			t.Errorf("expected round 1, got %d", s.Round)
		}
		if s.NextSlotID != nil {
			t.Errorf("expected NextSlotID to be nil for final round, got %v", s.NextSlotID)
		}
		if s.ParticipantID == nil {
			t.Errorf("expected participant to be assigned")
		}
		if s.IsBye {
			t.Errorf("expected no BYE in 2-person bracket")
		}
	}
}

func TestBracketGenerator_FourParticipants(t *testing.T) {
	participants := []string{"p1", "p2", "p3", "p4"}
	slots := bracket.GenerateSlots("comp-1", "B", "female", participants)

	// 4 participants = 4 slots (round 1) + 2 slots (round 2/final) = 6 slots
	if len(slots) != 6 {
		t.Fatalf("expected 6 slots, got %d", len(slots))
	}

	r1Slots := filterSlotsByRound(slots, 1)
	r2Slots := filterSlotsByRound(slots, 2)

	if len(r1Slots) != 4 {
		t.Errorf("expected 4 slots in round 1, got %d", len(r1Slots))
	}
	if len(r2Slots) != 2 {
		t.Errorf("expected 2 slots in round 2, got %d", len(r2Slots))
	}

	// Verify all round 1 slots point to a round 2 slot
	r2SlotIDs := make(map[string]bool)
	for _, s := range r2Slots {
		r2SlotIDs[s.ID] = true
		if s.NextSlotID != nil {
			t.Errorf("expected final round slot nextSlotID to be nil")
		}
	}

	for _, s := range r1Slots {
		if s.NextSlotID == nil {
			t.Errorf("expected round 1 slot to have nextSlotID")
		} else if !r2SlotIDs[*s.NextSlotID] {
			t.Errorf("round 1 slot nextSlotID %s not found in round 2 slots", *s.NextSlotID)
		}
	}
}

func TestBracketGenerator_WithByes_ThreeParticipants(t *testing.T) {
	participants := []string{"p1", "p2", "p3"}
	slots := bracket.GenerateSlots("comp-1", "C", "male", participants)

	// bracketSize is nextPowerOfTwo(3) = 4 -> 4 slots in R1, 2 slots in R2 = 6 total slots
	if len(slots) != 6 {
		t.Fatalf("expected 6 slots, got %d", len(slots))
	}

	r1Slots := filterSlotsByRound(slots, 1)
	byesCount := 0
	for _, s := range r1Slots {
		if s.IsBye {
			byesCount++
		}
	}
	if byesCount != 1 {
		t.Errorf("expected exactly 1 BYE slot, got %d", byesCount)
	}

	// The participant paired with the BYE slot (index 2 paired with index 3) must be auto-advanced
	// SlotIndex 2 has opponent 3. Since 3 is BYE, slot 2 should have WinnerID set to its ParticipantID
	var autoAdvancedSlotFound bool
	for _, s := range r1Slots {
		if s.ParticipantID != nil && s.WinnerID != nil && *s.WinnerID == *s.ParticipantID {
			autoAdvancedSlotFound = true
			if *s.ParticipantID != "p3" {
				t.Errorf("expected p3 to be auto-advanced, got %s", *s.ParticipantID)
			}
		}
	}

	if !autoAdvancedSlotFound {
		t.Errorf("expected at least one auto-advanced participant due to BYE")
	}
}

func TestBracketGenerator_EightParticipants(t *testing.T) {
	var participants []string
	for i := 1; i <= 8; i++ {
		participants = append(participants, fmt.Sprintf("p%d", i))
	}

	slots := bracket.GenerateSlots("comp-1", "D", "male", participants)
	// 8 R1 + 4 R2 + 2 R3 = 14 slots
	if len(slots) != 14 {
		t.Fatalf("expected 14 slots for 8 participants, got %d", len(slots))
	}

	r1 := filterSlotsByRound(slots, 1)
	r2 := filterSlotsByRound(slots, 2)
	r3 := filterSlotsByRound(slots, 3)

	if len(r1) != 8 || len(r2) != 4 || len(r3) != 2 {
		t.Fatalf("unexpected slot distribution: r1=%d, r2=%d, r3=%d", len(r1), len(r2), len(r3))
	}
}

func filterSlotsByRound(slots []store.BracketSlot, round int) []store.BracketSlot {
	var res []store.BracketSlot
	for _, s := range slots {
		if s.Round == round {
			res = append(res, s)
		}
	}
	return res
}
