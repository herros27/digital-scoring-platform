package consensus

import "time"

type JudgeRole string

const (
	RoleJudge   JudgeRole = "judge"
	RoleReferee JudgeRole = "referee"

	// Backward-compatible aliases
	RoleJudge1 JudgeRole = "judge_1"
	RoleJudge2 JudgeRole = "judge_2"
	RoleJudge3 JudgeRole = "judge_3"
)

type ScoreType string

const (
	ScoreTechnique ScoreType = "technique"
	ScoreKnockdown ScoreType = "knockdown"
	ScoreFall      ScoreType = "fall"
	ScorePenalty   ScoreType = "penalty"
)

type Target string

const (
	TargetRed  Target = "red"
	TargetBlue Target = "blue"
)

type JudgeInput struct {
	JudgeID    string
	Role       JudgeRole
	SeatNumber int
	MatchID    string
	Round      int
	ScoreType  ScoreType
	Target     Target
	Timestamp  time.Time
}
