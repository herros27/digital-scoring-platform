package consensus

import (
	"sync"
	"time"
)

const (
	windowDuration = 3 * time.Second // sliding time window
	majorityCount  = 2               // dari 3 juri, minimal 2 harus setuju
)

type ConsensusResult struct {
	Accepted  bool
	ScoreType ScoreType
	Target    Target
	Round     int
	Count     int // berapa juri yang setuju
}

type Engine struct {
	mu     sync.Mutex
	buffer []JudgeInput
}

func NewEngine() *Engine {
	return &Engine{
		buffer: make([]JudgeInput, 0),
	}
}

func (e *Engine) Submit(input JudgeInput) *ConsensusResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	// cek duplikat — satu juri tidak bisa submit 2x dalam window
	for _, existing := range e.buffer {
		if existing.JudgeID == input.JudgeID &&
			existing.ScoreType == input.ScoreType &&
			existing.Target == input.Target &&
			time.Since(existing.Timestamp) < windowDuration {
			return nil // ignore duplikat
		}
	}

	// tambah ke buffer
	e.buffer = append(e.buffer, input)

	// buang input yang sudah expired
	e.cleanup()

	// cek majority dalam window
	return e.evaluate(input)
}

func (e *Engine) evaluate(latest JudgeInput) *ConsensusResult {
	count := 0
	for _, inp := range e.buffer {
		if inp.ScoreType == latest.ScoreType &&
			inp.Target == latest.Target &&
			inp.Round == latest.Round &&
			time.Since(inp.Timestamp) < windowDuration {
			count++
		}
	}

	if count >= majorityCount {
		// clear buffer untuk kombinasi ini supaya tidak trigger ulang
		e.clearMatched(latest)
		return &ConsensusResult{
			Accepted:  true,
			ScoreType: latest.ScoreType,
			Target:    latest.Target,
			Round:     latest.Round,
			Count:     count,
		}
	}

	return &ConsensusResult{Accepted: false}
}

func (e *Engine) cleanup() {
	valid := e.buffer[:0]
	for _, inp := range e.buffer {
		if time.Since(inp.Timestamp) < windowDuration {
			valid = append(valid, inp)
		}
	}
	e.buffer = valid
}

func (e *Engine) clearMatched(ref JudgeInput) {
	remaining := e.buffer[:0]
	for _, inp := range e.buffer {
		if !(inp.ScoreType == ref.ScoreType &&
			inp.Target == ref.Target &&
			inp.Round == ref.Round) {
			remaining = append(remaining, inp)
		}
	}
	e.buffer = remaining
}

// PenaltyAuthority — hanya referee yang bisa kasih pelanggaran
func CanIssuePenalty(role JudgeRole) bool {
	return role == RoleReferee
}