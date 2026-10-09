<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useWebSocket } from '@/composables/useWebSocket'
import { useTimer } from '@/composables/useTimer'
import { DEFAULT_TECHNIQUES } from '@/composables/techniques'
import { useConsensusFeedback } from '@/composables/consensusFeedback'
import type { WsTechniqueKind } from '@/composables/useWebSocket'
import {
  MATCH_STATUS_LABEL,
  matchStatusFromState,
  type MatchStatus,
} from '@/composables/matchStatus'
import { useHaptics } from '@/composables/useHaptics'
import { generateUUID } from '@/lib/uuid'
import { useAuthStore } from '@/stores/auth'
import { authService } from '@/features/auth/api/authService'
import { MAX_PIN_LENGTH, isCompletePin, pinLoginErrorMessage } from '@/features/auth/lib/pin'
import { arenaService } from '@/features/arena/api/arenaService'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

// Query or session params
const arenaId = computed(
  () =>
    authStore.arenaId ||
    (route.query.arena_id as string) ||
    (route.params.arena_id as string) ||
    '',
)
const compId = computed(() => authStore.competitionId || (route.query.comp_id as string) || '')
const judgeSeat = computed(() => authStore.seatNumber || Number(route.query.seat) || 1)

// Nama gelanggang dari server (ID hanya untuk koneksi, tidak ditampilkan ke pengguna)
const arenaName = ref('')
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
async function loadArenaName() {
  arenaName.value = ''
  if (!authStore.token || !compId.value || !UUID_RE.test(arenaId.value)) return
  try {
    const arenas = await arenaService.getArenas(compId.value)
    arenaName.value = arenas.find((a) => a.id === arenaId.value)?.name ?? ''
  } catch {
    // nama tidak tersedia: tampilan memakai label umum
  }
}
watch([() => authStore.token, arenaId, compId], loadArenaName, { immediate: true })

const arenaLabel = computed(() => (arenaName.value || 'GELANGGANG').toUpperCase())
const judgeName = computed(() => `JURI ${judgeSeat.value} • ${arenaLabel.value}`)
const judgeIdTag = computed(() => `JURI-ID #${String(judgeSeat.value).padStart(2, '0')}`)

// --- Match Context ---
const currentRound = ref(1)
const matchStatus = ref<MatchStatus>('idle')
const roundStatus = computed(() => MATCH_STATUS_LABEL[matchStatus.value])
// Skor ditolak server saat istirahat; tombol dikunci agar juri tidak mengira input tercatat.
const isBreak = computed(() => matchStatus.value === 'break')
// Skor juga ditolak saat partai dijeda wasit; tombol dikunci dengan alasan yang sama.
const isPaused = computed(() => matchStatus.value === 'paused')
const scoringLocked = computed(() => isBreak.value || isPaused.value)

// Pesan penolakan dari server (mis. skor saat dijeda, BATAL tanpa input yang bisa dibatalkan)
const notice = ref('')
let noticeTimer: ReturnType<typeof setTimeout> | undefined
function showNotice(text: string) {
  notice.value = text
  clearTimeout(noticeTimer)
  noticeTimer = setTimeout(() => (notice.value = ''), 4000)
}
const ERROR_TEXT: Record<string, string> = {
  SCORING_NOT_ALLOWED: 'Skor tidak diterima: pertandingan sedang dijeda atau tidak berjalan.',
  UNDO_NOT_POSSIBLE:
    'Tidak ada input yang bisa dibatalkan (poin terkonfirmasi tidak dapat dibatalkan).',
}

// Daftar tombol teknik mengikuti ruleset pertandingan yang dimuat (nilai poin dari server)
const techniques = ref<WsTechniqueKind[]>(DEFAULT_TECHNIQUES)
const TECHNIQUE_HINT: Record<string, string> = {
  technique: 'SERANGAN TANGAN SAH',
  knockdown: 'SERANGAN KAKI SAH',
  fall: 'TEKNIK BANTINGAN / SAPUAN',
  catch_fall: 'TANGKAPAN LALU JATUHKAN',
  combo_1_2: 'PUKULAN LALU TENDANGAN',
  combo_1_1: 'DUA PUKULAN BERUNTUN',
}
// Lebih dari 4 teknik: tombol disusun dua kolom agar semuanya tetap terlihat tanpa menggulir
const denseButtons = computed(() => techniques.value.length > 4)
const techniqueHint = (type: string) => TECHNIQUE_HINT[type] ?? 'TEKNIK SAH'
const latencyMs = ref(12)

const redAthlete = ref({ name: 'MENUNGGU PERTANDINGAN', team: '', category: '' })
const blueAthlete = ref({ name: 'MENUNGGU PERTANDINGAN', team: '', category: '' })

// Umpan balik konsensus: lampu per sudut = jumlah juri yang sudah sepakat (anonim), jumlah lampu = juri ruleset
const feedback = useConsensusFeedback(() => techniques.value)
feedback.attach()
const redConsensusDots = feedback.dots('red')
const blueConsensusDots = feedback.dots('blue')

// Touch Logs
interface TouchLog {
  technique: string
  points: number
  target: 'red' | 'blue'
  timeStr: string
  idempotencyKey: string
}

const lastRedTouch = ref<TouchLog | null>(null)

const lastBlueTouch = ref<TouchLog | null>(null)
// Sudut yang terakhir disentuh: BATAL menarik input terakhir ini, bukan selalu merah.
const lastTapTarget = ref<'red' | 'blue' | null>(null)

// Nama atlet dari pertandingan yang sedang aktif di gelanggang ini (bila ada)
async function loadAthleteNames() {
  try {
    const snapshot = await arenaService.getLiveScoreboard(arenaId.value)
    if (snapshot.red_athlete) redAthlete.value.name = snapshot.red_athlete
    if (snapshot.blue_athlete) blueAthlete.value.name = snapshot.blue_athlete
  } catch {
    // tetap memakai placeholder
  }
}

let syncedMatchId = ''

// Haptics
const haptics = useHaptics()

// Timer
const timer = useTimer()

// Real-Time WebSocket
const currentToken = computed(() => authStore.token || undefined)
const { isConnected, sendMessage, onMessage } = useWebSocket(arenaId.value, currentToken.value)

// Active button tap animation state
const activeBtn = ref<string | null>(null)

// PIN Modal State
const showPinModal = ref(!authStore.isAuthenticated)
const pinInput = ref('')
const pinSlots = computed(() => (pinInput.value.length > 6 ? 8 : 6))
const pinError = ref('')
const pinLoading = ref(false)

