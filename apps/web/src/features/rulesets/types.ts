/** Bentuk ruleset dari API (`/api/v1/rulesets`). Bagian opsional kosong = bawaan server. */
export interface RulesetTechnique {
  type: string
  label: string
  points: number
}

export interface RulesetPenalty {
  type: string
  label: string
  points: number
  action?: 'disqualify'
}

export interface RulesetOutcome {
  type: string
  label: string
  phase: 'before_start' | 'in_match' | 'any'
}

export interface RulesetScoring {
  technique_points: number
  knockdown_points: number
  fall_points: number
  max_penalties: number
  penalty_threshold: number
  round_duration: number
  break_duration: number
  total_rounds: number
  techniques?: RulesetTechnique[]
  penalties?: RulesetPenalty[]
  consensus?: { judges: number; majority: number; window_ms: number }
  tiebreak?: { steps?: string[]; technique_priority?: string[]; manual_methods?: string[] }
  outcomes?: RulesetOutcome[]
}

export interface RulesetDTO {
  id: string
  name: string
  version: string
  scoring: RulesetScoring
  status: 'draft' | 'active' | 'archived' | string
  created_at?: string
}

export interface RulesetPayload {
  name: string
  version: string
  status: string
  scoring: RulesetScoring
}
