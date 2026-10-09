import { http } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'
import type {
  RulesetAssignment,
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
   * DELETE /api/v1/competitions/{comp_id}/categories — reset seluruh kategori (ganti standar).
   * Ditolak 409 bila masih ada peserta terdaftar pada kategori.
   */
  async resetCategories(competitionId: string): Promise<{ deleted: number }> {
    const res = await http.delete<{ deleted: number }>(ENDPOINTS.CATEGORIES.RESET(competitionId))
    return res.data
  },

  /**
   * PUT /api/v1/competitions/{comp_id}/categories/{id}/bracket-lock
   * Kunci/buka draw bagan sebuah kategori (disimpan di server dan ditegakkan pada brackets/generate).
   */
  async setBracketLock(competitionId: string, id: string, locked: boolean): Promise<Category> {
    const res = await http.put<Category>(ENDPOINTS.CATEGORIES.BRACKET_LOCK(competitionId, id), {
      locked,
    })
    return res.data
  },

  /**
   * PUT /api/v1/competitions/{comp_id}/categories/ruleset
   * Tetapkan ruleset khusus (durasi/jumlah babak) per golongan umur atau kategori.
   * ruleset_id null = ikut ruleset kejuaraan. Ditolak 409 bila sudah ada partai berjalan.
   */
  async assignRulesets(
    competitionId: string,
    assignments: RulesetAssignment[],
  ): Promise<{ updated: number }> {
    const res = await http.put<{ updated: number }>(
      ENDPOINTS.CATEGORIES.ASSIGN_RULESETS(competitionId),
      { assignments },
    )
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