function handlePinNumber(num: number | string) {
  if (pinInput.value.length < MAX_PIN_LENGTH) {
    pinInput.value += String(num)
  }
}

function handlePinClear() {
  pinInput.value = ''
  pinError.value = ''
}

async function handlePinSubmit() {
  if (!isCompletePin(pinInput.value)) {
    pinError.value = 'PIN harus 6 digit (juri) atau 8 digit (wasit)'
    return
  }
  pinLoading.value = true
  pinError.value = ''
  try {
    const res = await authService.judgeLogin({ pin: pinInput.value })

    authStore.setSession({
      token: res.token,
      role: res.role || 'judge',
      seatNumber: res.seat_number,
      arenaId: res.arena_id,
      competitionId: res.competition_id,
    })
    showPinModal.value = false
  } catch (err) {
    pinError.value = pinLoginErrorMessage(err)
  } finally {
    pinLoading.value = false
  }
}

async function handleJudgeLogout() {
  await authStore.logout()
  showPinModal.value = false
  router.push('/judge/login')
}

function handleScoreTap(target: 'red' | 'blue', technique: string, points: number, label: string) {
  // 1. Tactile feedback
  haptics.trigger(50)

  // 2. Button flash state
  const btnKey = `${target}-${technique}`
  activeBtn.value = btnKey
  setTimeout(() => {
    if (activeBtn.value === btnKey) activeBtn.value = null
  }, 350)

  // 3. Generate idempotency key
  const eventId = generateUUID()
  const idempotencyKey = eventId

  lastTapTarget.value = target

  // 4. Update local feedback
  const log: TouchLog = {
    technique: label,
    points,
    target,
    timeStr: timer.formattedTime.value,
    idempotencyKey,
  }

  if (target === 'red') {
    lastRedTouch.value = log
  } else {
    lastBlueTouch.value = log
  }

  // 5. Send real-time payload via WebSocket to backend
  sendMessage('judge_input', {
    event_id: eventId,
    idempotency_key: idempotencyKey,
    score_type: technique,
    target,
    round: currentRound.value,
  })
}

// Undo (BATAL) last touch
function handleCancelLast() {
  haptics.trigger([80, 50, 80])
  const lastTarget = lastTapTarget.value
  if (!lastTarget) {
    showNotice(ERROR_TEXT.UNDO_NOT_POSSIBLE)
    return
  }
  if (lastTarget === 'red') lastRedTouch.value = null
  else lastBlueTouch.value = null
  lastTapTarget.value = null

  sendMessage('judge_input', {
    event_id: generateUUID(),
    idempotency_key: generateUUID(),
    score_type: 'undo',
    target: lastTarget,
    round: currentRound.value,
  })
}

onMounted(() => {
  void loadAthleteNames()

  // Request state sync from backend hub
  sendMessage('sync_request', {})

  onMessage('sync_response', (payload) => {
    // partai baru dimuat ke gelanggang: muat ulang nama atlet
    if (payload.match_id && payload.match_id !== syncedMatchId) {
      syncedMatchId = payload.match_id
      void loadAthleteNames()
    }
    if (payload.techniques?.length) techniques.value = payload.techniques
    feedback.configure(payload.judge_count, payload.consensus?.majority)
    matchStatus.value =
      matchStatusFromState(payload.state, payload.timer_running) ?? matchStatus.value
    if (payload.round) currentRound.value = payload.round
    if (payload.timer_remaining_ms !== undefined) {
      timer.sync(
        payload.timer_remaining_ms,
        payload.timer_running ?? false,
        payload.timer_ends_at_ms,
      )
    }
  })

  onMessage('consensus_progress', (payload) => feedback.onProgress(payload))
  onMessage('score_confirmed', (payload) => feedback.onConfirmed(payload))
  onMessage('error', (payload) =>
    showNotice(ERROR_TEXT[payload.code] ?? payload.message ?? 'Perintah ditolak server.'),
  )

  onMessage('state_change', (payload) => {
    matchStatus.value =
      matchStatusFromState(payload.state, payload.timer_running) ?? matchStatus.value
    if (matchStatus.value === 'break' || matchStatus.value === 'finished') feedback.reset()
    if (payload.round) currentRound.value = payload.round
    if (payload.timer_remaining_ms !== undefined) {
      timer.sync(
        payload.timer_remaining_ms,
        payload.timer_running ?? true,
        payload.timer_ends_at_ms,
      )
    }
  })

  onMessage('timer_sync', (payload) => {
    if (payload.timer_remaining_ms !== undefined) {
      timer.sync(
        payload.timer_remaining_ms,
        payload.timer_running ?? false,
        payload.timer_ends_at_ms,
      )
    }
  })

  window.addEventListener('keydown', handleGlobalKey)
})

function handleGlobalKey(e: KeyboardEvent) {
  if (showPinModal.value) {
    if (e.key >= '0' && e.key <= '9') {
      handlePinNumber(e.key)
    } else if (e.key === 'Backspace') {
      pinInput.value = pinInput.value.slice(0, -1)
    } else if (e.key === 'Escape' || e.key === 'Delete') {
      handlePinClear()
    } else if (e.key === 'Enter') {
      if (isCompletePin(pinInput.value)) {
        void handlePinSubmit()
      }
    }
  }
}

const canRequestFullscreen = computed(
  () => typeof document !== 'undefined' && !!document.documentElement?.requestFullscreen,
)

async function requestLandscapeFullscreen() {
  try {
    if (!document.fullscreenElement && document.documentElement?.requestFullscreen) {
      await document.documentElement.requestFullscreen()
    }
    const orient = screen.orientation as unknown as
      { lock?: (mode: string) => Promise<void> } | undefined
    if (orient?.lock) {
      await orient.lock('landscape')
    }
  } catch {
    // browser fallback: panduan overlay CSS tetap aktif bila orientasi belum diputar
  }
}

onUnmounted(() => {
  window.removeEventListener('keydown', handleGlobalKey)
})
</script>

