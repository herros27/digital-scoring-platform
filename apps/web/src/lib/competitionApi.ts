import { api } from './api'
import type {
  Competition,
  CreateCompetitionPayload,
  UpdateCompetitionPayload,
  CompetitionListQuery,
  CompetitionListResult,
} from '@/types/competition'

export const competitionApi = {
  /**
   * 1. GET /api/v1/competitions
   * List all competitions with pagination and status filter.
   */
  async getCompetitions(query?: CompetitionListQuery): Promise<CompetitionListResult> {
    const params = new URLSearchParams()
    if (query?.page) params.append('page', String(query.page))
    if (query?.limit) params.append('limit', String(query.limit))
    if (query?.status) params.append('status', query.status)

    const qs = params.toString() ? `?${params.toString()}` : ''
    const res = await api.getPaged<Competition>(`/api/v1/competitions${qs}`)

    return {
      data: res.data || [],
      pagination: res.pagination || {
        page: query?.page || 1,
        limit: query?.limit || 20,
        total: res.data ? res.data.length : 0,
      },
    }
  },

  /**
   * 2. POST /api/v1/competitions
   * Create a new competition (Default status: draft).
   */
  async createCompetition(payload: CreateCompetitionPayload): Promise<Competition> {
    return api.post<Competition>('/api/v1/competitions', payload)
  },

  /**
   * 3. GET /api/v1/competitions/{id}
   * Get competition details by ID.
   */
  async getCompetition(id: string): Promise<Competition> {
    return api.get<Competition>(`/api/v1/competitions/${id}`)
  },

  /**
   * 4. PUT /api/v1/competitions/{id}
   * Update competition data.
   */
  async updateCompetition(id: string, payload: UpdateCompetitionPayload): Promise<Competition> {
    return api.put<Competition>(`/api/v1/competitions/${id}`, payload)
  },

  /**
   * 5. PUT /api/v1/competitions/{id}/activate (also supports POST)
   * Activate competition status.
   */
  async activateCompetition(id: string): Promise<{ status: string }> {
    return api.put<{ status: string }>(`/api/v1/competitions/${id}/activate`)
  },

  /**
   * 6. POST /api/v1/competitions/{id}/pause
   * Pause competition.
   */
  async pauseCompetition(id: string): Promise<{ status: string }> {
    return api.post<{ status: string }>(`/api/v1/competitions/${id}/pause`)
  },

  /**
   * 7. POST /api/v1/competitions/{id}/finish
   * Mark competition as completed.
   */
  async finishCompetition(id: string): Promise<{ status: string }> {
    return api.post<{ status: string }>(`/api/v1/competitions/${id}/finish`)
  },

  /**
   * 8. POST /api/v1/competitions/{id}/cancel
   * Cancel competition.
   */
  async cancelCompetition(id: string): Promise<{ status: string }> {
    return api.post<{ status: string }>(`/api/v1/competitions/${id}/cancel`)
  },
}
