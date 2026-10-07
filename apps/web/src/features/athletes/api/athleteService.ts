import { http } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'
import type {
  Athlete,
  Contingent,
  Participant,
  CreateAthleteFormValues,
  RegisterParticipantPayload,
  RegisterParticipantResult,
  VerifyParticipantResult,
} from '../types/athlete.schema'

export const athleteService = {
  /**
   * 1. GET /api/v1/athletes
   */
  async getAthletes(params?: {
    page?: number
    limit?: number
    gender?: string
  }): Promise<Athlete[]> {
    const query = new URLSearchParams()
    if (params?.page) query.append('page', String(params.page))
    if (params?.limit) query.append('limit', String(params.limit))
    if (params?.gender && params.gender !== 'all') query.append('gender', params.gender)

    const url = `${ENDPOINTS.ATHLETES.LIST}${query.toString() ? `?${query.toString()}` : ''}`
    const res = await http.get<Athlete[]>(url)
    return res.data || []
  },

  /**
   * 2. GET /api/v1/competitions/{comp_id}/athletes
   */
  async getAthletesByCompetition(compId: string): Promise<Athlete[]> {
    if (!compId) return []
    const url = ENDPOINTS.ATHLETES.BY_COMPETITION(compId)
    const res = await http.get<Athlete[]>(url)
    return res.data || []
  },

  /**
   * 3. POST /api/v1/athletes
   */
  async createAthlete(payload: CreateAthleteFormValues): Promise<Athlete> {
    const url = ENDPOINTS.ATHLETES.CREATE
    const res = await http.post<Athlete>(url, payload)
    return res.data
  },

  /**
   * 4. POST /api/v1/competitions/{comp_id}/athletes
   * Register existing or new athlete into tournament
   */
  async registerToCompetition(
    compId: string,
    payload: RegisterParticipantPayload,
  ): Promise<RegisterParticipantResult> {
    const url = ENDPOINTS.ATHLETES.REGISTER_TO_COMP(compId)
    const res = await http.post<RegisterParticipantResult>(url, payload)
    return res.data
  },

  /**
   * 5. GET /api/v1/competitions/{comp_id}/participants
   */
  async getParticipants(compId: string, status?: string): Promise<Participant[]> {
    if (!compId) return []
    const query = new URLSearchParams()
    if (status && status !== 'all') query.append('status', status)

    const url = `${ENDPOINTS.ATHLETES.PARTICIPANTS(compId)}${query.toString() ? `?${query.toString()}` : ''}`
    const res = await http.get<Participant[]>(url)
    return res.data || []
  },

  /**
   * 6. PUT /api/v1/competitions/{comp_id}/participants/{id}/verify
   */
  async verifyParticipant(
    compId: string,
    participantId: string,
    status: 'verified' | 'rejected',
    extra?: { actual_weight?: number; docs_verified?: Record<string, boolean> },
  ): Promise<VerifyParticipantResult> {
    const url = ENDPOINTS.ATHLETES.VERIFY_PARTICIPANT(compId, participantId)
    const res = await http.put<VerifyParticipantResult>(url, { status, ...extra })
    return res.data
  },

  /**
   * 7. GET /api/v1/contingents
   */
  async getContingents(): Promise<Contingent[]> {
    const url = ENDPOINTS.CONTINGENTS.LIST
    const res = await http.get<Contingent[]>(url)
    return res.data || []
  },

  /**
   * 8. GET /api/v1/contingents/{id}
   */
  async getContingent(id: string): Promise<Contingent> {
    const url = ENDPOINTS.CONTINGENTS.DETAIL(id)
    const res = await http.get<Contingent>(url)
    return res.data
  },
}