<template>
  <div class="judge-screen">
    <!-- ROTATE DEVICE OVERLAY (Wajib Landscape di Mobile & Tablet) -->
    <aside class="judge-rotate-prompt font-mono" aria-label="Instruksi Orientasi Layar">
      <div class="rotate-icon-wrapper">
        <svg
          class="rotate-phone-icon"
          width="50"
          height="50"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.8"
        >
          <rect x="4" y="2" width="16" height="20" rx="3" />
          <line x1="12" y1="18" x2="12.01" y2="18" stroke-width="2.5" />
        </svg>
        <svg
          class="rotate-arrow-icon"
          width="28"
          height="28"
          viewBox="0 0 24 24"
          fill="none"
          stroke="#10b981"
          stroke-width="2.5"
        >
          <path d="M21.5 2v6h-6" />
          <path d="M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67" />
        </svg>
      </div>
      <h2 class="rotate-prompt-title font-sans">PUTAR KE MODE LANSKAP</h2>
      <p class="rotate-prompt-desc font-mono">
        Layar Penilaian Juri Silat wajib digunakan dalam posisi mendatar (landscape) untuk akses
        simultan Sudut Merah &amp; Sudut Biru.
      </p>
      <button
        v-if="canRequestFullscreen"
        type="button"
        class="btn-rotate-fullscreen font-mono"
        @click="requestLandscapeFullscreen"
      >
        Layar Penuh / Kunci Lanskap ↗
      </button>
    </aside>

    <!-- TOP NAVIGATION & STATUS BAR -->
    <header class="judge-topbar">
      <!-- Left: Identity -->
      <div class="judge-topbar__left">
        <span class="judge-online-pill" :class="{ 'judge-online-pill--offline': !isConnected }">
          <span class="judge-dot" :class="{ 'judge-dot--offline': !isConnected }" />
          {{ isConnected ? 'ONLINE' : 'LOCAL' }}
        </span>
        <div class="judge-id-group">
          <h1 class="judge-main-title">{{ judgeName }}</h1>
          <span class="judge-sub-title font-mono">PERSILAT DIGITAL SCORING</span>
        </div>
      </div>

      <!-- Center: Round & Timer -->
      <div class="judge-topbar__center">
        <div class="judge-round-badge font-mono">
          <span class="judge-round-dot" />
          BABAK {{ currentRound }} - {{ roundStatus }}
        </div>
        <div class="judge-timer-badge font-mono">
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="12" cy="12" r="10" />
            <polyline points="12 6 12 12 16 14" />
          </svg>
          <span>{{ timer.formattedTime.value }}</span>
        </div>
        <div v-if="isPaused" class="judge-break-note judge-paused-note font-mono">
          Pertandingan dijeda wasit — skor tidak diterima
        </div>
        <div v-if="notice" class="judge-break-note judge-notice font-mono" role="alert">
          {{ notice }}
        </div>
        <div v-if="isBreak" class="judge-break-note font-mono">
          Babak berikutnya dimulai otomatis dalam {{ timer.formattedTime.value }}
        </div>
      </div>

      <!-- Right: Action Icons & Cancel Button -->
      <div class="judge-topbar__right font-mono">
        <span class="judge-chip-btn">TANDING</span>
        <span class="judge-chip-text">LOG SENTUHAN</span>
        <span class="judge-chip-text">STATUS ARENA</span>

        <span class="judge-sync-chip">
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="#34D399"
            stroke-width="2"
          >
            <path
              d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"
            />
          </svg>
          {{ latencyMs }}ms SYNCED
        </span>

        <!-- Haptics Toggle -->
        <button
          class="judge-icon-btn"
          :class="{ 'judge-icon-btn--active': haptics.isEnabled.value }"
          title="Toggle Getaran / Haptics"
          @click="haptics.toggle"
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9" />
            <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0" />
          </svg>
        </button>

        <!-- Cancel / Undo Button -->
        <button class="judge-batal-btn" @click="handleCancelLast">
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
          >
            <path d="M9 14 4 9l5-5" />
            <path d="M4 9h10.5a5.5 5.5 0 0 1 5.5 5.5v0a5.5 5.5 0 0 1-5.5 5.5H11" />
          </svg>
          <span>BATAL</span>
        </button>

        <div class="judge-avatar" title="Ganti Sesi Juri / PIN" @click="showPinModal = true">
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
            <circle cx="12" cy="7" r="4" />
          </svg>
        </div>
      </div>
    </header>

    <!-- COMPLIANCE SUB-HEADER STRIP -->
    <section class="judge-compliance-strip font-mono">
      <div class="judge-comp-left">
        <svg
          width="15"
          height="15"
          viewBox="0 0 24 24"
          fill="none"
          stroke="#34D399"
          stroke-width="2"
        >
          <path d="m14 12-8.5 8.5a2.12 2.12 0 1 1-3-3L11 9" />
          <path d="M15 13 9 7l4-4 6 6z" />
        </svg>
        <span>KONSUL JURI PERSILAT</span>
      </div>

      <div class="judge-comp-center">
        <svg
          width="15"
          height="15"
          viewBox="0 0 24 24"
          fill="none"
          stroke="#FCA5A5"
          stroke-width="2"
        >
          <path d="M9.88 9.88a3 3 0 1 0 4.24 4.24" />
          <path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68" />
          <path d="M6.61 6.61A13.526 13.526 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61" />
          <line x1="2" x2="22" y1="2" y2="22" />
        </svg>
        <span class="text-peach-200">PASAL 28: SKOR KUMULATIF DISEMBUNYIKAN OTOMATIS</span>
      </div>

      <div class="judge-comp-right">
        <span>SENSOR MEJA: <strong class="text-emerald-400">AKTIF</strong></span>
        <span class="judge-bar-divider">|</span>
        <span class="text-cyan-400">{{ judgeIdTag }}</span>
      </div>
    </section>

    <!-- MAIN TWO-COLUMN SPLIT SCREEN (SUDUT MERAH VS SUDUT BIRU) -->
    <main class="judge-grid">
      <!-- SUDUT MERAH (LEFT COLUMN) -->
      <section class="judge-corner judge-corner--red">
        <!-- Header Sudut Merah -->
        <div class="judge-corner__header">
          <div class="judge-athlete-info">
            <div class="judge-corner-badge judge-corner-badge--red font-mono">SUDUT MERAH</div>
            <h2 class="judge-athlete-name">{{ redAthlete.name }}</h2>
            <div class="judge-athlete-meta font-mono">
              {{ redAthlete.team }} &bull; {{ redAthlete.category }}
            </div>
          </div>

          <div class="judge-consensus-indicator font-mono">
            <span>KONSENSUS JURI</span>
            <div class="judge-dots-row">
              <span
                v-for="(on, idx) in redConsensusDots"
                :key="idx"
                class="judge-dot-light"
                :class="{
                  'judge-dot-light--red': on,
                  'judge-dot-light--confirmed': on && feedback.sides.red.confirmed,
                }"
              />
            </div>
            <span
              v-if="feedback.sides.red.text"
              class="judge-consensus-text"
              :class="{ 'judge-consensus-text--ok': feedback.sides.red.confirmed }"
              data-test="consensus-text-red"
              >{{ feedback.sides.red.text }}</span
            >
          </div>
        </div>

        <!-- 3 Extra-Large Ergonomic Scoring Buttons Red -->
        <div class="judge-buttons-list" :class="{ 'judge-buttons-list--dense': denseButtons }">
          <button
            v-for="tech in techniques"
            :key="tech.type"
            class="judge-btn judge-btn--red"
            :class="{ 'judge-btn--active': activeBtn === 'red-' + tech.type }"
            :disabled="scoringLocked"
            @click="handleScoreTap('red', tech.type, tech.points, tech.label)"
          >
            <div class="judge-btn__icon-box">
              <svg
                width="34"
                height="34"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <template v-if="tech.type === 'technique'">
                  <path d="M18 11V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0" />
                  <path d="M14 10V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v2" />
                  <path d="M10 10.5V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v8" />
                  <path
                    d="M18 8a2 2 0 1 1 4 0v6a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15"
                  />
                </template>
                <template v-else-if="tech.type === 'knockdown'">
                  <circle cx="16" cy="4" r="2" />
                  <path d="m14 7-3 4-4-1" />
                  <path d="m11 11 3 5 4-1" />
                  <path d="m5 17 5-2" />
                </template>
                <template v-else>
                  <circle cx="7" cy="5" r="2" />
                  <path d="M5 9h6l4 5" />
                  <path d="M6 14l2 5" />
                  <path d="M12 17l6 2" />
                </template>
              </svg>
            </div>
            <div class="judge-btn__label-group">
              <span class="judge-btn__title">{{ tech.label.toUpperCase() }}</span>
              <span class="judge-btn__subtitle font-mono">{{ techniqueHint(tech.type) }}</span>
            </div>
            <div class="judge-btn__pts font-mono">+{{ tech.points }}</div>
          </button>
        </div>

        <!-- Touch Feedback Strip Red -->
        <footer class="judge-touch-strip font-mono">
          <div class="judge-touch-last">
            <svg
              width="15"
              height="15"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="M18 11V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0" />
              <path d="M14 10V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v2" />
              <path d="M10 10.5V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v8" />
            </svg>
            <span v-if="lastRedTouch">
              SENTUHAN TERAKHIR:
              <strong>{{ lastRedTouch.technique }} (+{{ lastRedTouch.points }})</strong> [{{
                lastRedTouch.timeStr
              }}]
            </span>
            <span v-else class="text-slate-500">BELUM ADA SENTUHAN</span>
          </div>

          <div v-if="lastRedTouch" class="judge-touch-status">
            <svg
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="#34D399"
              stroke-width="2.5"
            >
              <polyline points="20 6 9 17 4 12" />
            </svg>
            <span>TERCATAT</span>
          </div>
        </footer>
      </section>

      <!-- SUDUT BIRU (RIGHT COLUMN) -->
      <section class="judge-corner judge-corner--blue">
        <!-- Header Sudut Biru -->
        <div class="judge-corner__header">
          <div class="judge-athlete-info">
            <div class="judge-corner-badge judge-corner-badge--blue font-mono">SUDUT BIRU</div>
            <h2 class="judge-athlete-name">{{ blueAthlete.name }}</h2>
            <div class="judge-athlete-meta font-mono">
              {{ blueAthlete.team }} &bull; {{ blueAthlete.category }}
            </div>
          </div>

          <div class="judge-consensus-indicator font-mono">
            <span>KONSENSUS JURI</span>
            <div class="judge-dots-row">
              <span
                v-for="(on, idx) in blueConsensusDots"
                :key="idx"
                class="judge-dot-light"
                :class="{
                  'judge-dot-light--blue': on,
                  'judge-dot-light--confirmed': on && feedback.sides.blue.confirmed,
                }"
              />
            </div>
            <span
              v-if="feedback.sides.blue.text"
              class="judge-consensus-text"
              :class="{ 'judge-consensus-text--ok': feedback.sides.blue.confirmed }"
              data-test="consensus-text-blue"
              >{{ feedback.sides.blue.text }}</span
            >
          </div>
        </div>

        <!-- 3 Extra-Large Ergonomic Scoring Buttons Blue -->
        <div class="judge-buttons-list" :class="{ 'judge-buttons-list--dense': denseButtons }">
          <button
            v-for="tech in techniques"
            :key="tech.type"
            class="judge-btn judge-btn--blue"
            :class="{ 'judge-btn--active': activeBtn === 'blue-' + tech.type }"
            :disabled="scoringLocked"
            @click="handleScoreTap('blue', tech.type, tech.points, tech.label)"
          >
            <div class="judge-btn__icon-box">
              <svg
                width="34"
                height="34"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <template v-if="tech.type === 'technique'">
                  <path d="M18 11V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0" />
                  <path d="M14 10V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v2" />
                  <path d="M10 10.5V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v8" />
                  <path
                    d="M18 8a2 2 0 1 1 4 0v6a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15"
                  />
                </template>
                <template v-else-if="tech.type === 'knockdown'">
                  <circle cx="16" cy="4" r="2" />
                  <path d="m14 7-3 4-4-1" />
                  <path d="m11 11 3 5 4-1" />
                  <path d="m5 17 5-2" />
                </template>
                <template v-else>
                  <circle cx="7" cy="5" r="2" />
                  <path d="M5 9h6l4 5" />
                  <path d="M6 14l2 5" />
                  <path d="M12 17l6 2" />
                </template>
              </svg>
            </div>
            <div class="judge-btn__label-group">
              <span class="judge-btn__title">{{ tech.label.toUpperCase() }}</span>
              <span class="judge-btn__subtitle font-mono">{{ techniqueHint(tech.type) }}</span>
            </div>
            <div class="judge-btn__pts font-mono">+{{ tech.points }}</div>
          </button>
        </div>

        <!-- Touch Feedback Strip Blue -->
        <footer class="judge-touch-strip font-mono">
          <div class="judge-touch-last">
            <svg
              width="15"
              height="15"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="M18 11V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0" />
              <path d="M14 10V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v2" />
              <path d="M10 10.5V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v8" />
            </svg>
            <span v-if="lastBlueTouch">
              SENTUHAN TERAKHIR:
              <strong>{{ lastBlueTouch.technique }} (+{{ lastBlueTouch.points }})</strong> [{{
                lastBlueTouch.timeStr
              }}]
            </span>
            <span v-else class="text-slate-500">BELUM ADA SENTUHAN</span>
          </div>

          <div v-if="lastBlueTouch" class="judge-touch-status">
            <svg
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="#34D399"
              stroke-width="2.5"
            >
              <polyline points="20 6 9 17 4 12" />
            </svg>
            <span>TERCATAT</span>
          </div>
        </footer>
      </section>
    </main>

    <!-- BOTTOM COMPLIANCE STRIP -->
    <footer class="judge-bottom-bar font-mono">
      <div class="judge-bottom__left">STANDAR RESMI PB IPSI / PERSILAT 2024</div>
      <div class="judge-bottom__center text-emerald-400">
        TRANSMISI TERENKRIPSI KE KETUA PERTANDINGAN
      </div>
      <div class="judge-bottom__right">TOTAL SKOR DISEMBUNYIKAN (ATURAN PERSILAT PASAL 28)</div>
    </footer>

    <!-- PIN LOGIN ONBOARDING MODAL -->
    <div v-if="showPinModal" class="judge-pin-backdrop">
      <div class="judge-pin-modal">
        <div class="judge-pin-header">
          <div class="judge-pin-icon">
            <svg
              width="26"
              height="26"
              viewBox="0 0 24 24"
              fill="none"
              stroke="#10B981"
              stroke-width="2.2"
            >
              <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
              <path d="M7 11V7a5 5 0 0 1 10 0v4" />
            </svg>
          </div>
          <div>
            <h2 class="judge-pin-title font-mono">LOGIN TABLET JURI</h2>
            <p class="judge-pin-desc">Masukkan PIN Juri untuk autentikasi ke Server Gelanggang</p>
          </div>
        </div>

        <div v-if="arenaId" class="judge-pin-meta font-mono">
          <div class="judge-pin-meta-item">
            <span class="label">GELANGGANG:</span>
            <span class="val">{{ arenaLabel }}</span>
          </div>
          <div v-if="authStore.seatNumber || route.query.seat" class="judge-pin-meta-item">
            <span class="label">KURSI:</span>
            <span class="val">JURI {{ judgeSeat }}</span>
          </div>
        </div>

        <!-- PIN Display Dots -->
        <div class="judge-pin-display font-mono">
          <span
            v-for="i in pinSlots"
            :key="i"
            class="judge-pin-dot"
            :class="{ 'judge-pin-dot--filled': pinInput.length >= i }"
          >
            {{ pinInput.length >= i ? '●' : '○' }}
          </span>
        </div>

        <div v-if="pinError" class="judge-pin-error font-mono">
          {{ pinError }}
        </div>

        <!-- Numpad Grid -->
        <div class="judge-numpad font-mono">
          <button v-for="n in 9" :key="n" class="numpad-btn" @click="handlePinNumber(n)">
            {{ n }}
          </button>
          <button class="numpad-btn numpad-btn--clear" @click="handlePinClear">C</button>
          <button class="numpad-btn" @click="handlePinNumber(0)">0</button>
          <button
            class="numpad-btn numpad-btn--submit"
            :disabled="pinLoading || !isCompletePin(pinInput)"
            @click="handlePinSubmit"
          >
            {{ pinLoading ? '...' : 'OK' }}
          </button>
        </div>

        <div class="judge-pin-footer font-mono">
          <button
            v-if="authStore.isAuthenticated"
            class="judge-link-btn text-rose-400"
            @click="handleJudgeLogout"
          >
            Keluar dari Tablet Juri (Logout)
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ==========================================================================
   JUDGE TABLET PWA SCORING SCREEN (Figma Node 1:256 / 02_judge_tablet_pwa.png)
   ========================================================================== */

