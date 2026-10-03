package domain

import "time"

type Competition struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Location    string    `json:"location"`
	StartDate   string    `json:"start_date"`
	EndDate     string    `json:"end_date"`
	RulesetName string    `json:"ruleset_name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type Arena struct {
	ID            string    `json:"id"`
	CompetitionID string    `json:"competition_id"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"created_at"`
}

type Participant struct {
	ID            string    `json:"id"`
	CompetitionID string    `json:"competition_id"`
	Name          string    `json:"name"`
	Club          string    `json:"club"`
	WeightClass   string    `json:"weight_class"`
	Gender        string    `json:"gender"`
	CreatedAt     time.Time `json:"created_at"`
}

type Match struct {
	ID            string     `json:"id"`
	CompetitionID string     `json:"competition_id"`
	ArenaID       string     `json:"arena_id"`
	BracketSlotID string     `json:"bracket_slot_id"`
	AthleteRedID  string     `json:"athlete_red_id"`
	AthleteBlueID string     `json:"athlete_blue_id"`
	CurrentRound  int        `json:"current_round"`
	Status        string     `json:"status"`
	WinnerID      string     `json:"winner_id"`
	CreatedAt     time.Time  `json:"created_at"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
}

type MatchEvent struct {
	ID        string    `json:"id"`
	MatchID   string    `json:"match_id"`
	EventType string    `json:"event_type"`
	Payload   string    `json:"payload"` // raw JSON
	JudgeID   string    `json:"judge_id"`
	CreatedAt time.Time `json:"created_at"`
}

type MatchScore struct {
	ID               string    `json:"id"`
	MatchID          string    `json:"match_id"`
	Round            int       `json:"round"`
	AthleteRedScore  int       `json:"athlete_red_score"`
	AthleteBlueScore int       `json:"athlete_blue_score"`
	RedPenalties     int       `json:"red_penalties"`
	BluePenalties    int       `json:"blue_penalties"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type BracketSlot struct {
	ID            string  `json:"id"`
	CompetitionID string  `json:"competition_id"`
	WeightClass   string  `json:"weight_class"`
	Gender        string  `json:"gender"`
	Round         int     `json:"round"`
	SlotIndex     int     `json:"slot_index"`
	ParticipantID *string `json:"participant_id"`
	IsBye         bool    `json:"is_bye"`
	WinnerID      *string `json:"winner_id"`
	NextSlotID    *string `json:"next_slot_id"`
}

type JudgePin struct {
	ID            string    `json:"id"`
	CompetitionID string    `json:"competition_id"`
	ArenaID       string    `json:"arena_id"`
	Role          string    `json:"role"`
	PinHash       string    `json:"-"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type Athlete struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Gender    string    `json:"gender"`
	Team      string    `json:"team"`
	CreatedAt time.Time `json:"created_at"`
}

type Category struct {
	ID            string    `json:"id"`
	CompetitionID string    `json:"competition_id"`
	Name          string    `json:"name"`
	Gender        string    `json:"gender"`
	WeightClass   string    `json:"weight_class"`
	RulesetID     *string   `json:"ruleset_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type Ruleset struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Scoring   any       `json:"scoring"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
