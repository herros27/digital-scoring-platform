import { http } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'
import type {
  Competition,
  CreateCompetitionFormValues,
  UpdateCompetitionFormValues,
  CompetitionFilterValues,
  CompetitionStats,
} from '../types/competition.schema'
import type { CompetitionListResult } from '@/types/competition'
import type { WsTechniqueKind } from '@/composables/useWebSocket'

export const competitionService = {
  /**
   * GET /api/v1/competitions/{comp_id}/techniques: teknik (tombol skor juri) menurut ruleset kejuaraan.
   */
  async getTechniques(compId: string): Promise<WsTechniqueKind[]> {
    const res = await http.get<WsTechniqueKind[]>(ENDPOINTS.TECHNIQUES(compId))
    return res.data || []
  },

  /**
   * 1. GET /api/v1/competitions
   */
  async getCompetitions(params?: CompetitionFilterValues): Promise<CompetitionListResult> {
    const query = new URLSearchParams()
    if (params?.page) query.append('page', String(params.page))
    if (params?.limit) query.append('limit', String(params.limit))
    if (params?.status && params.status !== 'all') query.append('status', params.status)
    if (params?.search?.trim()) query.append('q', params.search.trim())

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
   * GET /api/v1/competitions/stats: jumlah per status di seluruh data, tidak bergantung filter/halaman.
   */
  async getStats(): Promise<CompetitionStats> {
    const res = await http.get<CompetitionStats>(ENDPOINTS.COMPETITIONS.STATS)
    return res.data
  },

  /**
   * 2. POST /api/v1/competitions
   */
  async createCompetition(payload: CreateCompetitionFormValues): Promise<Competition> {
    const res = await http.post<Competition>(ENDPOINTS.COMPETITIONS.CREATE, payload)
    return res.data
  },

  /**
   * 3. GET /api/v1/competitions/{id}
   */
  async getCompetition(id: string): Promise<Competition> {
    const res = await http.get<Competition>(ENDPOINTS.COMPETITIONS.DETAIL(id))
    return res.data
  },

  /**
   * 4. PUT /api/v1/competitions/{id}
   */
  async updateCompetition(id: string, payload: UpdateCompetitionFormValues): Promise<Competition> {
    const res = await http.put<Competition>(ENDPOINTS.COMPETITIONS.UPDATE(id), payload)
    return res.data
  },

  /**
   * 5. PUT /api/v1/competitions/{id}/activate
   */
  async activateCompetition(id: string): Promise<{ status: string }> {
    const res = await http.put<{ status: string }>(ENDPOINTS.COMPETITIONS.ACTIVATE(id))
    return res.data
  },

  /**
   * 6. POST /api/v1/competitions/{id}/pause
   */
  async pauseCompetition(id: string): Promise<{ status: string }> {
    const res = await http.post<{ status: string }>(ENDPOINTS.COMPETITIONS.PAUSE(id))
    return res.data
  },

  /**
   * 7. POST /api/v1/competitions/{id}/finish
   */
  async finishCompetition(id: string): Promise<{ status: string }> {
    const res = await http.post<{ status: string }>(ENDPOINTS.COMPETITIONS.FINISH(id))
    return res.data
  },

  /**
   * 8. POST /api/v1/competitions/{id}/cancel
   */
  async cancelCompetition(id: string): Promise<{ status: string }> {
    const res = await http.post<{ status: string }>(ENDPOINTS.COMPETITIONS.CANCEL(id))
    return res.data
  },
}