.judge-screen {
  width: 100%;
  min-height: 100dvh;
  height: 100dvh;
  background-color: #050811;
  color: #f8fafc;
  display: flex;
  flex-direction: column;
  padding: clamp(6px, 1.2vw, 14px) clamp(8px, 1.6vw, 18px);
  padding-top: max(clamp(6px, 1.2vw, 14px), env(safe-area-inset-top));
  padding-bottom: max(clamp(6px, 1.2vw, 14px), env(safe-area-inset-bottom));
  padding-left: max(clamp(8px, 1.6vw, 18px), env(safe-area-inset-left));
  padding-right: max(clamp(8px, 1.6vw, 18px), env(safe-area-inset-right));
  overflow-x: hidden;
  overflow-y: auto;
  user-select: none;
  -webkit-touch-callout: none;
  font-family: var(--font-sans);
  box-sizing: border-box;
}

/* --- Topbar --- */
.judge-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: clamp(6px, 1vh, 10px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  gap: 8px;
  flex-shrink: 0;
}

.judge-topbar__left {
  display: flex;
  align-items: center;
  gap: clamp(6px, 1vw, 12px);
  min-width: 0;
}

.judge-online-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background-color: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.4);
  color: #34d399;
  font-family: var(--font-mono);
  font-size: clamp(0.625rem, 1vw, 0.6875rem);
  font-weight: 800;
  padding: 3px 8px;
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.judge-online-pill--offline {
  background-color: rgba(245, 158, 11, 0.15);
  border-color: rgba(245, 158, 11, 0.4);
  color: #fbbf24;
}

