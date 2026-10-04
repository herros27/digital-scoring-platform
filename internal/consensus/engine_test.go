package consensus

import (
	"sync"
	"testing"
	"time"
)

func TestConsensusEngine_MajorityReached(t *testing.T) {
	engine := NewEngine()
	now := time.Now()

	// Judge 1 submits red kick round 1
	in1 := JudgeInput{
		JudgeID:   "judge-1",
		Role:      RoleJudge1,
		MatchID:   "match-101",
		Round:     1,
		ScoreType: ScoreTechnique,
		Target:    TargetRed,
		Timestamp: now,
	}
	res1 := engine.Submit(in1)
	if res1 == nil || res1.Accepted {
		t.Fatalf("expected res1 to be rejected (waiting for consensus), got: %+v", res1)
	}

	// Judge 2 submits same red kick round 1 within window (100ms later)
	in2 := JudgeInput{
		JudgeID:   "judge-2",
		Role:      RoleJudge2,
		MatchID:   "match-101",
		Round:     1,
		ScoreType: ScoreTechnique,
		Target:    TargetRed,
		Timestamp: now.Add(100 * time.Millisecond),
	}
	res2 := engine.Submit(in2)
	if res2 == nil || !res2.Accepted {
		t.Fatalf("expected res2 to achieve consensus (2 out of 3), got: %+v", res2)
	}
	if res2.Count != 2 {
		t.Errorf("expected count 2, got %d", res2.Count)
	}
	if res2.Target != TargetRed || res2.ScoreType != ScoreTechnique {
		t.Errorf("unexpected target or scoreType in consensus result: %+v", res2)
	}
}

func TestConsensusEngine_DuplicateJudgeIgnored(t *testing.T) {
	engine := NewEngine()
	now := time.Now()

	in1 := JudgeInput{
		JudgeID:   "judge-1",
		Role:      RoleJudge1,
		MatchID:   "match-102",
		Round:     1,
		ScoreType: ScoreTechnique,
		Target:    TargetBlue,
		Timestamp: now,
	}
	res1 := engine.Submit(in1)
	if res1 == nil || res1.Accepted {
		t.Fatalf("res1 should not be accepted: %+v", res1)
	}

	// Judge 1 presses button again within window
	dupIn1 := JudgeInput{
		JudgeID:   "judge-1",
		Role:      RoleJudge1,
		MatchID:   "match-102",
		Round:     1,
		ScoreType: ScoreTechnique,
		Target:    TargetBlue,
		Timestamp: now.Add(50 * time.Millisecond),
	}
	resDup := engine.Submit(dupIn1)
	if resDup != nil {
		t.Fatalf("expected nil for duplicate submission from same judge, got: %+v", resDup)
	}
}

func TestConsensusEngine_MismatchScoreTypeAndTarget(t *testing.T) {
	engine := NewEngine()
	now := time.Now()

	// Judge 1: red technique
	res1 := engine.Submit(JudgeInput{
		JudgeID:   "judge-1",
		Role:      RoleJudge1,
		Round:     1,
		ScoreType: ScoreTechnique,
		Target:    TargetRed,
		Timestamp: now,
	})
	if res1 == nil || res1.Accepted {
		t.Fatalf("unexpected res1: %+v", res1)
	}

	// Judge 2: blue technique (different target)
	res2 := engine.Submit(JudgeInput{
		JudgeID:   "judge-2",
		Role:      RoleJudge2,
		Round:     1,
		ScoreType: ScoreTechnique,
		Target:    TargetBlue,
		Timestamp: now.Add(20 * time.Millisecond),
	})
	if res2 == nil || res2.Accepted {
		t.Fatalf("unexpected res2: %+v", res2)
	}

	// Judge 3: red knockdown (different score type)
	res3 := engine.Submit(JudgeInput{
		JudgeID:   "judge-3",
		Role:      RoleJudge3,
		Round:     1,
		ScoreType: ScoreKnockdown,
		Target:    TargetRed,
		Timestamp: now.Add(40 * time.Millisecond),
	})
	if res3 == nil || res3.Accepted {
		t.Fatalf("unexpected res3: %+v", res3)
	}
}

func TestConsensusEngine_ExpiredWindow(t *testing.T) {
	engine := NewEngine()

	// Judge 1 submitted 4 seconds ago (> 3 second window)
	oldTime := time.Now().Add(-4 * time.Second)
	res1 := engine.Submit(JudgeInput{
		JudgeID:   "judge-1",
		Role:      RoleJudge1,
		Round:     1,
		ScoreType: ScoreFall,
		Target:    TargetRed,
		Timestamp: oldTime,
	})
	if res1 == nil || res1.Accepted {
		t.Fatalf("unexpected res1: %+v", res1)
	}

	// Judge 2 submits now
	res2 := engine.Submit(JudgeInput{
		JudgeID:   "judge-2",
		Role:      RoleJudge2,
		Round:     1,
		ScoreType: ScoreFall,
		Target:    TargetRed,
		Timestamp: time.Now(),
	})
	if res2 == nil || res2.Accepted {
		t.Fatalf("expected res2 not accepted because Judge 1 expired from window, got: %+v", res2)
	}
}

func TestConsensusEngine_DifferentRounds(t *testing.T) {
	engine := NewEngine()
	now := time.Now()

	// Judge 1 votes round 1
	engine.Submit(JudgeInput{
		JudgeID:   "judge-1",
		Role:      RoleJudge1,
		Round:     1,
		ScoreType: ScoreTechnique,
		Target:    TargetRed,
		Timestamp: now,
	})

	// Judge 2 votes round 2
	res2 := engine.Submit(JudgeInput{
		JudgeID:   "judge-2",
		Role:      RoleJudge2,
		Round:     2,
		ScoreType: ScoreTechnique,
		Target:    TargetRed,
		Timestamp: now.Add(50 * time.Millisecond),
	})

	if res2 == nil || res2.Accepted {
		t.Fatalf("expected not accepted for mismatched rounds, got: %+v", res2)
	}
}

func TestConsensusEngine_ConcurrentSubmissions(t *testing.T) {
	engine := NewEngine()
	var wg sync.WaitGroup

	// Concurrently submit votes from 3 judges
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(judgeNum int) {
			defer wg.Done()
			engine.Submit(JudgeInput{
				JudgeID:   "judge-concurrent",
				Role:      JudgeRole("judge_" + string(rune('0'+judgeNum))),
				Round:     1,
				ScoreType: ScoreTechnique,
				Target:    TargetRed,
				Timestamp: time.Now(),
			})
		}(i)
	}

	wg.Wait()
}
