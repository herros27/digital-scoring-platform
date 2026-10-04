package rules

type ScoringRule struct {
	TechniquePoints  int `json:"technique_points"`
	KnockdownPoints  int `json:"knockdown_points"`
	FallPoints       int `json:"fall_points"`
	MaxPenalties     int `json:"max_penalties"`
	PenaltyThreshold int `json:"penalty_threshold"` // tiap N pelanggaran = 1 poin lawan
	RoundDuration    int `json:"round_duration"`    // detik
	BreakDuration    int `json:"break_duration"`    // detik
	TotalRounds      int `json:"total_rounds"`
}

type Ruleset struct {
	Name    string      `json:"name"`
	Version string      `json:"version"`
	Scoring ScoringRule `json:"scoring"`
}