.judge-dot {
  width: 7px;
  height: 7px;
  background-color: #10b981;
  border-radius: 50%;
  animation: pulse 1.5s infinite;
  flex-shrink: 0;
}
.judge-dot--offline {
  background-color: #f59e0b;
}

.judge-id-group {
  display: flex;
  flex-direction: column;
  line-height: 1.15;
  min-width: 0;
}

.judge-main-title {
  font-size: clamp(0.875rem, 1.8vw, 1.125rem);
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #ffffff;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.judge-sub-title {
  font-size: clamp(0.5625rem, 1vw, 0.625rem);
  letter-spacing: 0.08em;
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.judge-topbar__center {
  display: flex;
  align-items: center;
  gap: clamp(6px, 1vw, 10px);
  flex-shrink: 0;
}

.judge-round-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: clamp(4px, 0.8vh, 6px) clamp(8px, 1.2vw, 14px);
  background-color: rgba(6, 182, 212, 0.12);
  border: 1px solid rgba(6, 182, 212, 0.35);
  border-radius: var(--radius-sm);
  color: #22d3ee;
  font-size: clamp(0.6875rem, 1.2vw, 0.75rem);
  font-weight: 700;
  white-space: nowrap;
}

.judge-round-dot {
  width: 7px;
  height: 7px;
  background-color: #06b6d4;
  border-radius: 50%;
  animation: pulse 1.5s infinite;
  flex-shrink: 0;
}

