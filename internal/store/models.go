package store

import "github.com/herros27/digital-scoring/internal/domain"

// Type aliases to domain models.
// Allows repository implementations to use domain types directly while keeping
// compatibility across the store package.
type Competition = domain.Competition
type Arena = domain.Arena
type Participant = domain.Participant
type Match = domain.Match
type MatchEvent = domain.MatchEvent
type MatchScore = domain.MatchScore
type MatchScoreDetail = domain.MatchScoreDetail
type BracketSlot = domain.BracketSlot
type JudgePin = domain.JudgePin
type Athlete = domain.Athlete
type Category = domain.Category
type Ruleset = domain.Ruleset
type Official = domain.Official
type Contingent = domain.Contingent
