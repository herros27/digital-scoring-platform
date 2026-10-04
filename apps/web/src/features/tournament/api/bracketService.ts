import { apiClient } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'

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

export interface GenerateBracketPayload {
  weight_class: string
  gender: 'male' | 'female' | 'putra' | 'putri'
  participant_ids?: string[]
}

export const bracketService = {
  /**
   * GET /api/v1/competitions/{comp_id}/brackets?weight_class=...&gender=...
   */
  async getBracketTree(compId: string, weightClass: string, gender: string): Promise<any> {
    const res = await apiClient.get<any, { success: boolean; data: any }>(
      `${ENDPOINTS.BRACKETS.TREE(compId)}?weight_class=${encodeURIComponent(weightClass)}&gender=${encodeURIComponent(gender)}`
    )
    return res.data
  },

  /**
   * POST /api/v1/competitions/{comp_id}/brackets/generate
   */
  async generateBracket(compId: string, payload: GenerateBracketPayload): Promise<BracketSlotDTO[]> {
    const res = await apiClient.post<any, { success: boolean; data: BracketSlotDTO[] }>(
      ENDPOINTS.BRACKETS.GENERATE(compId),
      payload
    )
    return res.data
  },

  /**
   * GET /api/v1/competitions/{comp_id}/participants
   */
  async getParticipants(compId: string): Promise<ParticipantDTO[]> {
    const res = await apiClient.get<any, { success: boolean; data: ParticipantDTO[] }>(
      ENDPOINTS.PARTICIPANTS.LIST(compId)
    )
    return res.data || []
  },
}
