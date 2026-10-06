<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useWebSocket } from '@/composables/useWebSocket'
import { useTimer } from '@/composables/useTimer'
import { DEFAULT_TECHNIQUES } from '@/composables/techniques'
import type { WsTechniqueKind } from '@/composables/useWebSocket'
import { MATCH_STATUS_LABEL, matchStatusFromState, type MatchStatus } from '@/composables/matchStatus'
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
const arenaId = computed(() => authStore.arenaId || (route.query.arena_id as string) || (route.params.arena_id as string) || '')
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

// Daftar tombol teknik mengikuti ruleset pertandingan yang dimuat (nilai poin dari server)
const techniques = ref<WsTechniqueKind[]>(DEFAULT_TECHNIQUES)
const TECHNIQUE_HINT: Record<string, string> = {
  technique: 'SERANGAN TANGAN SAH',
  knockdown: 'SERANGAN KAKI SAH',
  fall: 'TEKNIK BANTINGAN / SAPUAN',
  catch_fall: 'TANGKAPAN LALU JATUHKAN',
}
const techniqueHint = (type: string) => TECHNIQUE_HINT[type] ?? 'TEKNIK SAH'
const latencyMs = ref(12)

const redAthlete = ref({ name: 'MENUNGGU PERTANDINGAN', team: '', category: '' })
const blueAthlete = ref({ name: 'MENUNGGU PERTANDINGAN', team: '', category: '' })

// Consensus Indicators (5 dots per side)
const redConsensusDots = ref([false, false, false, false, false])
const blueConsensusDots = ref([false, false, false, false, false])

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
  let lastTarget: 'red' | 'blue' | null = null
  if (lastRedTouch.value) {
    lastTarget = 'red'
    lastRedTouch.value = null
  } else if (lastBlueTouch.value) {
    lastTarget = 'blue'
    lastBlueTouch.value = null
  }

  if (lastTarget) {
    sendMessage('judge_input', {
      event_id: generateUUID(),
      idempotency_key: generateUUID(),
      score_type: 'undo',
      target: lastTarget,
      round: currentRound.value,
    })
  }
}

onMounted(() => {
  void loadAthleteNames()

  // Request state sync from backend hub
  sendMessage('sync_request', {})

  onMessage('sync_response', (payload) => {
    if (payload.techniques?.length) techniques.value = payload.techniques
    matchStatus.value = matchStatusFromState(payload.state, payload.timer_running) ?? matchStatus.value
    if (payload.round) currentRound.value = payload.round
    if (payload.timer_remaining_ms !== undefined) {
      timer.sync(payload.timer_remaining_ms, payload.timer_running ?? false, payload.timer_ends_at_ms)
    }
  })

  onMessage('state_change', (payload) => {
    matchStatus.value = matchStatusFromState(payload.state, payload.timer_running) ?? matchStatus.value
    if (payload.round) currentRound.value = payload.round
    if (payload.timer_remaining_ms !== undefined) {
      timer.sync(payload.timer_remaining_ms, payload.timer_running ?? true, payload.timer_ends_at_ms)
    }
  })

  onMessage('timer_sync', (payload) => {
    if (payload.timer_remaining_ms !== undefined) {
      timer.sync(payload.timer_remaining_ms, payload.timer_running ?? false, payload.timer_ends_at_ms)
    }
  })
})
</script>

