import type { WsMatchEvent, WsTarget } from '@/composables/useWebSocket'

export type LogCategory = 'poin' | 'sanksi' | 'kontrol' | 'juri' | 'sinyal'

export interface LogEntry {
  id: string
  round: number
  clockMs: number
  title: string
  detail: string
  category: LogCategory
  target?: WsTarget
}

/** Nama teknik sesuai tombol tablet juri (technique=Pukulan, knockdown=Tendangan, fall=Jatuhan). */
export const SCORE_TYPE_LABEL: Record<string, string> = {
  technique: 'Pukulan',
  knockdown: 'Tendangan',
  fall: 'Jatuhan',
  catch_fall: 'Tangkapan + Jatuhan',
}

export const TARGET_LABEL: Record<string, string> = { red: 'MERAH', blue: 'BIRU' }

export function formatClock(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000))
  const m = Math.floor(total / 60)
  const s = total % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

const asNumber = (v: unknown, fallback = 0): number => (typeof v === 'number' ? v : fallback)
const asString = (v: unknown, fallback = ''): string => (typeof v === 'string' ? v : fallback)
const asTarget = (v: unknown): WsTarget | undefined => (v === 'red' || v === 'blue' ? v : undefined)

/** Payload dari REST berupa string JSON, dari WebSocket sudah berupa objek. */
const isRecord = (v: unknown): v is Record<string, unknown> =>
  !!v && typeof v === 'object' && !Array.isArray(v)

export function parsePayload(raw: unknown): Record<string, unknown> {
  if (typeof raw === 'string') {
    try {
      const parsed: unknown = JSON.parse(raw)
      return isRecord(parsed) ? parsed : {}
    } catch {
      return {}
    }
  }
  return isRecord(raw) ? raw : {}
}

export interface RawMatchEvent {
  id: string
  event_type: string
  payload: unknown
  judge_id?: string
  created_at?: string
}

