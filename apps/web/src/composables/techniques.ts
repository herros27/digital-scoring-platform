import type { WsTechniqueKind } from '@/composables/useWebSocket'

import type { WsOutcomeKind } from '@/composables/useWebSocket'

/** Hasil khusus bawaan sebelum snapshot server tiba (sama dengan bawaan ruleset). */
export const DEFAULT_OUTCOMES: WsOutcomeKind[] = [
  { type: 'ko', label: 'Menang Mutlak (KO)', phase: 'in_match' },
  { type: 'tko', label: 'Menang Teknik (TKO)', phase: 'in_match' },
  { type: 'walkover', label: 'Menang WO', phase: 'before_start' },
  { type: 'withdrawal', label: 'Undur Diri', phase: 'any' },
]

/**
 * Daftar teknik sementara sebelum snapshot server tiba (sama dengan turunan ruleset lama:
 * Pukulan 1, Tendangan 2, Jatuhan 3). Setelah tersinkron, daftar dari ruleset yang dipakai.
 */
export const DEFAULT_TECHNIQUES: WsTechniqueKind[] = [
  { type: 'technique', label: 'Pukulan', points: 1 },
  { type: 'knockdown', label: 'Tendangan', points: 2 },
  { type: 'fall', label: 'Jatuhan', points: 3 },
]
