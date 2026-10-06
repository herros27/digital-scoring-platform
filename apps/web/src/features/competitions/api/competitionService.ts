import { http } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'
import type {
  Competition,
  CreateCompetitionFormValues,
  UpdateCompetitionFormValues,
  CompetitionFilterValues,
} from '../types/competition.schema'
import type { CompetitionListResult } from '@/types/competition'

export const competitionService = {
  /**
   * 1. GET /api/v1/competitions
   */
  async getCompetitions(params?: CompetitionFilterValues): Promise<CompetitionListResult> {
    const query = new URLSearchParams()
    if (params?.page) query.append('page', String(params.page))
    if (params?.limit) query.append('limit', String(params.limit))
    if (params?.status && params.status !== 'all') query.append('status', params.status)

    const url = `${ENDPOINTS.COMPETITIONS.LIST}${query.toString() ? `?${query.toString()}` : ''}`
    const res = await http.get<Competition[]>(url)

    return {
      data: res.data || [],
      pagination: res.pagination || {
        page: params?.page || 1,
        limit: params?.limit || 20,
        total: res.data ? res.data.length : 0,
      },
    }
  },

  /**
   * 2. POST /api/v1/competitions
   */
  async createCompetition(payload: CreateCompetitionFormValues): Promise<Competition> {
    const res = await http.post<Competition>(
      ENDPOINTS.COMPETITIONS.CREATE,
      payload
    )
    return res.data
  },

  /**
   * 3. GET /api/v1/competitions/{id}
   */
  async getCompetition(id: string): Promise<Competition> {
    const res = await http.get<Competition>(
      ENDPOINTS.COMPETITIONS.DETAIL(id)
    )
    return res.data
  },

  /**
   * 4. PUT /api/v1/competitions/{id}
   */
  async updateCompetition(id: string, payload: UpdateCompetitionFormValues): Promise<Competition> {
    const res = await http.put<Competition>(
      ENDPOINTS.COMPETITIONS.UPDATE(id),
      payload
    )
    return res.data
  },

  /**
   * 5. PUT /api/v1/competitions/{id}/activate
   */
  async activateCompetition(id: string): Promise<{ status: string }> {
    const res = await http.put<{ status: string }>(
      ENDPOINTS.COMPETITIONS.ACTIVATE(id)
    )
    return res.data
  },

  /**
   * 6. POST /api/v1/competitions/{id}/pause
   */
  async pauseCompetition(id: string): Promise<{ status: string }> {
    const res = await http.post<{ status: string }>(
      ENDPOINTS.COMPETITIONS.PAUSE(id)
    )
    return res.data
  },

  /**
   * 7. POST /api/v1/competitions/{id}/finish
   */
  async finishCompetition(id: string): Promise<{ status: string }> {
    const res = await http.post<{ status: string }>(
      ENDPOINTS.COMPETITIONS.FINISH(id)
    )
    return res.data
  },

  /**
   * 8. POST /api/v1/competitions/{id}/cancel
   */
  async cancelCompetition(id: string): Promise<{ status: string }> {
    const res = await http.post<{ status: string }>(
      ENDPOINTS.COMPETITIONS.CANCEL(id)
    )
    return res.data
  },
}
