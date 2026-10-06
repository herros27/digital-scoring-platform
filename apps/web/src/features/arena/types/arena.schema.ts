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

/** Satu baris GET /live/arenas: gelanggang yang sedang punya pertandingan aktif. */
export interface LiveArena {
  arena_id: string
  arena_name: string
  match_id: string
  state: string
  round: number
  red_athlete?: string
  blue_athlete?: string
  red_total: number
  blue_total: number
}

/**
 * Respons GET /live/scoreboard/{arena_id}. Backend saat ini hanya mengembalikan
 * `{ status: 'no active match' }`; field lain terisi bila ada pertandingan aktif.
 */
export interface LiveScoreboardSnapshot {
  status?: string
  arena_name?: string
  competition_name?: string
  /** Kelas partai, mis. "Kelas A Putri" */
  class_label?: string
  red_team?: string
  blue_team?: string
  /** Nomor urut partai pada antrean gelanggang */
  match_number?: number
  match_id?: string
  ruleset_name?: string
  round?: number
  red_athlete?: string
  blue_athlete?: string
}
