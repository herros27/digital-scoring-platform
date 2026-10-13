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
  category_id?: string
  created_at: string
}

/** Slot bagan sebagaimana dikembalikan GET /competitions/{comp_id}/brackets (domain.BracketSlot). */
export interface BracketSlotRecord {
  id: string
  competition_id: string
  weight_class: string
  gender: string
  round: number
  slot_index: number
  category_id?: string | null
  participant_id: string | null
  is_bye: boolean
  winner_id: string | null
  next_slot_id: string | null
}

export interface GenerateBracketPayload {
  /** Bagan terisolasi per kategori: kelas bobot, gender & mode bagan diambil dari kategori. */
  category_id?: string
  weight_class?: string
  gender?: Gender
  participant_ids?: string[]
}

export const bracketService = {
  /**
   * GET /api/v1/competitions/{comp_id}/brackets?weight_class=...&gender=...
   */
  async getBracketTreeByCategory(compId: string, categoryId: string): Promise<BracketSlotRecord[]> {
    const res = await http.get<BracketSlotRecord[]>(
      `${ENDPOINTS.BRACKETS.TREE(compId)}?category_id=${encodeURIComponent(categoryId)}`,
    )
    return res.data
  },

  async getBracketTree(
    compId: string,
    weightClass: string,
    gender: string,
  ): Promise<BracketSlotRecord[]> {
    const res = await http.get<BracketSlotRecord[]>(
      `${ENDPOINTS.BRACKETS.TREE(compId)}?weight_class=${encodeURIComponent(weightClass)}&gender=${encodeURIComponent(gender)}`,
    )
    return res.data
  },

  /**
   * POST /api/v1/competitions/{comp_id}/brackets/generate
   */
  async generateBracket(
    compId: string,
    payload: GenerateBracketPayload,
  ): Promise<BracketSlotRecord[]> {
    const res = await http.post<BracketSlotRecord[]>(ENDPOINTS.BRACKETS.GENERATE(compId), payload)
    return res.data
  },

  /**
   * DELETE /api/v1/competitions/{comp_id}/categories/{id}/bracket — hapus bagan kategori beserta partai idle-nya.
   * Ditolak 409 bila draw terkunci atau sudah ada partai yang dimulai.
   */
  async resetBracket(compId: string, categoryId: string): Promise<void> {
    await http.delete<unknown>(ENDPOINTS.BRACKETS.RESET(compId, categoryId))
  },

  /**
   * GET /api/v1/competitions/{comp_id}/participants
   */
  async getParticipants(compId: string): Promise<ParticipantDTO[]> {
    const res = await http.get<ParticipantDTO[]>(ENDPOINTS.PARTICIPANTS.LIST(compId))
    return res.data || []
  },
}
