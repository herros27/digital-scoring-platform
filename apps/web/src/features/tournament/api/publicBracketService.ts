import { http } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'

/** GET /api/v1/public/competitions — kejuaraan berstatus aktif. */
export interface PublicCompetition {
  id: string
  name: string
  location: string
  /** YYYY-MM-DD */
  start_date: string
  end_date: string
}

/** Satu bagan (satu kategori) pada daftar bagan publik. */
export interface PublicBracketSummary {
  category_id: string
  name: string
  gender: string
  age_group: string
  weight_class: string
  participants: number
  rounds: number
  /** Babak final sudah punya pemenang. */
  completed: boolean
}

export interface PublicAthlete {
  name: string
  team: string
}

/** Slot bagan publik: hanya nama & kontingen; tidak ada ID peserta, NIK, atau tanggal lahir. */
export interface PublicBracketSlot {
  id: string
  round: number
  slot_index: number
  is_bye: boolean
  next_slot_id: string | null
  is_winner: boolean
  participant: PublicAthlete | null
}

export interface PublicBracketDetail {
  competition: PublicCompetition
  category: PublicBracketSummary
  slots: PublicBracketSlot[]
}

export const publicBracketService = {
  async getCompetitions(): Promise<PublicCompetition[]> {
    const res = await http.get<PublicCompetition[]>(ENDPOINTS.PUBLIC.COMPETITIONS)
    return res.data || []
  },

  async getBrackets(compId: string): Promise<PublicBracketSummary[]> {
    const res = await http.get<PublicBracketSummary[]>(ENDPOINTS.PUBLIC.BRACKETS(compId))
    return res.data || []
  },

  async getBracket(compId: string, categoryId: string): Promise<PublicBracketDetail> {
    const res = await http.get<PublicBracketDetail>(ENDPOINTS.PUBLIC.BRACKET(compId, categoryId))
    return res.data
  },
}