/** Ubah satu kejadian server menjadi baris log yang bisa dibaca; null bila tidak dikenali. */
export function toLogEntry(ev: RawMatchEvent | WsMatchEvent): LogEntry | null {
  const p = parsePayload(ev.payload)
  const round = asNumber(p.round)
  const clockMs = asNumber(p.clock_ms)
  const target = asTarget(p.target)
  const side = target ? `Sudut ${TARGET_LABEL[target]}` : ''
  const base = { id: String(ev.id), round, clockMs, target }

  switch (ev.event_type) {
    case 'match_started':
      return {
        ...base,
        category: 'kontrol',
        title: 'Pertandingan dimulai',
        detail: `Babak ${round}`,
      }
    case 'match_paused':
      return {
        ...base,
        category: 'kontrol',
        title: 'Pertandingan dijeda',
        detail: `Sisa waktu ${formatClock(asNumber(p.remaining_ms, clockMs))}`,
      }
    case 'match_resumed':
      return {
        ...base,
        category: 'kontrol',
        title: 'Pertandingan dilanjutkan',
        detail: `Babak ${round}`,
      }
    case 'match_cancelled':
      return {
        ...base,
        category: 'kontrol',
        title: 'Pertandingan dibatalkan',
        detail: 'Tanpa pemenang',
      }
    case 'match_finished': {
      const corner = asString(p.corner)
      return {
        ...base,
        category: 'kontrol',
        title: 'Pertandingan selesai',
        detail: corner
          ? `Pemenang Sudut ${TARGET_LABEL[corner] ?? corner.toUpperCase()} • ${asString(p.result_type)}`
          : '',
      }
    }
    case 'round_ended':
      return {
        ...base,
        category: 'kontrol',
        title: `Babak ${round} selesai`,
        detail: 'Waktu babak habis',
      }
    case 'round_started':
      return {
        ...base,
        category: 'kontrol',
        title: `Babak ${round} dimulai`,
        detail: 'Istirahat berakhir',
      }
    case 'time_adjusted': {
      const deltaSec = Math.round(asNumber(p.delta_ms) / 1000)
      return {
        ...base,
        category: 'kontrol',
        title: `Waktu dikoreksi ${deltaSec > 0 ? '+' : ''}${deltaSec} detik`,
        detail: `Sisa waktu ${formatClock(asNumber(p.remaining_ms))}`,
      }
    }
    case 'round_reset':
      return {
        ...base,
        category: 'kontrol',
        title: 'Waktu babak di-reset',
        detail: `Babak ${round}`,
      }
    case 'score_confirmed': {
      const label = SCORE_TYPE_LABEL[asString(p.score_type)] ?? asString(p.score_type)
      return {
        ...base,
        category: 'poin',
        title: `Poin sah ${side} +${asNumber(p.points)}`,
        detail: `${label} • konsensus ${asNumber(p.count)} juri`,
      }
    }
    case 'judge_input': {
      const label = SCORE_TYPE_LABEL[asString(p.score_type)] ?? asString(p.score_type)
      return {
        ...base,
        category: 'juri',
        title: `Juri ${asNumber(p.seat_number) || '?'}: ${label} ${side}`,
        detail: 'Input mentah juri',
      }
    }
    case 'penalty_applied':
      return {
        ...base,
        category: 'sanksi',
        title: `${asString(p.label, 'Sanksi')} ${side} (-${asNumber(p.points)})`,
        detail: 'Dijatuhkan wasit',
      }
    case 'penalty_revoked':
      return {
        ...base,
        category: 'sanksi',
        title: `${asString(p.label, 'Sanksi')} ${side} dibatalkan`,
        detail: `Poin dikembalikan +${asNumber(p.points)}`,
      }
    case 'outcome_declared':
      return {
        ...base,
        category: 'kontrol',
        title: asString(p.label, 'Hasil khusus'),
        detail:
          `Pemenang Sudut ${TARGET_LABEL[asString(p.winner_corner)] ?? ''}` +
          (asString(p.note) ? ` • ${asString(p.note)}` : ''),
      }
    case 'tiebreak_resolved':
      return {
        ...base,
        category: 'kontrol',
        title: 'Seri diselesaikan aturan seri',
        detail: `Pemenang Sudut ${TARGET_LABEL[asString(p.winner_corner)] ?? ''} • ${asString(p.detail)}`,
      }
    case 'referee_decision':
      return {
        ...base,
        category: 'kontrol',
        title: 'Keputusan wasit',
        detail: `Pemenang Sudut ${TARGET_LABEL[asString(p.winner_corner)] ?? ''} • ${asString(p.method)}`,
      }
    case 'disqualified':
      return {
        ...base,
        category: 'sanksi',
        title: `DISKUALIFIKASI ${side}`,
        detail: `Pemenang Sudut ${TARGET_LABEL[asString(p.winner_corner)] ?? ''}`,
      }
    case 'arena_signal': {
      const signal = asString(p.signal)
      const active = p.active === true
      if (signal === 'gong')
        return { ...base, category: 'sinyal', title: 'Gong dibunyikan', detail: 'Sinyal arena' }
      if (signal === 'medic')
        return {
          ...base,
          category: 'sinyal',
          title: active ? 'Tim medis dipanggil' : 'Panggilan medis selesai',
          detail: 'Dokter pertandingan',
        }
      if (signal === 'var')
        return {
          ...base,
          category: 'sinyal',
          title: active ? 'Tinjauan video (VAR) dimulai' : 'Tinjauan video (VAR) selesai',
          detail: 'Peninjauan tayangan ulang',
        }
      return null
    }
    default:
      return null
  }
}

export interface TechniqueCounts {
  /** jumlah poin sah per kode teknik (`score_type`) */
  byType: Record<string, number>
  total: number
}

const emptyCounts = (): TechniqueCounts => ({ byType: {}, total: 0 })

/**
 * Hitung poin sah per teknik dari kejadian `score_confirmed` (satu kejadian = satu poin sah).
 * `known` (kode teknik ruleset) membatasi yang dihitung; tanpa itu semua jenis yang berpoin dihitung.
 */
export function countTechniques(
  events: Array<Pick<RawMatchEvent, 'event_type' | 'payload'>>,
  known?: string[],
): Record<WsTarget, TechniqueCounts> {
  const out: Record<WsTarget, TechniqueCounts> = { red: emptyCounts(), blue: emptyCounts() }
  for (const ev of events) {
    if (ev.event_type !== 'score_confirmed') continue
    const p = parsePayload(ev.payload)
    const target = asTarget(p.target)
    if (!target) continue
    const type = asString(p.score_type)
    if (!type || type === 'undo') continue
    if (known ? !known.includes(type) : !(type in SCORE_TYPE_LABEL)) continue
    out[target].byType[type] = (out[target].byType[type] ?? 0) + 1
    out[target].total++
  }
  return out
}
