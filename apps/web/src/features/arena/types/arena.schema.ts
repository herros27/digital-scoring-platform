import { z } from 'zod'

export const createArenaSchema = z.object({
  name: z.string().min(2, 'Nama gelanggang minimal 2 karakter').max(100, 'Maksimal 100 karakter'),
})

export const updateArenaSchema = z.object({
  name: z.string().min(2, 'Nama gelanggang minimal 2 karakter').max(100, 'Maksimal 100 karakter'),
})

export type CreateArenaFormValues = z.infer<typeof createArenaSchema>
export type UpdateArenaFormValues = z.infer<typeof updateArenaSchema>

export interface Arena {
  id: string
  competition_id: string
  name: string
  created_at: string
  // UI extended states
  current_match?: string
  status?: 'live' | 'ready' | 'idle'
  connected_judges?: number
}

/**
 * Respons GET /live/scoreboard/{arena_id}. Backend saat ini hanya mengembalikan
 * `{ status: 'no active match' }`; field lain terisi bila ada pertandingan aktif.
 */
export interface LiveScoreboardSnapshot {
  status?: string
  round?: number
  red_athlete?: string
  blue_athlete?: string
}
