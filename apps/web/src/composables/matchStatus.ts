export type MatchStatus = 'idle' | 'ready' | 'running' | 'paused' | 'break' | 'finished'

/**
 * Petakan `state` mesin pertandingan dari server ke status tampilan.
 *
 * - `round_N` → `running`, atau `paused` bila `timerRunning === false` (jeda tidak mengubah `state`).
 * - `break_N` → `break`.
 * - `finished` dan `cancelled` → `finished` (keduanya terminal; tombol mulai tidak boleh aktif).
 * - State yang tidak dikenal → `undefined` (status lokal dipertahankan).
 */
export function matchStatusFromState(
  state: string | undefined,
  timerRunning?: boolean,
): MatchStatus | undefined {
  if (!state) return undefined
  if (state === 'idle') return 'idle'
  if (state === 'ready') return 'ready'
  if (state.startsWith('round')) return timerRunning === false ? 'paused' : 'running'
  if (state.startsWith('break')) return 'break'
  if (state === 'finished' || state === 'cancelled') return 'finished'
  return undefined
}
