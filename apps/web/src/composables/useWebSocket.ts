import { getErrorMessage } from '@/api/error'
import { ref, onMounted, onUnmounted } from 'vue'
import { serverClock } from './serverClock'

/** Status mesin pertandingan yang dikirim server. */
export type WsMatchState = string

/** Skor satu ronde pada `sync_response.scores` (struct Go tanpa tag JSON, jadi PascalCase). */
export interface WsRoundScore {
  RedTechnical: number
  BlueTechnical: number
  RedPenalties: number
  BluePenalties: number
}

export type WsTarget = 'red' | 'blue'

/** Satu jenis sanksi resmi dari katalog server (poin = potongan). */
export interface WsPenaltyKind {
  type: string
  label: string
  points: number
}

/** Sanksi yang sudah dijatuhkan pada pertandingan berjalan. */
export interface WsPenaltyEntry {
  id: number
  round: number
  target: WsTarget
  type: string
  label: string
  points: number
  clock_ms: number
}

/** Input juri terakhir pada satu kursi. */
export interface WsJudgeAction {
  score_type: string
  target: string
  round: number
  clock_ms: number
  at: string
}

export interface WsJudgePresence {
  seat: number
  connected: boolean
  inputs: number
  last_input: WsJudgeAction | null
}

/** Perangkat yang sedang terhubung ke gelanggang. */
export interface WsPresence {
  judges: WsJudgePresence[]
  referees: number
  admins: number
  scoreboards: number
}

/** Hasil akhir pertandingan (setelah selesai). */
export interface WsMatchOutcome {
  winner: string
  corner: string
  result_type: string
}

/** Kejadian pertandingan (baris `match_events`), juga disiarkan real-time ke meja wasit. */
export interface WsMatchEvent {
  id: string
  match_id: string
  event_type: string
  payload: Record<string, unknown>
  judge_id: string
  created_at: string
}

export interface WsArenaSignal {
  signal: 'gong' | 'medic' | 'var'
  active: boolean
}

export interface WsPenaltyList {
  penalties: WsPenaltyEntry[]
}

export interface WsSyncResponse {
  match_id?: string
  arena_id?: string
  timer_ends_at_ms?: number
  server_time_ms?: number
  tie_pending?: boolean
  ruleset_name?: string
  round_duration_ms?: number
  break_duration_ms?: number
  total_rounds?: number
  penalties?: WsPenaltyEntry[]
  penalty_catalog?: WsPenaltyKind[]
  medic_active?: boolean
  var_active?: boolean
  presence?: WsPresence
  result?: WsMatchOutcome | null
  round?: number
  state?: WsMatchState
  timer_remaining_ms?: number
  timer_running?: boolean
  red_total?: number
  blue_total?: number
  /** Skor per ronde, indeks 0 = ronde 1 */
  scores?: WsRoundScore[]
}

export interface WsStateChange {
  round?: number
  state?: WsMatchState
  timer_remaining_ms?: number
  timer_ends_at_ms?: number
  server_time_ms?: number
  /** false saat wasit menjeda pertandingan; state tetap `round_N` */
  timer_running?: boolean
  /** Waktu ronde terakhir habis dengan skor seri: menunggu keputusan wasit */
  tie_pending?: boolean
}

/** Koreksi waktu berkala dari server selama timer berjalan. */
export interface WsTimerSync {
  timer_remaining_ms?: number
  timer_running?: boolean
  /** Tenggat absolut (epoch ms); 0 bila berhenti */
  timer_ends_at_ms?: number
  server_time_ms?: number
}

export interface WsScoreConfirmed {
  score_type: string
  target: string
  points: number
  count?: number
  round?: number
}

export interface WsScoreUpdate {
  round?: number
  red_technical?: number
  blue_technical?: number
  red_penalties?: number
  blue_penalties?: number
}

export interface WsMatchResult {
  corner?: string
  result_type?: string
}

export interface WsErrorPayload {
  code: string
  message: string
}

/** Pesan server → klien: nama event ke bentuk payload-nya. */
export interface WsServerMessages {
  sync_response: WsSyncResponse
  state_change: WsStateChange
  timer_sync: WsTimerSync
  score_confirmed: WsScoreConfirmed
  score_update: WsScoreUpdate
  match_result: WsMatchResult
  penalties: WsPenaltyList
  arena_signal: WsArenaSignal
  presence: WsPresence
  match_event: WsMatchEvent
  error: WsErrorPayload
}

export type WsServerMessageType = keyof WsServerMessages

export interface WsMessage<T = unknown> {
  type: string
  payload: T
  from?: string
}

type Listener = (payload: never) => void

export function useWebSocket(arenaId: string, token?: string) {
  const ws = ref<WebSocket | null>(null)
  const isConnected = ref(false)
  const lastError = ref<string | null>(null)
  const messageListeners = new Map<string, Listener[]>()

  let reconnectTimeout: ReturnType<typeof setTimeout> | null = null
  let reconnectAttempts = 0

  function connect() {
    if (!arenaId) return
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const defaultHost = window.location.host
    const baseUrl = import.meta.env.VITE_WS_BASE_URL || `${protocol}//${defaultHost}`
    const query = token ? `?token=${encodeURIComponent(token)}` : ''
    const url = `${baseUrl}/ws/arena/${arenaId}${query}`

    try {
      ws.value = new WebSocket(url)

      ws.value.onopen = () => {
        isConnected.value = true
        lastError.value = null
        reconnectAttempts = 0
        // Request immediate state sync
        sendMessage('sync_request', {})
      }

      ws.value.onmessage = (event) => {
        try {
          const data: WsMessage = JSON.parse(event.data)
          // selaraskan jam klien dengan jam server dari setiap pesan yang membawanya
          const serverTime = (data.payload as { server_time_ms?: unknown } | null)?.server_time_ms
          if (typeof serverTime === 'number') serverClock.observe(serverTime)
          const handlers = messageListeners.get(data.type) || []
          handlers.forEach((fn) => (fn as (payload: unknown) => void)(data.payload))
        } catch (err) {
          console.error('[WS] Failed to parse message:', err)
        }
      }

      ws.value.onclose = () => {
        isConnected.value = false
        scheduleReconnect()
      }

      ws.value.onerror = () => {
        lastError.value = 'WebSocket connection error'
        ws.value?.close()
      }
    } catch (err) {
      lastError.value = getErrorMessage(err, 'Connection failed')
      scheduleReconnect()
    }
  }

  function scheduleReconnect() {
    if (reconnectTimeout) clearTimeout(reconnectTimeout)
    // Exponential backoff: min 1s, max 10s
    const delay = Math.min(1000 * Math.pow(1.5, reconnectAttempts), 10000)
    reconnectAttempts++
    reconnectTimeout = setTimeout(connect, delay)
  }

  /** Mengirim pesan; mengembalikan false bila koneksi belum/tidak terbuka (pesan tidak terkirim). */
  function sendMessage(type: string, payload: Record<string, unknown>): boolean {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify({ type, payload }))
      return true
    }
    return false
  }

  function onMessage<K extends WsServerMessageType>(
    type: K,
    callback: (payload: WsServerMessages[K]) => void,
  ) {
    if (!messageListeners.has(type)) {
      messageListeners.set(type, [])
    }
    messageListeners.get(type)!.push(callback)
  }

  onMounted(connect)
  onUnmounted(() => {
    if (reconnectTimeout) clearTimeout(reconnectTimeout)
    ws.value?.close()
  })

  return {
    isConnected,
    lastError,
    sendMessage,
    onMessage,
  }
}
