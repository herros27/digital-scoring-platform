import { getErrorMessage } from '@/api/error'
import { ref, onMounted, onUnmounted } from 'vue'

/** Status mesin pertandingan yang dikirim server. */
export type WsMatchState = string

/** Skor satu ronde pada `sync_response.scores` (struct Go tanpa tag JSON, jadi PascalCase). */
export interface WsRoundScore {
  RedTechnical: number
  BlueTechnical: number
  RedPenalties: number
  BluePenalties: number
}

export interface WsSyncResponse {
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
  /** false saat wasit menjeda pertandingan; state tetap `round_N` */
  timer_running?: boolean
}

export interface WsTimerSync {
  remaining_ms?: number
  running?: boolean
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

  function sendMessage(type: string, payload: Record<string, unknown>) {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify({ type, payload }))
    }
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