.judge-timer-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: clamp(4px, 0.8vh, 6px) clamp(8px, 1.2vw, 14px);
  background-color: rgba(15, 23, 42, 0.85);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: var(--radius-sm);
  font-size: clamp(0.8125rem, 1.5vw, 0.9375rem);
  font-weight: 800;
  color: #38bdf8;
  white-space: nowrap;
}

.judge-topbar__right {
  display: flex;
  align-items: center;
  gap: clamp(6px, 1vw, 10px);
  font-size: 0.75rem;
  flex-shrink: 0;
}

.judge-chip-btn {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  padding: 4px 10px;
  border-radius: var(--radius-xs);
  color: #cbd5e1;
  font-weight: 700;
  font-size: 0.6875rem;
}

.judge-chip-text {
  color: #64748b;
  font-size: 0.6875rem;
}

.judge-sync-chip {
  display: flex;
  align-items: center;
  gap: 5px;
  background-color: rgba(16, 185, 129, 0.08);
  padding: 4px 8px;
  border-radius: var(--radius-xs);
  color: #34d399;
  font-size: 0.6875rem;
  font-weight: 700;
  white-space: nowrap;
}

.judge-icon-btn {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-xs);
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #94a3b8;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast);
  touch-action: manipulation;
  flex-shrink: 0;
}
.judge-icon-btn--active {
  color: #34d399;
  border-color: #10b981;
}

.judge-break-note {
  margin-top: 4px;
  font-size: 11px;
  font-weight: 700;
  color: #fbbf24;
  text-align: center;
}

.judge-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
  filter: grayscale(0.6);
}

.judge-batal-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background-color: #b91c1c;
  color: #ffffff;
  border: none;
  padding: 6px 12px;
  min-height: 32px;
  border-radius: var(--radius-xs);
  font-family: var(--font-mono);
  font-size: clamp(0.6875rem, 1.2vw, 0.75rem);
  font-weight: 800;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(185, 28, 28, 0.4);
  transition: all var(--transition-fast);
  touch-action: manipulation;
  white-space: nowrap;
  flex-shrink: 0;
}
.judge-batal-btn:active {
  transform: scale(0.96);
}

.judge-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background-color: rgba(255, 255, 255, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #cbd5e1;
  cursor: pointer;
  touch-action: manipulation;
  flex-shrink: 0;
}

/* --- Compliance Sub-Header --- */
.judge-compliance-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  background-color: rgba(10, 16, 30, 0.7);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: var(--radius-xs);
  margin-top: 6px;
  font-size: clamp(0.625rem, 1vw, 0.6875rem);
  color: #94a3b8;
  flex-shrink: 0;
}

.judge-comp-left,
.judge-comp-center,
.judge-comp-right {
  display: flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.text-peach-200 {
  color: #fca5a5;
}
.text-cyan-400 {
  color: #22d3ee;
}
.judge-bar-divider {
  opacity: 0.3;
}

/* --- Main Split Screen (Two Columns) --- */
.judge-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: clamp(8px, 1.5vw, 16px);
  margin-top: clamp(6px, 1vh, 10px);
  flex: 1;
  min-height: 0;
}

.judge-corner {
  display: flex;
  flex-direction: column;
  border-radius: var(--radius-md, 12px);
  padding: clamp(8px, 1.4vw, 16px);
  border: 1px solid transparent;
  min-height: 0;
  box-sizing: border-box;
}

.judge-corner--red {
  background: linear-gradient(180deg, rgba(127, 29, 29, 0.25) 0%, rgba(13, 19, 33, 0.95) 100%);
  border-color: rgba(220, 38, 38, 0.35);
}

.judge-corner--blue {
  background: linear-gradient(180deg, rgba(30, 58, 138, 0.25) 0%, rgba(13, 19, 33, 0.95) 100%);
  border-color: rgba(37, 99, 235, 0.35);
}

.judge-corner__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: clamp(6px, 1vh, 12px);
  flex-shrink: 0;
  gap: 6px;
}

.judge-athlete-info {
  min-width: 0;
}

.judge-corner-badge {
  display: inline-block;
  padding: 2px 7px;
  border-radius: var(--radius-xs);
  font-size: clamp(0.5625rem, 1vw, 0.6875rem);
  font-weight: 800;
  letter-spacing: 0.08em;
  margin-bottom: 2px;
}
.judge-corner-badge--red {
  background-color: #dc2626;
  color: #ffffff;
}
.judge-corner-badge--blue {
  background-color: #2563eb;
  color: #ffffff;
}

.judge-athlete-name {
  font-size: clamp(0.95rem, 1.8vw, 1.45rem);
  font-weight: 800;
  line-height: 1.15;
  color: #ffffff;
  letter-spacing: -0.02em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.judge-athlete-meta {
  font-size: clamp(0.5625rem, 0.9vw, 0.6875rem);
  color: #94a3b8;
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.judge-consensus-indicator {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  flex-shrink: 0;
}
.judge-consensus-indicator span {
  font-size: 0.5625rem;
  color: #64748b;
  letter-spacing: 0.08em;
  white-space: nowrap;
}

.judge-dots-row {
  display: flex;
  gap: 4px;
}

.judge-dot-light {
  width: clamp(8px, 1.2vw, 12px);
  height: clamp(8px, 1.2vw, 12px);
  border-radius: 50%;
  background-color: rgba(255, 255, 255, 0.15);
}
.judge-dot-light--red {
  background-color: #ef4444;
  box-shadow: 0 0 8px rgba(239, 68, 68, 0.8);
}
.judge-dot-light--confirmed {
  animation: judge-dot-pop 0.35s ease-out;
}
@keyframes judge-dot-pop {
  0% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.7);
  }
  100% {
    transform: scale(1);
  }
}
.judge-consensus-indicator .judge-consensus-text {
  font-size: clamp(0.5625rem, 0.9vw, 0.625rem);
  color: #fbbf24;
  letter-spacing: 0.04em;
  max-width: 160px;
  text-align: right;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.judge-consensus-indicator .judge-consensus-text--ok {
  color: #34d399;
}
.judge-dot-light--blue {
  background-color: #3b82f6;
  box-shadow: 0 0 8px rgba(59, 130, 246, 0.8);
}

/* --- Extra-Large Ergonomic Scoring Buttons --- */
.judge-buttons-list {
  display: flex;
  flex-direction: column;
  gap: clamp(6px, 1.2vh, 12px);
  flex: 1;
  min-height: 0;
}
.judge-buttons-list--dense {
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-auto-rows: 1fr;
  gap: clamp(6px, 1vh, 10px);
}
.judge-buttons-list--dense .judge-btn {
  min-height: clamp(48px, 6vh, 68px);
  padding: clamp(6px, 1vw, 10px) clamp(8px, 1.2vw, 14px);
}
.judge-buttons-list--dense .judge-btn__title {
  font-size: clamp(0.9rem, 1.6vw, 1.15rem);
  line-height: 1.1;
}
.judge-buttons-list--dense .judge-btn__pts {
  font-size: clamp(1.3rem, 2.4vw, 1.7rem);
}
.judge-buttons-list--dense .judge-btn__icon-box {
  display: none;
}

.judge-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: clamp(8px, 1.2vw, 14px) clamp(10px, 1.8vw, 20px);
  border-radius: var(--radius-md, 12px);
  border: 1px solid transparent;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  transition:
    transform 0.08s ease,
    filter 0.08s ease,
    box-shadow 0.08s ease;
  user-select: none;
  min-height: clamp(52px, 8vh, 80px);
  touch-action: manipulation;
  box-sizing: border-box;
}

