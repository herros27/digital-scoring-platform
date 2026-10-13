export type MatchStatus =
  'idle' | 'ready' | 'running' | 'paused' | 'break' | 'finished' | 'cancelled'

export interface Match {
  id: string
  competition_id: string
  arena_id?: string
  bracket_slot_id?: string
  athlete_red_id: string
  athlete_blue_id: string
  current_round: number
  status: MatchStatus
  winner_id?: string
  created_at: string
  started_at?: string | null
  finished_at?: string | null
  // UI Display enriched fields
  partai_number?: number
  arena_name?: string
  red_athlete_name?: string
  red_athlete_club?: string
  blue_athlete_name?: string
  blue_athlete_club?: string
  weight_class?: string
}