<template>
  <div class="judge-screen">
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
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <polyline points="12 6 12 12 16 14" />
          </svg>
          <span>{{ timer.formattedTime.value }}</span>
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
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#34D399" stroke-width="2">
            <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
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
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9" />
            <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0" />
          </svg>
        </button>

        <!-- Cancel / Undo Button -->
        <button class="judge-batal-btn" @click="handleCancelLast">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <path d="M9 14 4 9l5-5" />
            <path d="M4 9h10.5a5.5 5.5 0 0 1 5.5 5.5v0a5.5 5.5 0 0 1-5.5 5.5H11" />
          </svg>
          <span>BATAL</span>
        </button>

        <div class="judge-avatar" title="Ganti Sesi Juri / PIN" @click="showPinModal = true">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
            <circle cx="12" cy="7" r="4" />
          </svg>
        </div>
      </div>
    </header>

    <!-- COMPLIANCE SUB-HEADER STRIP -->
    <section class="judge-compliance-strip font-mono">
      <div class="judge-comp-left">
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#34D399" stroke-width="2">
          <path d="m14 12-8.5 8.5a2.12 2.12 0 1 1-3-3L11 9" />
          <path d="M15 13 9 7l4-4 6 6z" />
        </svg>
        <span>KONSUL JURI PERSILAT</span>
      </div>

      <div class="judge-comp-center">
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#FCA5A5" stroke-width="2">
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
            <div class="judge-corner-badge judge-corner-badge--red font-mono">
              SUDUT MERAH
            </div>
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
                :class="{ 'judge-dot-light--red': on }"
              />
            </div>
          </div>
        </div>

        <!-- 3 Extra-Large Ergonomic Scoring Buttons Red -->
        <div class="judge-buttons-list">
          <button
            v-for="tech in techniques"
            :key="tech.type"
            class="judge-btn judge-btn--red"
            :class="{ 'judge-btn--active': activeBtn === 'red-' + tech.type }"
            :disabled="isBreak"
            @click="handleScoreTap('red', tech.type, tech.points, tech.label)"
          >
            <div class="judge-btn__icon-box">
              <svg width="34" height="34" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <template v-if="tech.type === 'technique'">
                  <path d="M18 11V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0" />
                  <path d="M14 10V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v2" />
                  <path d="M10 10.5V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v8" />
                  <path d="M18 8a2 2 0 1 1 4 0v6a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15" />
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
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 11V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0" />
              <path d="M14 10V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v2" />
              <path d="M10 10.5V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v8" />
            </svg>
            <span v-if="lastRedTouch">
              SENTUHAN TERAKHIR: <strong>{{ lastRedTouch.technique }} (+{{ lastRedTouch.points }})</strong> [{{ lastRedTouch.timeStr }}]
            </span>
            <span v-else class="text-slate-500">BELUM ADA SENTUHAN</span>
          </div>

          <div v-if="lastRedTouch" class="judge-touch-status">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#34D399" stroke-width="2.5">
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
            <div class="judge-corner-badge judge-corner-badge--blue font-mono">
              SUDUT BIRU
            </div>
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
                :class="{ 'judge-dot-light--blue': on }"
              />
            </div>
          </div>
        </div>

        <!-- 3 Extra-Large Ergonomic Scoring Buttons Blue -->
        <div class="judge-buttons-list">
          <button
            v-for="tech in techniques"
            :key="tech.type"
            class="judge-btn judge-btn--blue"
            :class="{ 'judge-btn--active': activeBtn === 'blue-' + tech.type }"
            :disabled="isBreak"
            @click="handleScoreTap('blue', tech.type, tech.points, tech.label)"
          >
            <div class="judge-btn__icon-box">
              <svg width="34" height="34" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <template v-if="tech.type === 'technique'">
                  <path d="M18 11V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0" />
                  <path d="M14 10V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v2" />
                  <path d="M10 10.5V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v8" />
                  <path d="M18 8a2 2 0 1 1 4 0v6a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15" />
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
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 11V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0" />
              <path d="M14 10V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v2" />
              <path d="M10 10.5V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v8" />
            </svg>
            <span v-if="lastBlueTouch">
              SENTUHAN TERAKHIR: <strong>{{ lastBlueTouch.technique }} (+{{ lastBlueTouch.points }})</strong> [{{ lastBlueTouch.timeStr }}]
            </span>
            <span v-else class="text-slate-500">BELUM ADA SENTUHAN</span>
          </div>

          <div v-if="lastBlueTouch" class="judge-touch-status">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#34D399" stroke-width="2.5">
              <polyline points="20 6 9 17 4 12" />
            </svg>
            <span>TERCATAT</span>
          </div>
        </footer>
      </section>
    </main>

    <!-- BOTTOM COMPLIANCE STRIP -->
    <footer class="judge-bottom-bar font-mono">
      <div class="judge-bottom__left">
        STANDAR RESMI PB IPSI / PERSILAT 2024
      </div>
      <div class="judge-bottom__center text-emerald-400">
        TRANSMISI TERENKRIPSI KE KETUA PERTANDINGAN
      </div>
      <div class="judge-bottom__right">
        TOTAL SKOR DISEMBUNYIKAN (ATURAN PERSILAT PASAL 28)
      </div>
    </footer>

    <!-- PIN LOGIN ONBOARDING MODAL -->
    <div v-if="showPinModal" class="judge-pin-backdrop">
      <div class="judge-pin-modal">
        <div class="judge-pin-header">
          <div class="judge-pin-icon">
            <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="#10B981" stroke-width="2.2">
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
          <button class="numpad-btn numpad-btn--clear" @click="handlePinClear">
            C
          </button>
          <button class="numpad-btn" @click="handlePinNumber(0)">
            0
          </button>
          <button
            class="numpad-btn numpad-btn--submit"
            :disabled="pinLoading || !isCompletePin(pinInput)"
            @click="handlePinSubmit"
          >
            {{ pinLoading ? '...' : 'OK' }}
          </button>
        </div>

        <div class="judge-pin-footer font-mono">
          <button v-if="authStore.isAuthenticated" class="judge-link-btn text-rose-400" @click="handleJudgeLogout">
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
  width: 100vw;
  height: 100vh;
  background-color: #050811;
  color: #f8fafc;
  display: flex;
  flex-direction: column;
  padding: 12px 18px;
  overflow: hidden;
  user-select: none;
  -webkit-touch-callout: none;
  font-family: var(--font-sans);
}

