import { apiClient } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'
import type { Arena } from '../types/arena.schema'

export const arenaService = {
  /**
   * 1. GET /api/v1/competitions/{comp_id}/arenas
   */
  async getArenas(compId: string): Promise<Arena[]> {
    if (!compId) return []
    const res = await apiClient.get<any, { success: boolean; data: Arena[] }>(
      ENDPOINTS.ARENAS.LIST(compId)
    )
    return res.data || []
  },

  /**
   * 2. POST /api/v1/competitions/{comp_id}/arenas
   */
  async createArena(compId: string, name: string): Promise<Arena> {
    const res = await apiClient.post<any, { success: boolean; data: Arena }>(
      ENDPOINTS.ARENAS.CREATE(compId),
      { name }
    )
    return res.data
  },

  /**
   * 3. GET /api/v1/competitions/{comp_id}/arenas/{id}
   */
  async getArena(compId: string, id: string): Promise<Arena> {
    const res = await apiClient.get<any, { success: boolean; data: Arena }>(
      ENDPOINTS.ARENAS.DETAIL(compId, id)
    )
    return res.data
  },

  /**
   * 4. PUT /api/v1/competitions/{comp_id}/arenas/{id}
   */
  async updateArena(compId: string, id: string, name: string): Promise<Arena> {
    const res = await apiClient.put<any, { success: boolean; data: Arena }>(
      ENDPOINTS.ARENAS.UPDATE(compId, id),
      { name }
    )
    return res.data
  },

  /**
   * 5. DELETE /api/v1/competitions/{comp_id}/arenas/{id}
   */
  async deleteArena(compId: string, id: string): Promise<void> {
    await apiClient.delete(ENDPOINTS.ARENAS.DELETE(compId, id))
  },

  /**
   * 6. Quick seed standard tournament arenas (Gelanggang A s/d D)
   */
  async seedStandardArenas(compId: string): Promise<{ created: number; skipped: number }> {
    const standardNames = [
      'Gelanggang A (Utama)',
      'Gelanggang B',
      'Gelanggang C',
      'Gelanggang D',
    ]

    const existing = await this.getArenas(compId)
    const existingNames = new Set(existing.map((a) => a.name.toLowerCase().trim()))

    let created = 0
    let skipped = 0

    for (const name of standardNames) {
      // Check if standard prefix exists
      const prefix = name.split(' ')[0].toLowerCase()
      const alreadyHas = Array.from(existingNames).some((en) => en.startsWith(prefix))
      if (alreadyHas) {
        skipped++
        continue
      }

      try {
        await this.createArena(compId, name)
        created++
      } catch (err) {
        console.warn(`Failed to seed arena ${name}:`, err)
        skipped++
      }
    }

    return { created, skipped }
  },

  /**
   * 7. GET /api/v1/live/scoreboard/{arena_id} (Public)
   */
  async getLiveScoreboard(arenaId: string): Promise<any> {
    const res = await apiClient.get<any, { success: boolean; data: any }>(
      ENDPOINTS.LIVE.SCOREBOARD(arenaId)
    )
    return res.data
  },
}
