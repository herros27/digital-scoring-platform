import { ref, onMounted, onUnmounted } from 'vue'

export interface WsMessage<T = any> {
  type: string
  payload: T
  from?: string
}

export function useWebSocket(arenaId: string, token?: string) {
  const ws = ref<WebSocket | null>(null)
  const isConnected = ref(false)
  const lastError = ref<string | null>(null)
  const messageListeners = new Map<string, Array<(payload: any) => void>>()

  let reconnectTimeout: any = null
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
          handlers.forEach(fn => fn(data.payload))
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
    } catch (err: any) {
      lastError.value = err.message || 'Connection failed'
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

  function sendMessage(type: string, payload: any) {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify({ type, payload }))
    }
  }

  function onMessage<T = any>(type: string, callback: (payload: T) => void) {
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