/* --- Topbar --- */
.judge-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 10px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.judge-topbar__left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.judge-online-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background-color: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.4);
  color: #34d399;
  font-family: var(--font-mono);
  font-size: 0.6875rem;
  font-weight: 800;
  padding: 4px 10px;
  border-radius: var(--radius-full);
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
}
.judge-dot--offline {
  background-color: #f59e0b;
}

.judge-id-group {
  display: flex;
  flex-direction: column;
  line-height: 1.15;
}

.judge-main-title {
  font-size: 1.125rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #ffffff;
}

.judge-sub-title {
  font-size: 0.625rem;
  letter-spacing: 0.08em;
  color: #64748b;
}

.judge-topbar__center {
  display: flex;
  align-items: center;
  gap: 10px;
}

.judge-round-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  background-color: rgba(6, 182, 212, 0.12);
  border: 1px solid rgba(6, 182, 212, 0.35);
  border-radius: var(--radius-sm);
  color: #22d3ee;
  font-size: 0.75rem;
  font-weight: 700;
}

.judge-round-dot {
  width: 7px;
  height: 7px;
  background-color: #06b6d4;
  border-radius: 50%;
  animation: pulse 1.5s infinite;
}

.judge-timer-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  background-color: rgba(15, 23, 42, 0.85);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: var(--radius-sm);
  font-size: 0.9375rem;
  font-weight: 800;
  color: #38bdf8;
}

.judge-topbar__right {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 0.75rem;
}

.judge-chip-btn {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  padding: 4px 10px;
  border-radius: var(--radius-xs);
  color: #cbd5e1;
  font-weight: 700;
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
}
.judge-icon-btn--active {
  color: #34d399;
  border-color: #10b981;
}

.judge-break-note {
  margin-top: 4px;
  font-size: 12px;
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
  padding: 6px 14px;
  border-radius: var(--radius-xs);
  font-family: var(--font-mono);
  font-size: 0.75rem;
  font-weight: 800;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(185, 28, 28, 0.4);
  transition: all var(--transition-fast);
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
}

/* --- Compliance Sub-Header --- */
.judge-compliance-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background-color: rgba(10, 16, 30, 0.7);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: var(--radius-xs);
  margin-top: 8px;
  font-size: 0.6875rem;
  color: #94a3b8;
}

.judge-comp-left, .judge-comp-center, .judge-comp-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.text-peach-200 { color: #fca5a5; }
.text-cyan-400 { color: #22d3ee; }
.judge-bar-divider { opacity: 0.3; }

/* --- Main Split Screen (Two Columns) --- */
.judge-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-top: 10px;
  flex: 1;
}

.judge-corner {
  display: flex;
  flex-direction: column;
  border-radius: var(--radius-md);
  padding: 16px;
  border: 1px solid transparent;
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
  margin-bottom: 12px;
}

.judge-corner-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: var(--radius-xs);
  font-size: 0.6875rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  margin-bottom: 4px;
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
  font-size: 1.5rem;
  font-weight: 800;
  line-height: 1.1;
  color: #ffffff;
  letter-spacing: -0.02em;
}

.judge-athlete-meta {
  font-size: 0.6875rem;
  color: #94a3b8;
  margin-top: 2px;
}

.judge-consensus-indicator {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
}
.judge-consensus-indicator span {
  font-size: 0.5625rem;
  color: #64748b;
  letter-spacing: 0.08em;
}

.judge-dots-row {
  display: flex;
  gap: 5px;
}

.judge-dot-light {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background-color: rgba(255, 255, 255, 0.15);
}
.judge-dot-light--red {
  background-color: #ef4444;
  box-shadow: 0 0 8px rgba(239, 68, 68, 0.8);
}
.judge-dot-light--blue {
  background-color: #3b82f6;
  box-shadow: 0 0 8px rgba(59, 130, 246, 0.8);
}

/* --- 3 Extra-Large Ergonomic Scoring Buttons --- */
.judge-buttons-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex: 1;
}

