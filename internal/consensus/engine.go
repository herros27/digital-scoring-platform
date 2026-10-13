package consensus

import (
	"sync"
	"time"

	"github.com/herros27/digital-scoring/internal/metrics"
)

const (
	defaultWindow   = 3 * time.Second // sliding time window bawaan
	defaultMajority = 2               // dari 3 juri, minimal 2 harus setuju
)

type ConsensusResult struct {
	Accepted  bool
	ScoreType ScoreType
	Target    Target
	Round     int
	Count     int // berapa juri yang setuju
}

// Majority jumlah juri yang harus sepakat.
func (e *Engine) Majority() int { return e.majority }

// Window lama jendela kesepakatan.
func (e *Engine) Window() time.Duration { return e.window }

type Engine struct {
	mu       sync.Mutex
	buffer   []JudgeInput
	window   time.Duration
	majority int
}

// NewEngine engine dengan aturan bawaan (2 juri sepakat dalam 3 detik).
func NewEngine() *Engine { return NewEngineWith(defaultWindow, defaultMajority) }

// NewEngineWith engine dengan window dan jumlah juri minimal yang harus sepakat sesuai ruleset.
func NewEngineWith(window time.Duration, majority int) *Engine {
	if window <= 0 {
		window = defaultWindow
	}
	if majority <= 0 {
		majority = defaultMajority
	}
	return &Engine{
		buffer:   make([]JudgeInput, 0),
		window:   window,
		majority: majority,
	}
}

func (e *Engine) Submit(input JudgeInput) *ConsensusResult {
	start := time.Now()
	e.mu.Lock()
	defer e.mu.Unlock()

	// cek duplikat — satu juri tidak bisa submit 2x dalam window
	for _, existing := range e.buffer {
		if existing.JudgeID == input.JudgeID &&
			existing.ScoreType == input.ScoreType &&
			existing.Target == input.Target &&
			time.Since(existing.Timestamp) < e.window {
			metrics.RecordConsensusDuration("duplicate", time.Since(start))
			return nil // ignore duplikat
		}
	}

	// tambah ke buffer
	e.buffer = append(e.buffer, input)

	// buang input yang sudah expired
	e.cleanup()

	// cek majority dalam window
	res := e.evaluate(input)
	status := "pending"
	if res != nil && res.Accepted {
		status = "accepted"
	}
	metrics.RecordConsensusDuration(status, time.Since(start))
	return res
}

// Retracted input yang ditarik kembali oleh juri.
type Retracted struct {
	ScoreType ScoreType
	Target    Target
	Round     int
	Remaining int // juri lain yang masih menunggu dengan teknik dan sudut yang sama
}

// Retract menarik input terakhir juri pada sudut itu yang masih di dalam window dan belum menjadi poin.
// false bila tidak ada yang bisa ditarik (sudah terkonfirmasi, kedaluwarsa, atau memang belum menekan).
func (e *Engine) Retract(judgeID string, target Target, round int) (Retracted, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cleanup()
	for i := len(e.buffer) - 1; i >= 0; i-- {
		in := e.buffer[i]
		if in.JudgeID != judgeID || in.Target != target || in.Round != round {
			continue
		}
		e.buffer = append(e.buffer[:i], e.buffer[i+1:]...)
		remaining := 0
		for _, other := range e.buffer {
			if other.ScoreType == in.ScoreType && other.Target == in.Target && other.Round == in.Round {
				remaining++
			}
		}
		return Retracted{ScoreType: in.ScoreType, Target: in.Target, Round: in.Round, Remaining: remaining}, true
	}
	return Retracted{}, false
}

func (e *Engine) evaluate(latest JudgeInput) *ConsensusResult {
	count := 0
	for _, inp := range e.buffer {
		if inp.ScoreType == latest.ScoreType &&
			inp.Target == latest.Target &&
			inp.Round == latest.Round &&
			time.Since(inp.Timestamp) < e.window {
			count++
		}
	}

	if count >= e.majority {
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

	// belum cukup: laporkan kemajuan (jumlah juri yang sudah sepakat dalam window)
	return &ConsensusResult{
		Accepted:  false,
		ScoreType: latest.ScoreType,
		Target:    latest.Target,
		Round:     latest.Round,
		Count:     count,
	}
}

func (e *Engine) cleanup() {
	valid := e.buffer[:0]
	for _, inp := range e.buffer {
		if time.Since(inp.Timestamp) < e.window {
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
