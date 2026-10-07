import { http } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'
import type {
  Category,
  CreateCategoryFormValues,
  StandardClassPreset,
} from '../types/category.schema'
import { IPSI_STANDARD_PRESETS, presetToPayload } from '../types/category.schema'

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
   * 4. Batch seed preset kelas resmi (IPSI / PERSILAT 2026).
   * Kategori yang namanya sudah ada dilewati; pembuatan dijalankan paralel terbatas
   * agar ratusan kelas tidak membanjiri server.
   */
  async seedPresetCategories(
    competitionId: string,
    customPresets?: StandardClassPreset[],
  ): Promise<{ created: number; skipped: number }> {
    const presets = customPresets || IPSI_STANDARD_PRESETS
    const existing = await this.getCategories(competitionId)
    const existingNames = new Set(existing.map((c) => c.name.toLowerCase().trim()))

    const todo = presets.filter((p) => !existingNames.has(p.categoryName.toLowerCase().trim()))
    let created = 0
    let failed = 0

    const CONCURRENCY = 6
    let cursor = 0
    const worker = async () => {
      while (cursor < todo.length) {
        const preset = todo[cursor++]
        try {
          await this.createCategory(competitionId, presetToPayload(preset))
          created++
        } catch (err) {
          console.warn(`Failed to seed preset ${preset.categoryName}:`, err)
          failed++
        }
      }
    }
    await Promise.all(Array.from({ length: Math.min(CONCURRENCY, todo.length) }, worker))

    return { created, skipped: presets.length - todo.length + failed }
  },
}