/* Touch active tactile bevel */
.judge-btn:active,
.judge-btn--active {
  transform: scale(0.975);
  filter: brightness(1.25);
}

/* Red Button Styling */
.judge-btn--red {
  background: linear-gradient(135deg, #b91c1c 0%, #7f1d1d 100%);
  border-color: rgba(239, 68, 68, 0.6);
  color: #ffffff;
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.2),
    0 6px 18px rgba(185, 28, 28, 0.35);
}
.judge-btn--red.judge-btn--active {
  box-shadow:
    0 0 35px rgba(239, 68, 68, 0.9),
    inset 0 0 15px rgba(255, 255, 255, 0.35);
}

/* Blue Button Styling */
.judge-btn--blue {
  background: linear-gradient(135deg, #1d4ed8 0%, #1e3a8a 100%);
  border-color: rgba(96, 165, 250, 0.6);
  color: #ffffff;
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.2),
    0 6px 18px rgba(29, 78, 216, 0.35);
}
.judge-btn--blue.judge-btn--active {
  box-shadow:
    0 0 35px rgba(96, 165, 250, 0.9),
    inset 0 0 15px rgba(255, 255, 255, 0.35);
}

.judge-btn__icon-box {
  width: clamp(34px, 4.5vw, 56px);
  height: clamp(34px, 4.5vw, 56px);
  border-radius: var(--radius-sm, 8px);
  background-color: rgba(0, 0, 0, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.judge-btn__icon-box svg {
  width: clamp(20px, 2.5vw, 32px);
  height: clamp(20px, 2.5vw, 32px);
}

.judge-btn__label-group {
  display: flex;
  flex-direction: column;
  flex: 1;
  margin-left: clamp(8px, 1.5vw, 18px);
  text-align: left;
  min-width: 0;
}

.judge-btn__title {
  font-size: clamp(1.1rem, 2.4vw, 2.1rem);
  font-weight: 900;
  letter-spacing: -0.02em;
  line-height: 1.05;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.judge-btn__subtitle {
  font-size: clamp(0.5625rem, 0.9vw, 0.6875rem);
  letter-spacing: 0.08em;
  opacity: 0.8;
  margin-top: 3px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.judge-btn__pts {
  font-size: clamp(1.6rem, 3.2vw, 3.25rem);
  font-weight: 900;
  line-height: 1;
  letter-spacing: -0.04em;
  padding: clamp(2px, 0.5vh, 4px) clamp(8px, 1.2vw, 14px);
  border-radius: var(--radius-sm, 8px);
  background-color: rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.15);
  flex-shrink: 0;
}

/* --- Touch Feedback Strip --- */
.judge-touch-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: clamp(6px, 1vh, 10px);
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  margin-top: clamp(6px, 1vh, 10px);
  font-size: clamp(0.625rem, 1vw, 0.6875rem);
  color: #94a3b8;
  flex-shrink: 0;
  gap: 6px;
}

.judge-touch-last {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.judge-touch-last strong {
  color: #ffffff;
}

.judge-touch-status {
  display: flex;
  align-items: center;
  gap: 5px;
  color: #34d399;
  font-weight: 700;
  flex-shrink: 0;
}

/* --- Bottom Compliance Bar --- */
.judge-bottom-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: clamp(4px, 0.8vh, 8px);
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  margin-top: clamp(4px, 0.8vh, 8px);
  font-size: clamp(0.5625rem, 0.85vw, 0.625rem);
  color: #64748b;
  flex-shrink: 0;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.4;
    transform: scale(0.85);
  }
}

/* ==========================================================================
   PIN ONBOARDING MODAL STYLES
   ========================================================================== */
.judge-pin-backdrop {
  position: fixed;
  inset: 0;
  background-color: rgba(2, 6, 23, 0.88);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  padding: 16px;
  box-sizing: border-box;
}

.judge-pin-modal {
  background-color: #0b1120;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: var(--radius-lg, 16px);
  padding: clamp(16px, 4vw, 24px) clamp(14px, 4vw, 26px);
  max-width: 380px;
  width: 100%;
  max-height: 92dvh;
  overflow-y: auto;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.7);
  display: flex;
  flex-direction: column;
  align-items: center;
  box-sizing: border-box;
}

.judge-pin-header {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  margin-bottom: 14px;
}

.judge-pin-icon {
  width: 40px;
  height: 40px;
  background-color: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.3);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.judge-pin-title {
  font-size: clamp(1rem, 3.5vw, 1.125rem);
  font-weight: 800;
  color: #ffffff;
  line-height: 1.2;
}

.judge-pin-desc {
  font-size: 0.6875rem;
  color: #94a3b8;
  margin-top: 2px;
}

.judge-pin-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  background-color: #0f172a;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 6px;
  padding: 6px 12px;
  margin-bottom: 12px;
  font-size: 0.6875rem;
  box-sizing: border-box;
}

.judge-pin-meta-item .label {
  color: #64748b;
  margin-right: 6px;
}

.judge-pin-meta-item .val {
  color: #38bdf8;
  font-weight: 700;
}

.judge-pin-display {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: clamp(8px, 2vw, 12px);
  margin-bottom: 12px;
  font-size: clamp(1.35rem, 4vw, 1.75rem);
  color: #475569;
}

.judge-pin-dot--filled {
  color: #10b981;
  text-shadow: 0 0 10px rgba(16, 185, 129, 0.6);
}

