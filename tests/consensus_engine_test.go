package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/consensus"
)

func TestConsensusEngine_MajorityReached(t *testing.T) {
	engine := consensus.NewEngine()
	now := time.Now()

	// Judge 1 submits red kick round 1
	in1 := consensus.JudgeInput{
		JudgeID:   "judge-1",
		Role:      consensus.RoleJudge1,
		MatchID:   "match-101",
		Round:     1,
		ScoreType: consensus.ScoreTechnique,
		Target:    consensus.TargetRed,
		Timestamp: now,
	}
	res1 := engine.Submit(in1)
	if res1 == nil || res1.Accepted {
		t.Fatalf("expected res1 to be rejected (waiting for consensus), got: %+v", res1)
	}

	// Judge 2 submits same red kick round 1 within window (100ms later)
	in2 := consensus.JudgeInput{
		JudgeID:   "judge-2",
		Role:      consensus.RoleJudge2,
		MatchID:   "match-101",
		Round:     1,
		ScoreType: consensus.ScoreTechnique,
		Target:    consensus.TargetRed,
		Timestamp: now.Add(100 * time.Millisecond),
	}
	res2 := engine.Submit(in2)
	if res2 == nil || !res2.Accepted {
		t.Fatalf("expected res2 to achieve consensus (2 out of 3), got: %+v", res2)
	}
	if res2.Count != 2 {
		t.Errorf("expected count 2, got %d", res2.Count)
	}
	if res2.Target != consensus.TargetRed || res2.ScoreType != consensus.ScoreTechnique {
		t.Errorf("unexpected target or scoreType in consensus result: %+v", res2)
	}
}

func TestConsensusEngine_DuplicateJudgeIgnored(t *testing.T) {
	engine := consensus.NewEngine()
	now := time.Now()

	in1 := consensus.JudgeInput{
		JudgeID:   "judge-1",
		Role:      consensus.RoleJudge1,
		MatchID:   "match-102",
		Round:     1,
		ScoreType: consensus.ScoreTechnique,
		Target:    consensus.TargetBlue,
		Timestamp: now,
	}
	res1 := engine.Submit(in1)
	if res1 == nil || res1.Accepted {
		t.Fatalf("res1 should not be accepted: %+v", res1)
	}

	// Judge 1 presses button again within window
	dupIn1 := consensus.JudgeInput{
		JudgeID:   "judge-1",
		Role:      consensus.RoleJudge1,
		MatchID:   "match-102",
		Round:     1,
		ScoreType: consensus.ScoreTechnique,
		Target:    consensus.TargetBlue,
		Timestamp: now.Add(50 * time.Millisecond),
	}
	resDup := engine.Submit(dupIn1)
	if resDup != nil {
		t.Fatalf("expected nil for duplicate submission from same judge, got: %+v", resDup)
	}
}

func TestConsensusEngine_MismatchScoreTypeAndTarget(t *testing.T) {
	engine := consensus.NewEngine()
	now := time.Now()

	// Judge 1: red technique
	res1 := engine.Submit(consensus.JudgeInput{
		JudgeID:   "judge-1",
		Role:      consensus.RoleJudge1,
		Round:     1,
		ScoreType: consensus.ScoreTechnique,
		Target:    consensus.TargetRed,
		Timestamp: now,
	})
	if res1 == nil || res1.Accepted {
		t.Fatalf("unexpected res1: %+v", res1)
	}

	// Judge 2: blue technique (different target)
	res2 := engine.Submit(consensus.JudgeInput{
		JudgeID:   "judge-2",
		Role:      consensus.RoleJudge2,
		Round:     1,
		ScoreType: consensus.ScoreTechnique,
		Target:    consensus.TargetBlue,
		Timestamp: now.Add(20 * time.Millisecond),
	})
	if res2 == nil || res2.Accepted {
		t.Fatalf("unexpected res2: %+v", res2)
	}

	// Judge 3: red knockdown (different score type)
	res3 := engine.Submit(consensus.JudgeInput{
		JudgeID:   "judge-3",
		Role:      consensus.RoleJudge3,
		Round:     1,
		ScoreType: consensus.ScoreKnockdown,
		Target:    consensus.TargetRed,
		Timestamp: now.Add(40 * time.Millisecond),
	})
	if res3 == nil || res3.Accepted {
		t.Fatalf("unexpected res3: %+v", res3)
	}
}

func TestConsensusEngine_ExpiredWindow(t *testing.T) {
	engine := consensus.NewEngine()

	// Judge 1 submitted 4 seconds ago (> 3 second window)
	oldTime := time.Now().Add(-4 * time.Second)
	res1 := engine.Submit(consensus.JudgeInput{
		JudgeID:   "judge-1",
		Role:      consensus.RoleJudge1,
		Round:     1,
		ScoreType: consensus.ScoreFall,
		Target:    consensus.TargetRed,
		Timestamp: oldTime,
	})
	if res1 == nil || res1.Accepted {
		t.Fatalf("unexpected res1: %+v", res1)
	}

	// Judge 2 submits now
	res2 := engine.Submit(consensus.JudgeInput{
		JudgeID:   "judge-2",
		Role:      consensus.RoleJudge2,
		Round:     1,
		ScoreType: consensus.ScoreFall,
		Target:    consensus.TargetRed,
		Timestamp: time.Now(),
	})
	if res2 == nil || res2.Accepted {
		t.Fatalf("expected res2 not accepted because Judge 1 expired from window, got: %+v", res2)
	}
}

func TestConsensusEngine_DifferentRounds(t *testing.T) {
	engine := consensus.NewEngine()
	now := time.Now()

	// Judge 1 votes round 1
	engine.Submit(consensus.JudgeInput{
		JudgeID:   "judge-1",
		Role:      consensus.RoleJudge1,
		Round:     1,
		ScoreType: consensus.ScoreTechnique,
		Target:    consensus.TargetRed,
		Timestamp: now,
	})

	// Judge 2 votes round 2
	res2 := engine.Submit(consensus.JudgeInput{
		JudgeID:   "judge-2",
		Role:      consensus.RoleJudge2,
		Round:     2,
		ScoreType: consensus.ScoreTechnique,
		Target:    consensus.TargetRed,
		Timestamp: now.Add(50 * time.Millisecond),
	})

	if res2 == nil || res2.Accepted {
		t.Fatalf("expected not accepted for mismatched rounds, got: %+v", res2)
	}
}

func TestConsensusEngine_ConcurrentSubmissions(t *testing.T) {
	engine := consensus.NewEngine()
	var wg sync.WaitGroup

	// Concurrently submit votes from 3 judges
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(judgeNum int) {
			defer wg.Done()
			engine.Submit(consensus.JudgeInput{
				JudgeID:   "judge-concurrent",
				Role:      consensus.JudgeRole("judge_" + string(rune('0'+judgeNum))),
				Round:     1,
				ScoreType: consensus.ScoreTechnique,
				Target:    consensus.TargetRed,
				Timestamp: time.Now(),
			})
		}(i)
	}

	wg.Wait()
}
