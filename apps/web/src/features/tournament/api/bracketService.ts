import { http } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'
import type { Gender } from '@/features/athletes/types/athlete.schema'

export interface ParticipantDTO {
  id: string
  competition_id: string
  athlete_id?: string
  contingent_id?: string
  name: string
  club?: string
  weight_class: string
  gender: string
  registration_status: string
  created_at: string
}

export interface BracketSlotDTO {
  id: string
  round: number
  slot_number: number
  match_id?: string
  athlete_red_id?: string
  athlete_blue_id?: string
  winner_id?: string
}

/** Slot bagan sebagaimana dikembalikan GET /competitions/{comp_id}/brackets (domain.BracketSlot). */
export interface BracketSlotRecord {
  id: string
  competition_id: string
  weight_class: string
  gender: string
  round: number
  slot_index: number
  participant_id: string | null
  is_bye: boolean
  winner_id: string | null
  next_slot_id: string | null
}

export interface GenerateBracketPayload {
  weight_class: string
  gender: Gender
  participant_ids?: string[]
}

export const bracketService = {
  /**
   * GET /api/v1/competitions/{comp_id}/brackets?weight_class=...&gender=...
   */
  async getBracketTree(compId: string, weightClass: string, gender: string): Promise<BracketSlotRecord[]> {
    const res = await http.get<BracketSlotRecord[]>(
      `${ENDPOINTS.BRACKETS.TREE(compId)}?weight_class=${encodeURIComponent(weightClass)}&gender=${encodeURIComponent(gender)}`
    )
    return res.data
  },

  /**
   * POST /api/v1/competitions/{comp_id}/brackets/generate
   */
  async generateBracket(compId: string, payload: GenerateBracketPayload): Promise<BracketSlotDTO[]> {
    const res = await http.post<BracketSlotDTO[]>(
      ENDPOINTS.BRACKETS.GENERATE(compId),
      payload
    )
    return res.data
  },

  /**
   * GET /api/v1/competitions/{comp_id}/participants
   */
  async getParticipants(compId: string): Promise<ParticipantDTO[]> {
    const res = await http.get<ParticipantDTO[]>(
      ENDPOINTS.PARTICIPANTS.LIST(compId)
    )
    return res.data || []
  },
}
