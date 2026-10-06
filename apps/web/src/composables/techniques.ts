import type { WsTechniqueKind } from '@/composables/useWebSocket'

/**
 * Daftar teknik sementara sebelum snapshot server tiba (sama dengan turunan ruleset lama:
 * Pukulan 1, Tendangan 2, Jatuhan 3). Setelah tersinkron, daftar dari ruleset yang dipakai.
 */
export const DEFAULT_TECHNIQUES: WsTechniqueKind[] = [
  { type: 'technique', label: 'Pukulan', points: 1 },
  { type: 'knockdown', label: 'Tendangan', points: 2 },
  { type: 'fall', label: 'Jatuhan', points: 3 },
]
