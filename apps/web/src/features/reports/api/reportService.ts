import { apiClient } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'
import type { CompetitionSummary, MedalStanding, CompetitionExport } from '../types/report.schema'

export const reportService = {
  /**
   * 1. GET /api/v1/competitions/{comp_id}/report/summary
   */
  async getSummary(compId: string): Promise<CompetitionSummary> {
    if (!compId) throw new Error('Competition ID is required')
    const res = await apiClient.get<any, { success: boolean; data: CompetitionSummary }>(
      ENDPOINTS.REPORTS.SUMMARY(compId)
    )
    return res.data
  },

  /**
   * 2. GET /api/v1/competitions/{comp_id}/report/medals
   */
  async getMedalStandings(compId: string): Promise<MedalStanding[]> {
    if (!compId) return []
    const res = await apiClient.get<any, { success: boolean; data: MedalStanding[] }>(
      ENDPOINTS.REPORTS.MEDALS(compId)
    )
    return res.data || []
  },

  /**
   * 3. GET /api/v1/competitions/{comp_id}/report/export
   */
  async getExport(compId: string): Promise<CompetitionExport> {
    if (!compId) throw new Error('Competition ID is required')
    const res = await apiClient.get<any, { success: boolean; data: CompetitionExport }>(
      ENDPOINTS.REPORTS.EXPORT(compId)
    )
    return res.data
  },
}
