package bracket

import (
	"math"

	"github.com/google/uuid"
	"github.com/herros27/digital-scoring/internal/store"
)

// nextPowerOfTwo mencari angka pangkat dua terdekat >= n (min 2)
func nextPowerOfTwo(n int) int {
	if n <= 2 {
		return 2
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

// GenerateSlots menyusun struktur pohon bagan dari ronde 1 hingga Final
func GenerateSlots(competitionID, weightClass, gender string, participantIDs []string) []store.BracketSlot {
	numParticipants := len(participantIDs)
	if numParticipants < 2 {
		return nil
	}

	bracketSize := nextPowerOfTwo(numParticipants)
	totalRounds := int(math.Log2(float64(bracketSize)))

	var allSlots []store.BracketSlot

	// Map untuk menyimpan slot per ronde agar mudah menghubungkan next_slot_id:
	// key: "round:slot_index" -> generated UUID
	slotIDMap := make(map[string]string)

	// 1. Generate ID untuk seluruh slot di setiap ronde terlebih dahulu
	for r := 1; r <= totalRounds; r++ {
		numSlotsInRound := bracketSize / (1 << (r - 1))
		for i := 0; i < numSlotsInRound; i++ {
			key := slotKey(r, i)
			slotIDMap[key] = uuid.New().String()
		}
	}

	// 2. Susun slot dengan relasi next_slot_id
	for r := 1; r <= totalRounds; r++ {
		numSlotsInRound := bracketSize / (1 << (r - 1))

		for i := 0; i < numSlotsInRound; i++ {
			id := slotIDMap[slotKey(r, i)]
			var nextSlotID *string

			if r < totalRounds {
				// Dua slot yang bersebelahan (i=0 & i=1) menuju ke parent slot di ronde berikutnya (parent_i = i / 2)
				parentKey := slotKey(r+1, i/2)
				if parentID, ok := slotIDMap[parentKey]; ok {
					nextSlotID = &parentID
				}
			}

			slot := store.BracketSlot{
				ID:            id,
				CompetitionID: competitionID,
				WeightClass:   weightClass,
				Gender:        gender,
				Round:         r,
				SlotIndex:     i,
				NextSlotID:    nextSlotID,
			}

			// Khusus Ronde 1: Masukkan peserta & tentukan BYE
			if r == 1 {
				if i < numParticipants {
					pID := participantIDs[i]
					slot.ParticipantID = &pID
				} else {
					// Slot kosong di ronde 1 dianggap BYE
					slot.IsBye = true
				}
			}

			allSlots = append(allSlots, slot)
		}
	}

	// 3. Auto-advance untuk peserta yang mendapat BYE di Ronde 1
	// Jika lawannya BYE, maka peserta tersebut otomatis jadi winner di ronde 1
	// dan participant_id nya langsung diset ke slot di ronde 2.
	for i := 0; i < len(allSlots); i++ {
		if allSlots[i].Round == 1 && allSlots[i].ParticipantID != nil {
			opponentIndex := allSlots[i].SlotIndex ^ 1 // jika 0 lawannya 1, jika 2 lawannya 3
			for j := 0; j < len(allSlots); j++ {
				if allSlots[j].Round == 1 && allSlots[j].SlotIndex == opponentIndex && allSlots[j].IsBye {
					allSlots[i].WinnerID = allSlots[i].ParticipantID
					break
				}
			}
		}
	}

	return allSlots
}

func slotKey(round, index int) string {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte(string(rune(round*1000+index)))).String()
}
