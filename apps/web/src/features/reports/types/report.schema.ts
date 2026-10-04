import { z } from 'zod'

export const competitionSummarySchema = z.object({
  competition_id: z.string(),
  competition_name: z.string(),
  total_arenas: z.number().int().nonnegative(),
  total_categories: z.number().int().nonnegative(),
  total_athletes: z.number().int().nonnegative(),
  total_matches: z.number().int().nonnegative(),
  matches_by_status: z.record(z.string(), z.number().int().nonnegative()).default({}),
})

export const medalStandingSchema = z.object({
  team: z.string(),
  gold: z.number().int().nonnegative().default(0),
  silver: z.number().int().nonnegative().default(0),
  bronze: z.number().int().nonnegative().default(0),
  total: z.number().int().nonnegative().default(0),
})

export const competitionExportSchema = z.object({
  competition: z.any(),
  summary: competitionSummarySchema,
  medals: z.array(medalStandingSchema),
  categories: z.array(z.any()),
  matches: z.array(z.any()),
})

export type CompetitionSummary = z.infer<typeof competitionSummarySchema>
export type MedalStanding = z.infer<typeof medalStandingSchema>
export type CompetitionExport = z.infer<typeof competitionExportSchema>
