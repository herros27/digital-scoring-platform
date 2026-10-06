import { z } from 'zod'
import { competitionSchema } from '@/features/competitions/types/competition.schema'
import type { Category } from '@/features/categories/types/category.schema'
import type { Match } from '@/features/matches/types/match.schema'

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
  competition: competitionSchema,
  summary: competitionSummarySchema,
  medals: z.array(medalStandingSchema),
  categories: z.array(z.custom<Category>()),
  matches: z.array(z.custom<Match>()),
})

export type CompetitionSummary = z.infer<typeof competitionSummarySchema>
export type MedalStanding = z.infer<typeof medalStandingSchema>
export type CompetitionExport = z.infer<typeof competitionExportSchema>
