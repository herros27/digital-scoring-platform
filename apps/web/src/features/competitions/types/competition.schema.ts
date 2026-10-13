import { z } from 'zod'

export const competitionStatusSchema = z.enum([
  'draft',
  'active',
  'paused',
  'completed',
  'cancelled',
])

export const competitionSchema = z.object({
  id: z.string().uuid(),
  name: z.string().min(2, 'Nama kejuaraan minimal 2 karakter').max(100),
  location: z.string(),
  start_date: z.string(),
  end_date: z.string(),
  ruleset_id: z.string().optional(),
  ruleset_name: z.string(),
  status: competitionStatusSchema,
  created_at: z.string(),
})

export const createCompetitionSchema = z
  .object({
    name: z
      .string()
      .min(2, 'Nama kejuaraan minimal 2 karakter')
      .max(100, 'Nama kejuaraan maksimal 100 karakter'),
    location: z.string().min(1, 'Lokasi / venue wajib diisi'),
    start_date: z.string().min(1, 'Tanggal mulai wajib dipilih'),
    end_date: z.string().min(1, 'Tanggal selesai wajib dipilih'),
    ruleset_name: z.string().min(1, 'Pilih ruleset pertandingan'),
    ruleset_id: z.string().optional(),
  })
  .refine((data) => !data.start_date || !data.end_date || data.start_date <= data.end_date, {
    message: 'Tanggal selesai tidak boleh sebelum tanggal mulai',
    path: ['end_date'],
  })

export const updateCompetitionSchema = z.object({
  name: z.string().min(2).max(100).optional(),
  location: z.string().optional(),
  start_date: z.string().optional(),
  end_date: z.string().optional(),
  ruleset_name: z.string().optional(),
  ruleset_id: z.string().optional(),
})

export const competitionFilterSchema = z.object({
  status: z.union([competitionStatusSchema, z.literal('all')]).optional(),
  search: z.string().optional(),
  page: z.number().int().positive().default(1),
  limit: z.number().int().positive().max(100).default(20),
})

export const competitionStatsSchema = z.object({
  total: z.number(),
  draft: z.number(),
  active: z.number(),
  paused: z.number(),
  completed: z.number(),
  cancelled: z.number(),
})
export type CompetitionStats = z.infer<typeof competitionStatsSchema>

export type CompetitionStatus = z.infer<typeof competitionStatusSchema>
export type Competition = z.infer<typeof competitionSchema>
export type CreateCompetitionFormValues = z.infer<typeof createCompetitionSchema>
export type UpdateCompetitionFormValues = z.infer<typeof updateCompetitionSchema>
export type CompetitionFilterValues = z.infer<typeof competitionFilterSchema>
