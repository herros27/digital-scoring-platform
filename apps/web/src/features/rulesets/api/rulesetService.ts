import { http } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'
import type { RulesetDTO, RulesetPayload } from '../types'

export const rulesetService = {
  /** GET /api/v1/rulesets */
  async list(): Promise<RulesetDTO[]> {
    const res = await http.get<RulesetDTO[]>(ENDPOINTS.RULESETS.LIST)
    return res.data || []
  },

  /** POST /api/v1/rulesets */
  async create(payload: RulesetPayload): Promise<RulesetDTO> {
    const res = await http.post<RulesetDTO>(ENDPOINTS.RULESETS.CREATE, payload)
    return res.data
  },

  /** PUT /api/v1/rulesets/{id} */
  async update(id: string, payload: RulesetPayload): Promise<RulesetDTO> {
    const res = await http.put<RulesetDTO>(ENDPOINTS.RULESETS.UPDATE(id), payload)
    return res.data
  },

  /** DELETE /api/v1/rulesets/{id} */
  async remove(id: string): Promise<void> {
    await http.delete(ENDPOINTS.RULESETS.DELETE(id))
  },
}