.judge-pin-error {
  color: #ef4444;
  font-size: 0.75rem;
  margin-bottom: 10px;
  text-align: center;
}

.judge-numpad {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: clamp(6px, 1.5vw, 10px);
  width: 100%;
  margin-bottom: 12px;
}

.numpad-btn {
  height: clamp(42px, 6vh, 52px);
  background-color: #162032;
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #f8fafc;
  font-size: clamp(1.1rem, 3vw, 1.25rem);
  font-weight: 700;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  touch-action: manipulation;
  user-select: none;
}

.numpad-btn:hover {
  background-color: #1e293b;
  border-color: rgba(255, 255, 255, 0.2);
}

.numpad-btn:active {
  transform: scale(0.96);
}

.numpad-btn--clear {
  color: #ef4444;
  background-color: rgba(239, 68, 68, 0.08);
  border-color: rgba(239, 68, 68, 0.2);
}

.numpad-btn--submit {
  color: #064e3b;
  background-color: #10b981;
  border-color: #10b981;
}

.numpad-btn--submit:hover {
  background-color: #34d399;
}

.numpad-btn--submit:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

.judge-pin-footer {
  width: 100%;
  text-align: center;
}

.judge-link-btn {
  background: transparent;
  border: none;
  color: #64748b;
  font-size: 0.6875rem;
  cursor: pointer;
  text-decoration: underline;
  transition: color 0.15s ease;
  touch-action: manipulation;
}

.judge-link-btn:hover {
  color: #94a3b8;
}

/* ==========================================================================
   RESPONSIVE MEDIA QUERIES (TABLET PORTRAIT & MOBILE)
   ========================================================================== */

/* Medium tablets / Small screens (< 960px) */
@media (max-width: 960px) {
  .judge-topbar__right .judge-chip-btn,
  .judge-topbar__right .judge-chip-text {
    display: none;
  }
  .judge-comp-center {
    display: none;
  }
}

/* Small tablets / Large phones (< 768px) */
@media (max-width: 768px) {
  .judge-compliance-strip {
    display: none;
  }
  .judge-bottom-bar {
    display: none;
  }
  .judge-btn__subtitle {
    display: none;
  }
  .judge-btn__icon-box {
    display: none;
  }
  .judge-btn__label-group {
    margin-left: 0;
  }
}

/* Phones Portrait (< 580px) */
@media (max-width: 580px) {
  .judge-topbar {
    flex-wrap: wrap;
    gap: 6px;
  }
  .judge-topbar__left {
    order: 1;
  }
  .judge-topbar__right {
    order: 2;
  }
  .judge-topbar__center {
    order: 3;
    width: 100%;
    justify-content: space-between;
    padding-top: 4px;
    border-top: 1px solid rgba(255, 255, 255, 0.05);
  }
  .judge-touch-strip {
    font-size: 0.5625rem;
  }
  .judge-athlete-meta {
    display: none;
  }
  .judge-btn__title {
    font-size: 0.95rem;
  }
  .judge-btn__pts {
    font-size: 1.35rem;
    padding: 2px 6px;
  }
}

/* Landscape Mobile & Low-Height Screens (<= 560px height) */
@media (max-height: 560px) {
  .judge-screen {
    padding: 4px 8px;
  }
  .judge-topbar {
    padding-bottom: 4px;
  }
  .judge-compliance-strip {
    display: none;
  }
  .judge-bottom-bar {
    display: none;
  }
  .judge-corner {
    padding: 6px 10px;
  }
  .judge-corner__header {
    margin-bottom: 4px;
  }
  .judge-athlete-meta {
    display: none;
  }
  .judge-touch-strip {
    margin-top: 4px;
    padding-top: 4px;
  }
  .judge-buttons-list {
    gap: 4px;
  }
  .judge-btn {
    min-height: 42px;
    padding: 4px 10px;
  }
  .judge-btn__icon-box {
    display: none;
  }
  .judge-btn__subtitle {
    display: none;
  }
  .judge-btn__label-group {
    margin-left: 0;
  }
  .judge-btn__title {
    font-size: clamp(0.95rem, 2.5vw, 1.35rem);
  }
  .judge-btn__pts {
    font-size: clamp(1.35rem, 3vw, 1.85rem);
    padding: 2px 8px;
  }
}

/* ==========================================================================
   LANDSCAPE ENFORCEMENT PROMPT (WAJIB LANDSCAPE DI PERANGKAT MOBILE & TABLET)
   ========================================================================== */
.judge-rotate-prompt {
  display: none;
}

@media (max-width: 1024px) and (orientation: portrait) {
  .judge-rotate-prompt {
    position: fixed;
    inset: 0;
    z-index: 9999;
    background-color: rgba(5, 8, 17, 0.98);
    backdrop-filter: blur(12px);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: clamp(20px, 6vw, 36px);
    text-align: center;
    box-sizing: border-box;
  }

  .rotate-icon-wrapper {
    position: relative;
    width: 80px;
    height: 80px;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 20px;
  }

  .rotate-phone-icon {
    color: #cbd5e1;
    animation: rotate-device-anim 2.4s ease-in-out infinite;
    transform-origin: center center;
  }

  .rotate-arrow-icon {
    position: absolute;
    bottom: -4px;
    right: -4px;
    animation: pulse 1.8s infinite;
  }

  @keyframes rotate-device-anim {
    0%,
    15% {
      transform: rotate(0deg);
    }
    45%,
    75% {
      transform: rotate(90deg);
    }
    90%,
    100% {
      transform: rotate(0deg);
    }
  }

  .rotate-prompt-title {
    font-size: clamp(1.2rem, 4.5vw, 1.5rem);
    font-weight: 800;
    color: #ffffff;
    margin-bottom: 8px;
    letter-spacing: -0.01em;
  }

  .rotate-prompt-desc {
    font-size: clamp(0.75rem, 2.8vw, 0.875rem);
    color: #94a3b8;
    max-width: 340px;
    line-height: 1.5;
    margin-bottom: 24px;
  }

  .btn-rotate-fullscreen {
    background-color: #10b981;
    border: 1px solid #059669;
    color: #0f172a;
    font-weight: 700;
    padding: 10px 20px;
    border-radius: var(--radius-sm, 8px);
    font-size: 0.8125rem;
    cursor: pointer;
    touch-action: manipulation;
    box-shadow: 0 4px 14px rgba(16, 185, 129, 0.35);
    transition:
      transform 0.15s ease,
      background-color 0.15s ease;
  }

  .btn-rotate-fullscreen:active {
    transform: scale(0.96);
  }
}
</style>