.judge-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 22px;
  border-radius: var(--radius-md);
  border: 1px solid transparent;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  transition: transform 0.08s ease, filter 0.08s ease, box-shadow 0.08s ease;
  user-select: none;
  min-height: 80px;
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
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.2), 0 6px 18px rgba(185, 28, 28, 0.35);
}
.judge-btn--red.judge-btn--active {
  box-shadow: 0 0 35px rgba(239, 68, 68, 0.9), inset 0 0 15px rgba(255, 255, 255, 0.35);
}

/* Blue Button Styling */
.judge-btn--blue {
  background: linear-gradient(135deg, #1d4ed8 0%, #1e3a8a 100%);
  border-color: rgba(96, 165, 250, 0.6);
  color: #ffffff;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.2), 0 6px 18px rgba(29, 78, 216, 0.35);
}
.judge-btn--blue.judge-btn--active {
  box-shadow: 0 0 35px rgba(96, 165, 250, 0.9), inset 0 0 15px rgba(255, 255, 255, 0.35);
}

.judge-btn__icon-box {
  width: 60px;
  height: 60px;
  border-radius: var(--radius-sm);
  background-color: rgba(0, 0, 0, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.judge-btn__label-group {
  display: flex;
  flex-direction: column;
  flex: 1;
  margin-left: 20px;
  text-align: left;
}

.judge-btn__title {
  font-size: 2.25rem;
  font-weight: 900;
  letter-spacing: -0.02em;
  line-height: 1;
}

.judge-btn__subtitle {
  font-size: 0.6875rem;
  letter-spacing: 0.12em;
  opacity: 0.8;
  margin-top: 4px;
}

.judge-btn__pts {
  font-size: 3.5rem;
  font-weight: 900;
  line-height: 1;
  letter-spacing: -0.04em;
  padding: 4px 14px;
  border-radius: var(--radius-sm);
  background-color: rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.15);
}

/* --- Touch Feedback Strip --- */
.judge-touch-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 10px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  margin-top: 10px;
  font-size: 0.6875rem;
  color: #94a3b8;
}

.judge-touch-last {
  display: flex;
  align-items: center;
  gap: 6px;
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
}

/* --- Bottom Compliance Bar --- */
.judge-bottom-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 8px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  margin-top: 8px;
  font-size: 0.625rem;
  color: #64748b;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(0.85); }
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
}

.judge-pin-modal {
  background-color: #0B1120;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 14px;
  padding: 24px 28px;
  max-width: 380px;
  width: 90%;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.7);
  display: flex;
  flex-direction: column;
  align-items: center;
}

.judge-pin-header {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  margin-bottom: 16px;
}

.judge-pin-icon {
  width: 44px;
  height: 44px;
  background-color: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.3);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.judge-pin-title {
  font-size: 1.125rem;
  font-weight: 800;
  color: #FFFFFF;
  line-height: 1.2;
}

.judge-pin-desc {
  font-size: 0.6875rem;
  color: #94A3B8;
  margin-top: 2px;
}

.judge-pin-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  background-color: #0F172A;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 6px;
  padding: 6px 12px;
  margin-bottom: 14px;
  font-size: 0.6875rem;
}

.judge-pin-meta-item .label {
  color: #64748B;
  margin-right: 6px;
}

.judge-pin-meta-item .val {
  color: #38BDF8;
  font-weight: 700;
}

.judge-pin-display {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-bottom: 14px;
  font-size: 1.75rem;
  color: #475569;
}

.judge-pin-dot--filled {
  color: #10B981;
  text-shadow: 0 0 10px rgba(16, 185, 129, 0.6);
}

.judge-pin-error {
  color: #EF4444;
  font-size: 0.75rem;
  margin-bottom: 12px;
  text-align: center;
}

.judge-numpad {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  width: 100%;
  margin-bottom: 14px;
}

.numpad-btn {
  height: 52px;
  background-color: #162032;
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #F8FAFC;
  font-size: 1.25rem;
  font-weight: 700;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}

.numpad-btn:hover {
  background-color: #1E293B;
  border-color: rgba(255, 255, 255, 0.2);
  transform: translateY(-1px);
}

.numpad-btn:active {
  transform: translateY(1px);
}

.numpad-btn--clear {
  color: #EF4444;
  background-color: rgba(239, 68, 68, 0.08);
  border-color: rgba(239, 68, 68, 0.2);
}

.numpad-btn--submit {
  color: #064E3B;
  background-color: #10B981;
  border-color: #10B981;
}

.numpad-btn--submit:hover {
  background-color: #34D399;
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
  color: #64748B;
  font-size: 0.6875rem;
  cursor: pointer;
  text-decoration: underline;
  transition: color 0.15s ease;
}

.judge-link-btn:hover {
  color: #94A3B8;
}
</style>
