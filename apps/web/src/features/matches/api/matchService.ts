import { http } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'

export interface MatchDTO {
  id: string
  competition_id: string
  arena_id: string
  bracket_slot_id?: string
  athlete_red_id: string
  athlete_blue_id: string
  current_round: number
  status: 'idle' | 'ready' | 'running' | 'paused' | 'break' | 'finished' | 'cancelled'
  winner_id?: string
  created_at: string
}

/** Skor satu ronde (domain.MatchScore). */
export interface RoundScoreDTO {
  id: string
  match_id: string
  round: number
  athlete_red_score: number
  athlete_blue_score: number
  red_penalties: number
  blue_penalties: number
  updated_at: string
}

export interface JudgeScoreSummaryDTO {
  judge_id: string
  role: string
  seat_number: number
  connected: boolean
}

/** Respons GET /competitions/{comp_id}/matches/{id}/scores (domain.MatchScoreDetail). */
export interface MatchScoresDTO {
  match_id: string
  total_red: number
  total_blue: number
  rounds: RoundScoreDTO[]
  judges?: JudgeScoreSummaryDTO[]
  consensus?: { red_score: number; blue_score: number }
}

/** Respons POST .../matches/{id}/scores. `result` hanya ada bila status 'accepted'. */
export interface SubmitScoreResult {
  status: 'accepted' | 'duplicate_ignored'
  duplicate?: boolean
  result?: unknown
}

export const matchService = {
  /**
   * GET /api/v1/competitions/{comp_id}/matches
   */
  async getMatches(compId: string, status?: string): Promise<MatchDTO[]> {
    const qs = status ? `?status=${status}` : ''
    const res = await http.get<MatchDTO[]>(
      `${ENDPOINTS.MATCHES.LIST(compId)}${qs}`
    )
    return res.data || []
  },

  /**
   * GET /api/v1/competitions/{comp_id}/matches/{id}
   */
  async getMatch(compId: string, id: string): Promise<MatchDTO> {
    const res = await http.get<MatchDTO>(
      ENDPOINTS.MATCHES.DETAIL(compId, id)
    )
    return res.data
  },

  /**
   * POST /api/v1/competitions/{comp_id}/matches
   */
  async createMatch(
    compId: string,
    payload: {
      arena_id?: string
      bracket_slot_id?: string
      athlete_red_id: string
      athlete_blue_id: string
      current_round?: number
    }
  ): Promise<MatchDTO> {
    const res = await http.post<MatchDTO>(
      ENDPOINTS.MATCHES.CREATE(compId),
      payload
    )
    return res.data
  },

  /**
   * POST /api/v1/competitions/{comp_id}/matches/{id}/load-arena
   */
  async loadArena(compId: string, matchId: string, arenaId: string, rulesetPath?: string): Promise<{ status: string }> {
    const res = await http.post<{ status: string }>(
      ENDPOINTS.MATCHES.LOAD_ARENA(compId, matchId),
      {
        arena_id: arenaId,
        ruleset_path: rulesetPath || '',
      }
    )
    return res.data
  },

  /**
   * POST /api/v1/competitions/{comp_id}/matches/{id}/start
   */
  async startMatch(compId: string, matchId: string): Promise<{ status: string }> {
    const res = await http.post<{ status: string }>(
      ENDPOINTS.MATCHES.START(compId, matchId)
    )
    return res.data
  },

  /**
   * POST /api/v1/competitions/{comp_id}/matches/{id}/pause
   */
  async pauseMatch(compId: string, matchId: string): Promise<{ status: string }> {
    const res = await http.post<{ status: string }>(
      ENDPOINTS.MATCHES.PAUSE(compId, matchId)
    )
    return res.data
  },

  /**
   * POST /api/v1/competitions/{comp_id}/matches/{id}/resume
   */
  async resumeMatch(compId: string, matchId: string): Promise<{ status: string }> {
    const res = await http.post<{ status: string }>(
      ENDPOINTS.MATCHES.RESUME(compId, matchId)
    )
    return res.data
  },

  /**
   * POST /api/v1/competitions/{comp_id}/matches/{id}/finish
   */
  async finishMatch(compId: string, matchId: string): Promise<{ status: string }> {
    const res = await http.post<{ status: string }>(
      ENDPOINTS.MATCHES.FINISH(compId, matchId)
    )
    return res.data
  },

  /**
   * POST /api/v1/competitions/{comp_id}/matches/{id}/cancel
   */
  async cancelMatch(compId: string, matchId: string): Promise<{ status: string }> {
    const res = await http.post<{ status: string }>(
      ENDPOINTS.MATCHES.CANCEL(compId, matchId)
    )
    return res.data
  },

  /**
   * GET /api/v1/competitions/{comp_id}/matches/{id}/scores
   */
  async getMatchScores(compId: string, matchId: string): Promise<MatchScoresDTO> {
    const res = await http.get<MatchScoresDTO>(
      ENDPOINTS.MATCHES.SCORES(compId, matchId)
    )
    return res.data
  },

  /**
   * POST /api/v1/competitions/{comp_id}/matches/{id}/scores (Submit with Idempotency Key)
   */
  async submitScore(
    compId: string,
    matchId: string,
    payload: { judge_id: string; score_type: string; target: string; event_id?: string },
    idempotencyKey?: string
  ): Promise<SubmitScoreResult> {
    const res = await http.post<SubmitScoreResult>(
      ENDPOINTS.MATCHES.SUBMIT_SCORE(compId, matchId),
      payload,
      {
        headers: idempotencyKey ? { 'Idempotency-Key': idempotencyKey } : undefined,
      }
    )
    return res.data
  },
}
