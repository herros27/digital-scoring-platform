import { computed, onScopeDispose, reactive, ref } from 'vue'
import type {
  WsConsensusProgress,
  WsScoreConfirmed,
  WsTechniqueKind,
} from '@/composables/useWebSocket'

export type ConsensusSide = 'red' | 'blue'

interface SideState {
  lit: number
  text: string
  confirmed: boolean
}

const CONFIRMED_HOLD_MS = 1500

/**
 * Umpan balik konsensus di tablet juri: lampu kecil per sudut menunjukkan berapa juri yang sudah
 * menekan teknik yang sama (anonim), lalu menyala penuh sesaat saat poin terkonfirmasi. Bila tidak
 * ada kesepakatan sampai window habis, lampu padam sendiri.
 */
export function useConsensusFeedback(getTechniques: () => WsTechniqueKind[]) {
  const judgeCount = ref(3)
  const needed = ref(2)
  const sides = reactive<Record<ConsensusSide, SideState>>({
    red: { lit: 0, text: '', confirmed: false },
    blue: { lit: 0, text: '', confirmed: false },
  })
  const timers: Partial<Record<ConsensusSide, ReturnType<typeof setTimeout>>> = {}

  const labelOf = (type: string) =>
    (getTechniques().find((t) => t.type === type)?.label ?? type).toUpperCase()

  function clear(side: ConsensusSide) {
    clearTimeout(timers[side])
    delete timers[side]
    sides[side].lit = 0
    sides[side].text = ''
    sides[side].confirmed = false
  }

  function show(side: ConsensusSide, state: SideState, holdMs: number) {
    clearTimeout(timers[side])
    Object.assign(sides[side], state)
    timers[side] = setTimeout(() => clear(side), holdMs)
  }

  const isSide = (v: string): v is ConsensusSide => v === 'red' || v === 'blue'

  return {
    judgeCount,
    needed,
    sides,
    /** Atur jumlah lampu dan juri yang harus sepakat dari ruleset (dari sync_response). */
    configure(judges?: number, majority?: number) {
      if (judges && judges > 0) judgeCount.value = judges
      if (majority && majority > 0) needed.value = majority
    },
    onProgress(p: WsConsensusProgress) {
      if (!isSide(p.target)) return
      if (p.count <= 0) {
        // juri membatalkan input yang menunggu: padamkan lampu sudut itu
        clear(p.target)
        return
      }
      const need = p.needed || needed.value
      show(
        p.target,
        {
          lit: Math.min(p.count, judgeCount.value),
          text: `${labelOf(p.score_type)} ${p.count}/${need} • MENUNGGU JURI LAIN`,
          confirmed: false,
        },
        p.window_ms || 3000,
      )
    },
    onConfirmed(p: WsScoreConfirmed) {
      if (!isSide(p.target)) return
      show(
        p.target,
        {
          lit: Math.min(p.count ?? needed.value, judgeCount.value),
          text: `${labelOf(p.score_type)} +${p.points} TERKONFIRMASI`,
          confirmed: true,
        },
        CONFIRMED_HOLD_MS,
      )
    },
    reset() {
      clear('red')
      clear('blue')
    },
    dots: (side: ConsensusSide) =>
      computed(() => Array.from({ length: judgeCount.value }, (_, i) => i < sides[side].lit)),
    /** Pasang pembersihan timer ke scope komponen. */
    attach() {
      onScopeDispose(() => {
        clear('red')
        clear('blue')
      })
    },
  }
}
