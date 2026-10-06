import { http } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'
import type {
  Category,
  CreateCategoryFormValues,
  StandardClassPreset,
} from '../types/category.schema'
import { IPSI_STANDARD_PRESETS } from '../types/category.schema'

export const categoryService = {
  /**
   * 1. GET /api/v1/competitions/{comp_id}/categories
   */
  async getCategories(competitionId: string): Promise<Category[]> {
    if (!competitionId) return []
    const url = ENDPOINTS.CATEGORIES.LIST(competitionId)
    const res = await http.get<Category[]>(url)
    return res.data || []
  },

  /**
   * 2. POST /api/v1/competitions/{comp_id}/categories
   */
  async createCategory(
    competitionId: string,
    payload: CreateCategoryFormValues,
  ): Promise<Category> {
    const url = ENDPOINTS.CATEGORIES.CREATE(competitionId)
    const res = await http.post<Category>(url, payload)
    return res.data
  },

  /**
   * 3. GET /api/v1/competitions/{comp_id}/categories/{id}
   */
  async getCategory(competitionId: string, id: string): Promise<Category> {
    const url = ENDPOINTS.CATEGORIES.DETAIL(competitionId, id)
    const res = await http.get<Category>(url)
    return res.data
  },

  /**
   * 4. Batch seed standard IPSI weight class presets
   * Convenient helper for tournament admins to quickly populate standard classes
   */
  async seedPresetCategories(
    competitionId: string,
    customPresets?: StandardClassPreset[],
  ): Promise<{ created: number; skipped: number }> {
    const presets = customPresets || IPSI_STANDARD_PRESETS
    const existing = await this.getCategories(competitionId)
    const existingNames = new Set(existing.map((c) => c.name.toLowerCase().trim()))

    let created = 0
    let skipped = 0

    for (const preset of presets) {
      if (existingNames.has(preset.categoryName.toLowerCase().trim())) {
        skipped++
        continue
      }

      try {
        await this.createCategory(competitionId, {
          name: preset.categoryName,
          gender: preset.gender,
          weight_class: preset.code,
        })
        created++
      } catch (err) {
        console.warn(`Failed to seed preset ${preset.categoryName}:`, err)
        skipped++
      }
    }

    return { created, skipped }
  },
}
