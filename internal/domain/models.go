package domain

import "time"

type Competition struct {
	ID          string    `json:"id"`
	Name        string    `json:"name" validate:"required,min=2,max=100"`
	Location    string    `json:"location"`
	StartDate   string    `json:"start_date"`
	EndDate     string    `json:"end_date"`
	RulesetID   *string   `json:"ruleset_id,omitempty"`
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
	ID                 string `json:"id"`
	CompetitionID      string `json:"competition_id"`
	AthleteID          string `json:"athlete_id,omitempty"`
	ContingentID       string `json:"contingent_id,omitempty"`
	Name               string `json:"name,omitempty"`
	Club               string `json:"club,omitempty"`
	WeightClass        string `json:"weight_class"`
	Gender             string `json:"gender"`
	RegistrationStatus string `json:"registration_status"`
	CategoryID         string `json:"category_id,omitempty"`
	CategoryName       string `json:"category_name,omitempty"`
	AgeGroup           string `json:"age_group,omitempty"`
	PresetStandard     string `json:"-"`
	// AgeCheck hasil pemeriksaan umur vs golongan kategori (lihat AgeCheck); kosong bila sesuai.
	AgeCheck     string          `json:"age_check,omitempty"`
	BirthDate    string          `json:"birth_date,omitempty"`
	NIK          string          `json:"nik,omitempty"`
	ActualWeight *float64        `json:"actual_weight,omitempty"`
	DocsVerified map[string]bool `json:"docs_verified,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type Match struct {
	ID            string     `json:"id"`
	CompetitionID string     `json:"competition_id"`
	ArenaID       string     `json:"arena_id"`
	BracketSlotID string     `json:"bracket_slot_id"`
	AthleteRedID  string     `json:"athlete_red_id" validate:"required"`
	AthleteBlueID string     `json:"athlete_blue_id" validate:"required"`
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

type ConsensusSummary struct {
	RedScore  int `json:"red_score"`
	BlueScore int `json:"blue_score"`
}

type JudgeScoreSummary struct {
	JudgeID    string `json:"judge_id"`
	Role       string `json:"role"`
	SeatNumber int    `json:"seat_number"`
	Connected  bool   `json:"connected"`
	RedScore   int    `json:"red_score"`
	BlueScore  int    `json:"blue_score"`
}

type MatchScoreDetail struct {
	MatchID   string              `json:"match_id"`
	TotalRed  int                 `json:"total_red"`
	TotalBlue int                 `json:"total_blue"`
	Rounds    []MatchScore        `json:"rounds"`
	Judges    []JudgeScoreSummary `json:"judges,omitempty"`
	Consensus *ConsensusSummary   `json:"consensus,omitempty"`
}

type BracketSlot struct {
	ID            string  `json:"id"`
	CompetitionID string  `json:"competition_id"`
	WeightClass   string  `json:"weight_class"`
	Gender        string  `json:"gender"`
	Round         int     `json:"round"`
	SlotIndex     int     `json:"slot_index"`
	CategoryID    *string `json:"category_id,omitempty"`
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
	SeatNumber    int       `json:"seat_number,omitempty"`
	PinHash       string    `json:"-"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type Athlete struct {
	ID           string    `json:"id"`
	ContingentID string    `json:"contingent_id,omitempty"`
	Name         string    `json:"name" validate:"required,min=2,max=100"`
	Gender       string    `json:"gender" validate:"required,oneof=male female putra putri"`
	Team         string    `json:"team" validate:"required,min=1,max=100"`
	BirthDate    string    `json:"birth_date,omitempty"` // YYYY-MM-DD
	NIK          string    `json:"nik,omitempty" validate:"omitempty,max=50"`
	CreatedAt    time.Time `json:"created_at"`
}

type Contingent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	PicName   string    `json:"pic_name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
}

type Category struct {
	ID             string   `json:"id"`
	CompetitionID  string   `json:"competition_id"`
	Name           string   `json:"name"`
	Gender         string   `json:"gender"`
	WeightClass    string   `json:"weight_class"`
	RulesetID      *string  `json:"ruleset_id,omitempty"`
	AgeGroup       string   `json:"age_group"`
	PresetStandard string   `json:"preset_standard"`
	MinWeight      *float64 `json:"min_weight,omitempty"`
	MaxWeight      *float64 `json:"max_weight,omitempty"`
	BracketMode    string   `json:"bracket_mode"`
	// HomeArenaID gelanggang asal: semua partai kategori ini dimainkan di sana. Nil sampai bagan pertama dibuat.
	HomeArenaID *string `json:"home_arena_id,omitempty"`
	// BracketLockedAt bukan nil bila draw bagan kategori ini dikunci panitia: bagan tidak boleh diacak ulang.
	BracketLockedAt *time.Time `json:"bracket_locked_at,omitempty"`
	BracketLocked   bool       `json:"bracket_locked"`
	CreatedAt       time.Time  `json:"created_at"`
}

// CategoryRulesetAssignment menetapkan ruleset khusus untuk satu kategori (CategoryID) atau
// seluruh kategori satu golongan umur (AgeGroup). RulesetID nil = ikut ruleset kejuaraan.
type CategoryRulesetAssignment struct {
	AgeGroup   string  `json:"age_group,omitempty"`
	CategoryID string  `json:"category_id,omitempty"`
	RulesetID  *string `json:"ruleset_id"`
}

type Ruleset struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Scoring   any       `json:"scoring"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Official struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type CompetitionSummary struct {
	CompetitionID   string         `json:"competition_id"`
	CompetitionName string         `json:"competition_name"`
	TotalArenas     int            `json:"total_arenas"`
	TotalCategories int            `json:"total_categories"`
	TotalAthletes   int            `json:"total_athletes"`
	TotalMatches    int            `json:"total_matches"`
	MatchesByStatus map[string]int `json:"matches_by_status"`
}

type MedalStanding struct {
	Team   string `json:"team"`
	Gold   int    `json:"gold"`
	Silver int    `json:"silver"`
	Bronze int    `json:"bronze"`
	Total  int    `json:"total"`
}

type CompetitionExport struct {
	Competition *Competition        `json:"competition"`
	Summary     *CompetitionSummary `json:"summary"`
	Medals      []MedalStanding     `json:"medals"`
	Categories  []Category          `json:"categories"`
	Matches     []Match             `json:"matches"`
}
