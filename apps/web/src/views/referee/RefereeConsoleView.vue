<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useWebSocket } from '@/composables/useWebSocket'
import { useTimer } from '@/composables/useTimer'
import { useHaptics } from '@/composables/useHaptics'

import { useAuthStore } from '@/stores/auth'
import { matchService } from '@/features/matches/api/matchService'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const arenaId = computed(() => (route.params.arena_id as string) || (route.query.arena_id as string) || authStore.arenaId || 'gelanggang-1')
const compId = computed(() => (route.query.comp_id as string) || authStore.competitionId || '')
const matchId = computed(() => (route.query.match_id as string) || '')


// --- Active Tab / Navigation ---
const activeTab = ref('kontrol-tanding')

// --- Match Context ---
const tournamentTitle = ref('KEJUARAAN NASIONAL PENCAK SILAT 2025')
const matchHeader = ref('DEWASA PUTRA — KELAS C (55-60 KG) • PARTAI 42 SEMIFINAL')
const refereeName = ref('WASIT: H. SUPRAPTO (INA)')
const tableOfficer = ref('PETUGAS MEJA: DEDY KURNIAWAN, S.PD (PENGDA JATENG)')
const rulesetLabel = ref('PERSILAT REVISED 2024')

// Match State
const currentRound = ref(2)
const totalRounds = ref(3)
const roundStatus = ref<'idle' | 'ready' | 'running' | 'paused' | 'break' | 'finished'>('running')

// Athletes
const redAthlete = ref({
  name: 'AHMAD FAUZI',
  team: 'KONTINGEN JAWA TENGAH',
  corner: 'Sudut A • Kelas C Putra',
  score: 14,
  techniques: { pukulan: 4, tendangan: 5, jatuhan: 0, kuncian: 0 },
  penalties: { binaan: 1, teguran1: true, teguran2: false, peringatan1: false, peringatan2: false },
})

const blueAthlete = ref({
  name: 'BAYU PRATAMA',
  team: 'KONTINGEN JAWA TIMUR',
  corner: 'Sudut B • Kelas C Putra',
  score: 11,
  techniques: { pukulan: 6, tendangan: 6, jatuhan: 0, kuncian: 0 },
  penalties: { binaan: 2, teguran1: true, teguran2: true, peringatan1: true, peringatan2: false },
})

// Real-time Event Feed
interface MatchLogEvent {
  id: string
  timeStr: string
  title: string
  detail: string
  type: 'score' | 'penalty' | 'var' | 'doctor' | 'control'
}

const eventLogs = ref<MatchLogEvent[]>([
  {
    id: '1',
    timeStr: '01:31',
    title: 'Poin Sah Sudut Merah (+2 Tendangan)',
    detail: 'Konsensus Juri (J1, J2) diverifikasi otomatis',
    type: 'score',
  },
  {
    id: '2',
    timeStr: '01:15',
    title: 'Teguran 2 (-1) Sudut Biru',
    detail: 'Disahkan Wasit • Tarikan Pakaian',
    type: 'penalty',
  },
  {
    id: '3',
    timeStr: '00:58',
    title: 'Poin Sah Sudut Merah (+1 Pukulan)',
    detail: 'Konsensus Juri (J1, J3) sah',
    type: 'score',
  },
])

// Jury Monitor List
const juryList = ref([
  { seat: 1, name: 'M. RIDWAN', latency: '14ms', lastAction: 'Tendangan Merah (+2)', timeStr: '01:31', corner: 'MERAH' },
  { seat: 2, name: 'S. ANWAR', latency: '18ms', lastAction: 'Tendangan Merah (+2)', timeStr: '01:31', corner: 'MERAH' },
  { seat: 3, name: 'Y. PRANOTO', latency: '12ms', lastAction: 'Pukulan Biru (+1)', timeStr: '01:25', corner: 'BIRU' },
])

// Audio / Gong Feedback
const isGongRinging = ref(false)
const isDoctorCalled = ref(false)
const isVarActive = ref(false)

// Confirmation Modal for Disqualification or Finish
const confirmModal = ref<{ show: boolean; title: string; message: string; action: () => void } | null>(null)

function triggerRefereeLogout() {
  confirmModal.value = {
    show: true,
    title: 'Keluar dari Konsol Wasit?',
    message: 'Apakah Anda yakin ingin mengakhiri sesi meja pengawas wasit dan keluar?',
    action: async () => {
      await authStore.logout()
      router.push('/login')
    },
  }
}

// --- Timer & Haptics ---
const timer = useTimer()
const haptics = useHaptics()

// WebSocket
const token = authStore.token || undefined
const { isConnected, sendMessage, onMessage } = useWebSocket(arenaId.value, token)

// Control Actions
function triggerStart() {
  haptics.trigger(50)
  roundStatus.value = 'running'
  timer.sync(timer.remainingMs.value || 98000, true)
  sendMessage('state_change', { action: 'start' })
  if (compId.value && matchId.value) {
    matchService.startMatch(compId.value, matchId.value).catch(() => {})
  }
  addLog(timer.formattedTime.value, 'Ronde Dimulai / Lanjut', 'Wasit mengaktifkan timer', 'control')
}

function triggerPause() {
  haptics.trigger(50)
  roundStatus.value = 'paused'
  timer.pause()
  sendMessage('state_change', { action: 'pause' })
  if (compId.value && matchId.value) {
    matchService.pauseMatch(compId.value, matchId.value).catch(() => {})
  }
  addLog(timer.formattedTime.value, 'Pertandingan Dijeda (PAUSE)', 'Wasit menghentikan waktu', 'control')
}

function triggerResume() {
  haptics.trigger(50)
  roundStatus.value = 'running'
  timer.sync(timer.remainingMs.value, true)
  sendMessage('state_change', { action: 'resume' })
  if (compId.value && matchId.value) {
    matchService.resumeMatch(compId.value, matchId.value).catch(() => {})
  }
  addLog(timer.formattedTime.value, 'Pertandingan Dilanjutkan', 'Wasit melanjutkan waktu', 'control')
}

function triggerFinish() {
  confirmModal.value = {
    show: true,
    title: 'Selesaikan Pertandingan?',
    message: 'Apakah Anda yakin ingin menyelesaikan partai ini sekarang? Keputusan pemenang akan dihitung otomatis.',
    action: () => {
      haptics.trigger([100, 50, 100])
      roundStatus.value = 'finished'
      timer.pause()
      sendMessage('state_change', { action: 'finish' })
      if (compId.value && matchId.value) {
        matchService.finishMatch(compId.value, matchId.value).catch(() => {})
      }
      addLog(timer.formattedTime.value, 'Pertandingan Selesai', 'Wasit menutup laga resmi', 'control')
      confirmModal.value = null
    },
  }
}

// Adjust Timer
function adjustTimer(deltaSec: number) {
  haptics.trigger(30)
  const newMs = Math.max(0, timer.remainingMs.value + deltaSec * 1000)
  timer.sync(newMs, roundStatus.value === 'running')
}

function resetRoundTimer() {
  haptics.trigger([50, 50])
  timer.sync(120000, false)
  roundStatus.value = 'ready'
}

// Secondary Arena Buttons
function triggerGong() {
  haptics.trigger([80, 80, 80])
  isGongRinging.value = true
  setTimeout(() => { isGongRinging.value = false }, 1200)
  addLog('01:38', 'Gong Arena Dibunyikan', 'Sinyal waktu / peringatan arena', 'control')
}

function triggerDoctor() {
  isDoctorCalled.value = !isDoctorCalled.value
  haptics.trigger(70)
  addLog('01:38', 'Panggilan Tim Medis', 'Wasit memanggil dokter ke gelanggang', 'doctor')
}

function triggerVar() {
  isVarActive.value = !isVarActive.value
  haptics.trigger(70)
  addLog('01:38', 'Tinjauan Video (VAR) Diminta', 'Proses peninjauan tayangan ulang', 'var')
}

// Penalties Handling
function applyPenalty(target: 'red' | 'blue', type: 'teguran1' | 'teguran2' | 'peringatan1' | 'peringatan2' | 'diskualifikasi') {
  haptics.trigger(80)
  const athlete = target === 'red' ? redAthlete.value : blueAthlete.value
  const targetLabel = target === 'red' ? 'Sudut Merah' : 'Sudut Biru'

  if (type === 'diskualifikasi') {
    confirmModal.value = {
      show: true,
      title: `Diskualifikasi ${targetLabel}?`,
      message: `Tindakan ini akan mendiskualifikasi ${athlete.name} dan memenangkan sudut lawan. Lanjutkan?`,
      action: () => {
        sendMessage('state_change', { action: 'cancel' })
        if (compId.value && matchId.value) {
          matchService.cancelMatch(compId.value, matchId.value).catch(() => {})
        }
        addLog(timer.formattedTime.value, `DISKUALIFIKASI ${targetLabel.toUpperCase()}`, `Pesilat ${athlete.name} didiskualifikasi`, 'penalty')
        confirmModal.value = null
      },
    }
    return
  }

  let points = 1
  if (type === 'peringatan1') points = 5
  if (type === 'peringatan2') points = 10

  if (type === 'teguran1') athlete.penalties.teguran1 = !athlete.penalties.teguran1
  if (type === 'teguran2') athlete.penalties.teguran2 = !athlete.penalties.teguran2
  if (type === 'peringatan1') athlete.penalties.peringatan1 = !athlete.penalties.peringatan1
  if (type === 'peringatan2') athlete.penalties.peringatan2 = !athlete.penalties.peringatan2

  sendMessage('penalty', { target, level: points })
  addLog(timer.formattedTime.value, `${type.toUpperCase()} (-${points}) ${targetLabel}`, `Dikenakan pada ${athlete.name}`, 'penalty')
}

function addLog(timeStr: string, title: string, detail: string, type: MatchLogEvent['type']) {
  eventLogs.value.unshift({
    id: String(Date.now()),
    timeStr,
    title,
    detail,
    type,
  })
}

// Global Keyboard Shortcuts
function handleKeyDown(e: KeyboardEvent) {
  // Ignore inside inputs
  if (['INPUT', 'TEXTAREA'].includes((e.target as HTMLElement)?.tagName)) return

  if (e.code === 'Space') {
    e.preventDefault()
    if (roundStatus.value === 'running') triggerPause()
    else triggerStart()
  } else if (e.key === 'p' || e.key === 'P') {
    triggerPause()
  } else if (e.key === 'l' || e.key === 'L') {
    triggerResume()
  } else if (e.key === 'g' || e.key === 'G') {
    triggerGong()
  } else if (e.key === 'Escape') {
    triggerFinish()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
  // Set default initial timer to 01:38 (98000ms) matching Figma 03_referee_console.png
  timer.sync(98000, true)

  // Request state sync from backend orchestrator
  sendMessage('sync_request', {})

  onMessage('sync_response', (payload) => {
    if (payload.round) currentRound.value = payload.round
    if (payload.state) {
      if (payload.state.includes('round')) roundStatus.value = 'running'
      else if (payload.state.includes('break')) roundStatus.value = 'break'
      else if (payload.state === 'paused') roundStatus.value = 'paused'
      else if (payload.state === 'finished') roundStatus.value = 'finished'
    }
    if (payload.timer_remaining_ms !== undefined) {
      timer.sync(payload.timer_remaining_ms, payload.timer_running ?? false)
    }
    if (payload.red_total !== undefined) redAthlete.value.score = payload.red_total
    if (payload.blue_total !== undefined) blueAthlete.value.score = payload.blue_total
  })

  onMessage('state_change', (payload) => {
    if (payload.round) currentRound.value = payload.round
    if (payload.state) {
      if (payload.state.includes('round')) roundStatus.value = 'running'
      else if (payload.state.includes('break')) roundStatus.value = 'break'
      else if (payload.state === 'paused') roundStatus.value = 'paused'
      else if (payload.state === 'finished') roundStatus.value = 'finished'
    }
    if (payload.timer_remaining_ms !== undefined) {
      timer.sync(payload.timer_remaining_ms, roundStatus.value === 'running')
    }
  })

  onMessage('timer_sync', (payload) => {
    if (payload.remaining_ms !== undefined) {
      timer.sync(payload.remaining_ms, payload.running ?? false)
    }
  })

  onMessage('score_confirmed', (payload) => {
    addLog(timer.formattedTime.value, `Konsensus Poin: +${payload.points} ${payload.score_type.toUpperCase()}`, `Sudut ${payload.target.toUpperCase()}`, 'score')
  })

  onMessage('score_update', (payload) => {
    if (payload.red_technical !== undefined) {
      redAthlete.value.score = payload.red_technical - (payload.red_penalties || 0)
    }
    if (payload.blue_technical !== undefined) {
      blueAthlete.value.score = payload.blue_technical - (payload.blue_penalties || 0)
    }
  })

  onMessage('match_result', (payload) => {
    roundStatus.value = 'finished'
    timer.pause()
    const winnerCorner = payload.corner === 'red' ? 'SUDUT MERAH' : 'SUDUT BIRU'
    addLog(timer.formattedTime.value, `Pertandingan Berakhir: ${winnerCorner} MENANG`, payload.result_type || 'Keputusan Poin', 'control')
  })
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
})
</script>

<template>
  <div class="ref-app">
    <!-- SIDEBAR NAVIGATION -->
    <aside class="ref-sidebar">
      <div class="ref-sidebar__brand">
        <div class="ref-logo-badge">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#DC2626" stroke-width="2.5">
            <path d="M18 6 6 18" />
            <path d="m6 6 12 12" />
          </svg>
        </div>
        <div>
          <span class="ref-brand-sub font-mono">PERSILAT ARENA</span>
          <h2 class="ref-brand-title">MEJA PENGAWAS</h2>
        </div>
      </div>

      <div class="ref-arena-status">
        <div class="ref-arena-status__row">
          <span class="font-mono font-bold text-sm">GELANGGANG 1</span>
          <span class="ref-live-chip font-mono">LIVE OPS</span>
        </div>
        <div class="ref-arena-status__state font-mono">
          <span class="ref-pulse-dot" :class="{ 'ref-pulse-dot--connected': isConnected }" />
          {{ isConnected ? 'SISTEM ONLINE' : 'OFFLINE SIMULATOR' }}
        </div>
      </div>

      <!-- Menu Navigation -->
      <nav class="ref-nav font-mono">
        <button
          class="ref-nav__item"
          :class="{ 'ref-nav__item--active': activeTab === 'kontrol-tanding' }"
          @click="activeTab = 'kontrol-tanding'"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect width="18" height="18" x="3" y="4" rx="2" />
            <line x1="16" x2="16" y1="2" y2="6" />
            <line x1="8" x2="8" y1="2" y2="6" />
            <line x1="3" x2="21" y1="10" y2="10" />
          </svg>
          <span>Kontrol Tanding</span>
        </button>

        <button
          class="ref-nav__item"
          :class="{ 'ref-nav__item--active': activeTab === 'verifikasi-juri' }"
          @click="activeTab = 'verifikasi-juri'"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
            <circle cx="9" cy="7" r="4" />
            <path d="M22 21v-2a4 4 0 0 0-3-3.87" />
            <path d="M16 3.13a4 4 0 0 1 0 7.75" />
          </svg>
          <span>Verifikasi Juri 1-5</span>
        </button>

        <button
          class="ref-nav__item"
          :class="{ 'ref-nav__item--active': activeTab === 'sanksi-disiplin' }"
          @click="activeTab = 'sanksi-disiplin'"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10" />
          </svg>
          <span>Sanksi & Disiplin</span>
        </button>

        <button
          class="ref-nav__item"
          :class="{ 'ref-nav__item--active': activeTab === 'log-poin' }"
          @click="activeTab = 'log-poin'"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 3v18h18" />
            <path d="m19 9-5 5-4-4-3 3" />
          </svg>
          <span>Log Poin & Teknik</span>
        </button>

        <button
          class="ref-nav__item"
          :class="{ 'ref-nav__item--active': activeTab === 'bagan-antrean' }"
          @click="$router.push('/design-system')"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect width="7" height="7" x="3" y="3" rx="1" />
            <rect width="7" height="7" x="14" y="3" rx="1" />
            <rect width="7" height="7" x="14" y="14" rx="1" />
            <rect width="7" height="7" x="3" y="14" rx="1" />
          </svg>
          <span>Bagan & Antrean</span>
        </button>
      </nav>

      <!-- Telemetry Box Bottom -->
      <div class="ref-telemetry-box font-mono">
        <div class="ref-telem-row">
          <span>TELEMETRI SYNC</span>
          <strong class="text-emerald-400">100% OK</strong>
        </div>
        <div class="ref-telem-sub text-slate-400">
          &bull; WASIT KETUA &bull; MEJA 1
        </div>
        <div class="ref-telem-stats text-slate-500">
          LATENSI: 4ms &bull; 58.9 FPS
        </div>
      </div>
    </aside>

    <!-- MAIN BODY -->
    <div class="ref-main-content">
      <!-- TOPBAR -->
      <header class="ref-topbar">
        <div class="ref-topbar__left">
          <span class="ref-top-series font-mono">PERSILAT DIGITAL ARENA &bull; MEJA PENGAWAS</span>
          <div class="ref-top-headline">
            <h1 class="ref-top-title">GELANGGANG 1 - MEJA PERTANDINGAN</h1>
            <span class="ref-status-badge font-mono">
              <span class="ref-pulse-dot" /> SISTEM KONTROL AKTIF
            </span>
          </div>
        </div>

        <div class="ref-topbar__right font-mono">
          <div class="ref-referee-info">
            <span class="ref-referee-role">OPERATOR & WASIT RESMI</span>
            <strong class="ref-referee-name">{{ refereeName }}</strong>
          </div>
          <span class="ref-lan-chip">
            <span class="ref-pulse-dot" /> LAN JURI OK
          </span>
          <div class="ref-avatar" title="Keluar dari Konsol Wasit (Logout)" style="cursor: pointer;" @click="triggerRefereeLogout">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
              <circle cx="12" cy="7" r="4" />
            </svg>
          </div>
        </div>
      </header>

      <!-- MATCH BANNER -->
      <section class="ref-banner">
        <div class="ref-banner__left">
          <span class="ref-badge-arena font-mono">GELANGGANG 1</span>
          <div class="ref-banner-titles">
            <span class="ref-tourney-title font-mono">{{ tournamentTitle }}</span>
            <h2 class="ref-match-title">{{ matchHeader }}</h2>
          </div>
        </div>

        <div class="ref-banner__right font-mono">
          <span class="ref-chip-green">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <polyline points="20 6 9 17 4 12" />
            </svg>
            {{ rulesetLabel }}
          </span>
          <span class="ref-chip-round">
            BABAK: {{ currentRound }}/{{ totalRounds }}
          </span>
        </div>
      </section>

      <!-- MATCH SUMMARY & CHRONOMETER BOX (UPPER GRID) -->
      <section class="ref-summary-grid">
        <!-- SUDUT MERAH (LEFT) -->
        <div class="ref-fighter-card ref-fighter-card--red">
          <div class="ref-fighter-card__top font-mono">
            <span class="ref-corner-tag ref-corner-tag--red">SUDUT MERAH</span>
            <span class="ref-fighter-team">{{ redAthlete.team }}</span>
          </div>
          <h3 class="ref-fighter-name">{{ redAthlete.name }}</h3>
          <span class="ref-fighter-class font-mono">{{ redAthlete.corner }}</span>

          <div class="ref-score-row">
            <div class="ref-score-display ref-score-display--red">
              <span class="ref-score-sub font-mono">SKOR SAH</span>
              <div class="ref-score-num font-mono">{{ redAthlete.score }}</div>
              <span class="ref-score-note font-mono text-emerald-400">KONSENSUS JURI (+2 TERAKHIR)</span>
            </div>

            <!-- Penalties Summary Mini -->
            <div class="ref-penalties-mini font-mono">
              <div class="ref-pen-item">
                <span>TEGURAN</span>
                <span class="ref-pen-num ref-pen-num--red">1 (-1)</span>
              </div>
              <div class="ref-pen-item">
                <span>PERINGATAN</span>
                <span class="ref-pen-num">0</span>
              </div>
              <div class="ref-pen-item">
                <span>BINAAN</span>
                <span class="ref-pen-num ref-pen-num--green">1</span>
              </div>
            </div>
          </div>

          <div class="ref-tech-summary font-mono text-slate-400">
            Pukulan: <strong>{{ redAthlete.techniques.pukulan }}</strong> &bull;
            Tendangan: <strong>{{ redAthlete.techniques.tendangan }}</strong> &bull;
            Jatuhan: <strong>{{ redAthlete.techniques.jatuhan }}</strong> &bull;
            Kuncian: <strong>{{ redAthlete.techniques.kuncian }}</strong>
          </div>
        </div>

        <!-- CHRONOMETER RESMI (CENTER) -->
        <div class="ref-chrono-card">
          <div class="ref-chrono-header font-mono">
            <span class="ref-chrono-label">
              <span class="ref-pulse-dot" /> CHRONOMETER RESMI
            </span>
            <span class="ref-chrono-round">BABAK {{ currentRound }} &mdash; {{ roundStatus.toUpperCase() }}</span>
            <span class="ref-chrono-max">2:00 MAX</span>
          </div>

          <!-- Giant Timer Digits -->
          <div class="ref-chrono-time font-mono">
            {{ timer.formattedTime.value }}
            <div class="ref-chrono-dots">
              <span /><span /><span />
            </div>
          </div>

          <!-- Micro-Adjustment Buttons -->
          <div class="ref-chrono-controls font-mono">
            <button class="ref-time-adj-btn" title="Kurangi 10 Detik" @click="adjustTimer(-10)">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
                <path d="M3 3v5h5" />
              </svg>
              -10 DETIK
            </button>
            <button class="ref-time-adj-btn" title="Tambah 10 Detik" @click="adjustTimer(10)">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.85.99 6.74 2.74L21 8" />
                <path d="M21 3v5h-5" />
              </svg>
              +10 DETIK
            </button>
            <button class="ref-time-adj-btn" title="Reset Babak ke 02:00" @click="resetRoundTimer">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
                <path d="M3 3v5h5" />
              </svg>
              RESET BABAK
            </button>
          </div>
        </div>

        <!-- SUDUT BIRU (RIGHT) -->
        <div class="ref-fighter-card ref-fighter-card--blue">
          <div class="ref-fighter-card__top font-mono">
            <span class="ref-fighter-team">{{ blueAthlete.team }}</span>
            <span class="ref-corner-tag ref-corner-tag--blue">SUDUT BIRU</span>
          </div>
          <h3 class="ref-fighter-name">{{ blueAthlete.name }}</h3>
          <span class="ref-fighter-class font-mono">{{ blueAthlete.corner }}</span>

          <div class="ref-score-row">
            <!-- Penalties Summary Mini Blue -->
            <div class="ref-penalties-mini font-mono">
              <div class="ref-pen-item">
                <span>TEGURAN</span>
                <span class="ref-pen-num ref-pen-num--red">2 (-2)</span>
              </div>
              <div class="ref-pen-item">
                <span>PERINGATAN</span>
                <span class="ref-pen-num ref-pen-num--red">1 (-5)</span>
              </div>
              <div class="ref-pen-item">
                <span>BINAAN</span>
                <span class="ref-pen-num ref-pen-num--green">2</span>
              </div>
            </div>

            <div class="ref-score-display ref-score-display--blue">
              <span class="ref-score-sub font-mono">SKOR SAH</span>
              <div class="ref-score-num font-mono">{{ blueAthlete.score }}</div>
              <span class="ref-score-note font-mono text-red-400">TERPOTONG PENALTI -7</span>
            </div>
          </div>

          <div class="ref-tech-summary font-mono text-slate-400">
            Pukulan: <strong>{{ blueAthlete.techniques.pukulan }}</strong> &bull;
            Tendangan: <strong>{{ blueAthlete.techniques.tendangan }}</strong> &bull;
            Jatuhan: <strong>{{ blueAthlete.techniques.jatuhan }}</strong> &bull;
            Kuncian: <strong>{{ blueAthlete.techniques.kuncian }}</strong>
          </div>
        </div>
      </section>

      <!-- COMMAND STRIP KONTROL UTAMA PERTANDINGAN -->
      <section class="ref-command-section">
        <div class="ref-command-header font-mono">
          <div class="ref-command-title">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect width="20" height="14" x="2" y="5" rx="2" />
              <line x1="2" x2="22" y1="10" y2="10" />
            </svg>
            <span>KONTROL UTAMA PERTANDINGAN (COMMAND STRIP)</span>
          </div>
          <div class="ref-operator-mode">
            <span>MODE OPERATOR:</span>
            <strong class="ref-badge-active">KETUA PERTANDINGAN AKTIF</strong>
          </div>
        </div>

        <!-- 4 GIANT MATCH CONTROL BUTTONS -->
        <div class="ref-main-buttons-grid">
          <!-- MULAI RONDE -->
          <button
            class="ref-cmd-btn ref-cmd-btn--start"
            :disabled="roundStatus === 'running'"
            @click="triggerStart"
          >
            <div class="ref-cmd-icon">
              <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                <polygon points="5 3 19 12 5 21 5 3" />
              </svg>
            </div>
            <div class="ref-cmd-text">
              <span class="ref-cmd-label">MULAI RONDE</span>
              <span class="ref-cmd-shortcut font-mono">SHORTCUT [SPACEBAR]</span>
            </div>
          </button>

          <!-- JEDA / PAUSE -->
          <button
            class="ref-cmd-btn ref-cmd-btn--pause"
            :disabled="roundStatus !== 'running'"
            @click="triggerPause"
          >
            <div class="ref-cmd-icon">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                <rect width="4" height="16" x="6" y="4" />
                <rect width="4" height="16" x="14" y="4" />
              </svg>
            </div>
            <div class="ref-cmd-text">
              <span class="ref-cmd-label">JEDA / PAUSE</span>
              <span class="ref-cmd-shortcut font-mono">SHORTCUT [P]</span>
            </div>
          </button>

          <!-- LANJUTKAN -->
          <button
            class="ref-cmd-btn ref-cmd-btn--resume"
            :disabled="roundStatus === 'running'"
            @click="triggerResume"
          >
            <div class="ref-cmd-icon">
              <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                <polygon points="13 19 22 12 13 5 13 19" />
                <polygon points="2 19 11 12 2 5 2 19" />
              </svg>
            </div>
            <div class="ref-cmd-text">
              <span class="ref-cmd-label">LANJUTKAN</span>
              <span class="ref-cmd-shortcut font-mono">SHORTCUT [L]</span>
            </div>
          </button>

          <!-- SELESAIKAN -->
          <button
            class="ref-cmd-btn ref-cmd-btn--finish"
            @click="triggerFinish"
          >
            <div class="ref-cmd-icon">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                <rect width="14" height="14" x="5" y="5" rx="2" />
              </svg>
            </div>
            <div class="ref-cmd-text">
              <span class="ref-cmd-label">SELESAIKAN</span>
              <span class="ref-cmd-shortcut font-mono">SHORTCUT [ESC / CONFIRM]</span>
            </div>
          </button>
        </div>

        <!-- 3 SECONDARY ARENA BUTTONS (GONG, DOKTER, VAR) -->
        <div class="ref-secondary-strip">
          <button
            class="ref-sec-btn"
            :class="{ 'ref-sec-btn--active': isGongRinging }"
            @click="triggerGong"
          >
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9" />
              <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0" />
            </svg>
            <span>BUNYIKAN GONG ARENA</span>
            <span class="ref-sec-shortcut font-mono">[G]</span>
          </button>

          <button
            class="ref-sec-btn"
            :class="{ 'ref-sec-btn--active': isDoctorCalled }"
            @click="triggerDoctor"
          >
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 14V6a2 2 0 0 0-2-2H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h4" />
              <path d="M8 8h8M8 12h5" />
              <path d="M18 18v4M16 20h4" />
            </svg>
            <span>DOKTER PERTANDINGAN</span>
            <span class="ref-sec-shortcut font-mono">[MED]</span>
          </button>

          <button
            class="ref-sec-btn"
            :class="{ 'ref-sec-btn--active': isVarActive }"
            @click="triggerVar"
          >
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect width="18" height="12" x="3" y="6" rx="2" />
              <path d="M8 2h8M12 2v4" />
            </svg>
            <span>TINJAUAN VIDEO (VAR)</span>
            <span class="ref-sec-shortcut font-mono">[VAR]</span>
          </button>
        </div>
      </section>

      <!-- LOWER MANAGEMENT GRID (3 COLUMNS: SANCTIONS RED, SANCTIONS BLUE, JURY & LOGS) -->
      <section class="ref-bottom-grid">
        <!-- COLUMN 1: SANCTIONS RED -->
        <div class="ref-sanction-col">
          <div class="ref-sanction-header font-mono">
            <span class="ref-badge-mini-red">MERAH</span>
            <span class="ref-sanction-name">{{ redAthlete.name }}</span>
            <span class="ref-sanction-total text-red-400">TOTAL SANKSI: -1</span>
          </div>

          <div class="ref-sanction-list font-mono">
            <button
              class="ref-sanc-btn"
              :class="{ 'ref-sanc-btn--active': redAthlete.penalties.teguran1 }"
              @click="applyPenalty('red', 'teguran1')"
            >
              <span>&bull; Teguran 1</span>
              <strong class="ref-sanc-pts">-1 POIN</strong>
            </button>

            <button
              class="ref-sanc-btn"
              :class="{ 'ref-sanc-btn--active': redAthlete.penalties.teguran2 }"
              @click="applyPenalty('red', 'teguran2')"
            >
              <span>&bull; Teguran 2</span>
              <strong class="ref-sanc-pts">-1 POIN</strong>
            </button>

            <button
              class="ref-sanc-btn"
              :class="{ 'ref-sanc-btn--active': redAthlete.penalties.peringatan1 }"
              @click="applyPenalty('red', 'peringatan1')"
            >
              <span>⚠ Peringatan 1</span>
              <strong class="ref-sanc-pts ref-sanc-pts--warn">-5 POIN</strong>
            </button>

            <button
              class="ref-sanc-btn"
              :class="{ 'ref-sanc-btn--active': redAthlete.penalties.peringatan2 }"
              @click="applyPenalty('red', 'peringatan2')"
            >
              <span>⚠ Peringatan 2</span>
              <strong class="ref-sanc-pts ref-sanc-pts--warn">-10 POIN</strong>
            </button>

            <button class="ref-disqualify-btn font-mono" @click="applyPenalty('red', 'diskualifikasi')">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10" />
                <line x1="4.93" y1="4.93" x2="19.07" y2="19.07" />
              </svg>
              <span>DISKUALIFIKASI MERAH</span>
            </button>
          </div>

          <div class="ref-sanc-footer font-mono">
            <button class="ref-undo-sanc-btn" @click="addLog('01:38', 'Sanksi Merah Dibatalkan', 'Wasit menghapus penalti', 'penalty')">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M9 14 4 9l5-5" />
                <path d="M4 9h10.5a5.5 5.5 0 0 1 5.5 5.5v0a5.5 5.5 0 0 1-5.5 5.5H11" />
              </svg>
              BATALKAN SANKSI
            </button>
            <span class="text-slate-500">Catatan pelanggaran</span>
          </div>
        </div>

        <!-- COLUMN 2: SANCTIONS BLUE -->
        <div class="ref-sanction-col">
          <div class="ref-sanction-header font-mono">
            <span class="ref-badge-mini-blue">BIRU</span>
            <span class="ref-sanction-name">{{ blueAthlete.name }}</span>
            <span class="ref-sanction-total text-red-400">TOTAL SANKSI: -7</span>
          </div>

          <div class="ref-sanction-list font-mono">
            <button
              class="ref-sanc-btn"
              :class="{ 'ref-sanc-btn--active': blueAthlete.penalties.teguran1 }"
              @click="applyPenalty('blue', 'teguran1')"
            >
              <span>&bull; Teguran 1 (Aktif)</span>
              <strong class="ref-sanc-pts">-1 POIN</strong>
            </button>

            <button
              class="ref-sanc-btn"
              :class="{ 'ref-sanc-btn--active': blueAthlete.penalties.teguran2 }"
              @click="applyPenalty('blue', 'teguran2')"
            >
              <span>&bull; Teguran 2 (Aktif)</span>
              <strong class="ref-sanc-pts">-1 POIN</strong>
            </button>

            <button
              class="ref-sanc-btn"
              :class="{ 'ref-sanc-btn--active': blueAthlete.penalties.peringatan1 }"
              @click="applyPenalty('blue', 'peringatan1')"
            >
              <span>⚠ Peringatan 1 (Aktif)</span>
              <strong class="ref-sanc-pts ref-sanc-pts--warn">-5 POIN</strong>
            </button>

            <button
              class="ref-sanc-btn"
              :class="{ 'ref-sanc-btn--active': blueAthlete.penalties.peringatan2 }"
              @click="applyPenalty('blue', 'peringatan2')"
            >
              <span>⚠ Peringatan 2</span>
              <strong class="ref-sanc-pts ref-sanc-pts--warn">-10 POIN</strong>
            </button>

            <button class="ref-disqualify-btn font-mono" @click="applyPenalty('blue', 'diskualifikasi')">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10" />
                <line x1="4.93" y1="4.93" x2="19.07" y2="19.07" />
              </svg>
              <span>DISKUALIFIKASI BIRU</span>
            </button>
          </div>

          <div class="ref-sanc-footer font-mono">
            <button class="ref-undo-sanc-btn" @click="addLog('01:38', 'Sanksi Biru Dibatalkan', 'Wasit menghapus penalti', 'penalty')">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M9 14 4 9l5-5" />
                <path d="M4 9h10.5a5.5 5.5 0 0 1 5.5 5.5v0a5.5 5.5 0 0 1-5.5 5.5H11" />
              </svg>
              BATALKAN SANKSI
            </button>
            <span class="text-slate-500">Catatan pelanggaran</span>
          </div>
        </div>

        <!-- COLUMN 3: PANEL JURI & LOGS -->
        <div class="ref-jury-col">
          <!-- Jury Status Header -->
          <div class="ref-jury-header font-mono">
            <div class="ref-jury-title">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#34D399" stroke-width="2">
                <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
                <circle cx="9" cy="7" r="4" />
              </svg>
              <span>PANEL JURI (3 AKTIF)</span>
            </div>
            <span class="ref-badge-lock font-mono">&bull; LOCK 3/3 SAH</span>
          </div>

          <!-- 3 Juries -->
          <div class="ref-juries-list font-mono">
            <div v-for="j in juryList" :key="j.seat" class="ref-jury-card">
              <div class="ref-jury-card__left">
                <span class="ref-jury-seat font-bold">JURI {{ j.seat }} ({{ j.name }})</span>
                <span class="ref-jury-action text-slate-400">{{ j.lastAction }} &bull; {{ j.timeStr }}</span>
              </div>
              <div class="ref-jury-card__right">
                <span class="ref-jury-lat text-emerald-400">&bull; {{ j.latency }}</span>
                <span :class="`ref-badge-mini-${j.corner.toLowerCase()}`">{{ j.corner }}</span>
              </div>
            </div>
          </div>

          <!-- Real-Time Log Header -->
          <div class="ref-log-header font-mono">
            <span>LOG REAL-TIME EVENT</span>
            <span class="ref-sync-live-tag">SYNC LIVE</span>
          </div>

          <!-- Event Log List -->
          <div class="ref-log-list font-mono">
            <div v-for="ev in eventLogs" :key="ev.id" class="ref-log-card">
              <span class="ref-log-time">{{ ev.timeStr }}</span>
              <div class="ref-log-content">
                <strong class="ref-log-title" :class="{ 'text-emerald-400': ev.type === 'score', 'text-red-400': ev.type === 'penalty' }">
                  {{ ev.title }}
                </strong>
                <span class="ref-log-detail text-slate-400">{{ ev.detail }}</span>
              </div>
            </div>
          </div>

          <div class="ref-security-bar font-mono">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
              <path d="M7 11V7a5 5 0 0 1 10 0v4" />
            </svg>
            <span>SISTEM TERENKRIPSI PB IPSI / PERSILAT 2024</span>
          </div>
        </div>
      </section>

      <!-- BOTTOM ARENA HARDWARE STATUS BAR -->
      <footer class="ref-footer-strip font-mono">
        <div class="ref-footer-left">
          <span>&bull; HARDWARE GONG LINK: <strong class="text-emerald-400">SIAP</strong></span>
          <span class="ref-sep">&bull;</span>
          <span>&bull; PAPAN SKOR JUMBOTRON: <strong class="text-emerald-400">SYNC</strong></span>
          <span class="ref-sep">&bull;</span>
          <span>&bull; KAMERA VAR: <strong class="text-cyan-400">4 KANAL 1080P60</strong></span>
        </div>
        <div class="ref-footer-right text-slate-400">
          {{ tableOfficer }}
        </div>
      </footer>
    </div>

    <!-- MODAL CONFIRMATION -->
    <div v-if="confirmModal?.show" class="ref-modal-backdrop">
      <div class="ref-modal-box">
        <h3 class="ref-modal-title">{{ confirmModal.title }}</h3>
        <p class="ref-modal-msg">{{ confirmModal.message }}</p>
        <div class="ref-modal-actions font-mono">
          <button class="ref-modal-cancel" @click="confirmModal = null">Batal</button>
          <button class="ref-modal-confirm" @click="confirmModal.action">Konfirmasi</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ==========================================================================
   REFEREE & MAT OPERATOR CONSOLE (Figma Node 1:513 / 03_referee_console.png)
   ========================================================================== */

.ref-app {
  display: flex;
  width: 100vw;
  min-height: 100vh;
  background-color: #060913;
  color: #f8fafc;
  font-family: var(--font-sans);
  overflow-x: hidden;
  user-select: none;
}

/* --- Sidebar --- */
.ref-sidebar {
  width: 250px;
  background-color: #080d1a;
  border-right: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  flex-direction: column;
  padding: 20px 16px;
  flex-shrink: 0;
}

.ref-sidebar__brand {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
}

.ref-logo-badge {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-xs);
  background-color: rgba(220, 38, 38, 0.15);
  border: 1px solid rgba(220, 38, 38, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
}

.ref-brand-sub {
  font-size: 0.625rem;
  letter-spacing: 0.08em;
  color: #94a3b8;
}

.ref-brand-title {
  font-size: 1rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #ffffff;
}

.ref-arena-status {
  background-color: rgba(15, 23, 42, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: var(--radius-sm);
  padding: 12px;
  margin-bottom: 24px;
}

.ref-arena-status__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.ref-live-chip {
  background-color: rgba(16, 185, 129, 0.2);
  color: #34d399;
  border: 1px solid rgba(16, 185, 129, 0.4);
  font-size: 0.625rem;
  font-weight: 800;
  padding: 2px 6px;
  border-radius: var(--radius-xs);
}

.ref-arena-status__state {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  font-weight: 700;
  color: #34d399;
}

.ref-pulse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: #10b981;
  animation: pulse 1.5s infinite;
}

/* Nav Menu */
.ref-nav {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
}

.ref-nav__item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  color: #94a3b8;
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
  text-align: left;
}
.ref-nav__item:hover {
  background-color: rgba(255, 255, 255, 0.04);
  color: #ffffff;
}
.ref-nav__item--active {
  background-color: #10192e;
  border-color: #1e293b;
  color: #38bdf8;
}

/* Telemetry Box */
.ref-telemetry-box {
  background-color: rgba(10, 16, 30, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-sm);
  padding: 12px;
  font-size: 0.6875rem;
}

.ref-telem-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}
.ref-telem-sub { margin-bottom: 2px; }

/* --- Main Content --- */
.ref-main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 16px 24px;
  overflow-y: auto;
}

/* Topbar */
.ref-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.ref-top-series {
  font-size: 0.625rem;
  letter-spacing: 0.08em;
  color: #94a3b8;
}

.ref-top-headline {
  display: flex;
  align-items: center;
  gap: 14px;
}

.ref-top-title {
  font-size: 1.125rem;
  font-weight: 800;
  letter-spacing: -0.02em;
}

.ref-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background-color: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.35);
  color: #34d399;
  font-size: 0.6875rem;
  font-weight: 700;
  padding: 3px 10px;
  border-radius: var(--radius-full);
}

.ref-topbar__right {
  display: flex;
  align-items: center;
  gap: 14px;
}

.ref-referee-info {
  display: flex;
  flex-direction: column;
  text-align: right;
  line-height: 1.2;
}

.ref-referee-role {
  font-size: 0.5625rem;
  color: #64748b;
  letter-spacing: 0.08em;
}

.ref-referee-name {
  font-size: 0.8125rem;
  color: #e2e8f0;
}

.ref-lan-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background-color: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.3);
  color: #34d399;
  font-size: 0.6875rem;
  padding: 4px 10px;
  border-radius: var(--radius-xs);
}

.ref-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background-color: rgba(255, 255, 255, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Match Banner */
.ref-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 18px;
  background-color: rgba(15, 23, 42, 0.65);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: var(--radius-sm);
  margin-top: 12px;
}

.ref-banner__left {
  display: flex;
  align-items: center;
  gap: 14px;
}

.ref-badge-arena {
  background-color: rgba(220, 38, 38, 0.15);
  border: 1px solid rgba(220, 38, 38, 0.35);
  color: #f87171;
  font-size: 0.75rem;
  font-weight: 800;
  padding: 4px 10px;
  border-radius: var(--radius-xs);
}

.ref-banner-titles {
  display: flex;
  flex-direction: column;
}

.ref-tourney-title {
  font-size: 0.625rem;
  color: #94a3b8;
  letter-spacing: 0.08em;
}

.ref-match-title {
  font-size: 0.9375rem;
  font-weight: 800;
  color: #ffffff;
}

.ref-banner__right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.ref-chip-green {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background-color: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.4);
  color: #34d399;
  font-size: 0.6875rem;
  font-weight: 700;
  padding: 4px 10px;
  border-radius: var(--radius-xs);
}

.ref-chip-round {
  background-color: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #cbd5e1;
  font-size: 0.6875rem;
  padding: 4px 10px;
  border-radius: var(--radius-xs);
}

/* --- Summary Grid (Fighter Red, Chrono, Fighter Blue) --- */
.ref-summary-grid {
  display: grid;
  grid-template-columns: 1fr 320px 1fr;
  gap: 16px;
  margin-top: 12px;
}

.ref-fighter-card {
  padding: 16px;
  border-radius: var(--radius-md);
  border: 1px solid transparent;
  display: flex;
  flex-direction: column;
}

.ref-fighter-card--red {
  background: linear-gradient(180deg, rgba(127, 29, 29, 0.25) 0%, rgba(13, 19, 33, 0.95) 100%);
  border-color: rgba(220, 38, 38, 0.35);
}

.ref-fighter-card--blue {
  background: linear-gradient(180deg, rgba(30, 58, 138, 0.25) 0%, rgba(13, 19, 33, 0.95) 100%);
  border-color: rgba(37, 99, 235, 0.35);
}

.ref-fighter-card__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
  font-size: 0.6875rem;
}

.ref-corner-tag {
  padding: 2px 8px;
  border-radius: var(--radius-xs);
  font-weight: 800;
}
.ref-corner-tag--red { background-color: #dc2626; color: #ffffff; }
.ref-corner-tag--blue { background-color: #2563eb; color: #ffffff; }

.ref-fighter-team { color: #94a3b8; }

.ref-fighter-name {
  font-size: 1.5rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #ffffff;
  line-height: 1.1;
}

.ref-fighter-class {
  font-size: 0.6875rem;
  color: #94a3b8;
  margin-bottom: 12px;
}

.ref-score-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 12px;
}

.ref-score-display {
  flex: 1;
  background-color: rgba(6, 11, 23, 0.7);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: var(--radius-sm);
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
}

.ref-score-sub { font-size: 0.625rem; color: #64748b; }
.ref-score-num { font-size: 3.25rem; font-weight: 900; line-height: 1; }
.ref-score-display--red .ref-score-num { color: #ef4444; }
.ref-score-display--blue .ref-score-num { color: #3b82f6; }
.ref-score-note { font-size: 0.5625rem; margin-top: 4px; }

.ref-penalties-mini {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 0.6875rem;
  color: #94a3b8;
}

.ref-pen-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.ref-pen-num {
  padding: 2px 6px;
  border-radius: var(--radius-xs);
  background-color: rgba(255, 255, 255, 0.06);
  color: #ffffff;
  font-weight: 700;
}
.ref-pen-num--red { background-color: #b91c1c; color: #ffffff; }
.ref-pen-num--green { background-color: #047857; color: #ffffff; }

.ref-tech-summary {
  font-size: 0.6875rem;
  padding-top: 8px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

/* Chronometer Card */
.ref-chrono-card {
  background: radial-gradient(circle at center, rgba(16, 185, 129, 0.1) 0%, rgba(10, 16, 30, 0.8) 100%);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-md);
  padding: 14px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: space-between;
}

.ref-chrono-header {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.625rem;
  color: #94a3b8;
}

.ref-chrono-label {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #34d399;
  font-weight: 700;
}

.ref-chrono-round {
  background-color: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.35);
  color: #34d399;
  padding: 2px 8px;
  border-radius: var(--radius-xs);
}

.ref-chrono-time {
  font-size: 4rem;
  font-weight: 900;
  line-height: 1;
  color: #34d399;
  text-shadow: 0 0 25px rgba(16, 185, 129, 0.5);
  display: flex;
  flex-direction: column;
  align-items: center;
  margin: 8px 0;
}

.ref-chrono-dots {
  display: flex;
  gap: 6px;
  margin-top: 4px;
}
.ref-chrono-dots span {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background-color: #10b981;
  opacity: 0.7;
}

.ref-chrono-controls {
  display: flex;
  gap: 6px;
  width: 100%;
}

.ref-time-adj-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  background-color: rgba(15, 23, 42, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: var(--radius-xs);
  color: #cbd5e1;
  padding: 6px 2px;
  font-size: 0.625rem;
  font-weight: 700;
  cursor: pointer;
  transition: all var(--transition-fast);
}
.ref-time-adj-btn:hover {
  background-color: rgba(255, 255, 255, 0.1);
  color: #ffffff;
}

/* --- Command Strip Section --- */
.ref-command-section {
  background-color: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: var(--radius-md);
  padding: 16px;
  margin-top: 14px;
}

.ref-command-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  font-size: 0.75rem;
}

.ref-command-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 800;
  color: #e2e8f0;
}

.ref-operator-mode {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #64748b;
}

.ref-badge-active {
  background-color: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.35);
  color: #34d399;
  padding: 3px 8px;
  border-radius: var(--radius-xs);
}

/* 4 Giant Buttons Grid */
.ref-main-buttons-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.ref-cmd-btn {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 20px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  cursor: pointer;
  color: #ffffff;
  transition: all 0.1s ease;
  user-select: none;
  text-align: left;
}

.ref-cmd-btn:active {
  transform: scale(0.97);
}
.ref-cmd-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
  transform: none;
}

.ref-cmd-btn--start {
  background: linear-gradient(135deg, #059669 0%, #047857 100%);
  border-color: rgba(52, 211, 153, 0.5);
  box-shadow: 0 4px 14px rgba(5, 150, 105, 0.35);
}

.ref-cmd-btn--pause {
  background: linear-gradient(135deg, #334155 0%, #1e293b 100%);
  border-color: rgba(255, 255, 255, 0.15);
}

.ref-cmd-btn--resume {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
  border-color: rgba(96, 165, 250, 0.5);
  box-shadow: 0 4px 14px rgba(37, 99, 235, 0.35);
}

.ref-cmd-btn--finish {
  background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
  border-color: rgba(248, 113, 113, 0.5);
  box-shadow: 0 4px 14px rgba(220, 38, 38, 0.35);
}

.ref-cmd-text {
  display: flex;
  flex-direction: column;
}

.ref-cmd-label {
  font-size: 1.25rem;
  font-weight: 900;
  letter-spacing: -0.02em;
}

.ref-cmd-shortcut {
  font-size: 0.625rem;
  letter-spacing: 0.08em;
  opacity: 0.75;
}

/* 3 Secondary Strip */
.ref-secondary-strip {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-top: 10px;
}

.ref-sec-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  background-color: rgba(15, 23, 42, 0.85);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-xs);
  padding: 10px 16px;
  color: #cbd5e1;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.ref-sec-btn:hover {
  background-color: rgba(255, 255, 255, 0.08);
  color: #ffffff;
}

.ref-sec-btn--active {
  background-color: rgba(220, 38, 38, 0.25) !important;
  border-color: #dc2626 !important;
  color: #f87171 !important;
  box-shadow: 0 0 15px rgba(220, 38, 38, 0.4);
}

.ref-sec-shortcut {
  font-size: 0.6875rem;
  color: #64748b;
}

/* --- Lower Management Grid (3 Columns) --- */
.ref-bottom-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1.35fr;
  gap: 16px;
  margin-top: 14px;
  flex: 1;
}

.ref-sanction-col, .ref-jury-col {
  background-color: rgba(15, 23, 42, 0.65);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: var(--radius-md);
  padding: 14px;
  display: flex;
  flex-direction: column;
}

.ref-sanction-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  font-size: 0.75rem;
}

.ref-badge-mini-red {
  background-color: #dc2626;
  color: #ffffff;
  padding: 2px 6px;
  border-radius: var(--radius-xs);
  font-weight: 800;
}
.ref-badge-mini-blue {
  background-color: #2563eb;
  color: #ffffff;
  padding: 2px 6px;
  border-radius: var(--radius-xs);
  font-weight: 800;
}

.ref-sanction-name { font-weight: 700; color: #ffffff; }

.ref-sanction-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
}

.ref-sanc-btn {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background-color: rgba(10, 16, 30, 0.65);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: var(--radius-xs);
  color: #cbd5e1;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.ref-sanc-btn:hover {
  background-color: rgba(255, 255, 255, 0.06);
}

.ref-sanc-btn--active {
  background-color: rgba(220, 38, 38, 0.2);
  border-color: rgba(220, 38, 38, 0.5);
  color: #f87171;
}

.ref-sanc-pts {
  color: #f87171;
  background-color: rgba(0, 0, 0, 0.35);
  padding: 2px 6px;
  border-radius: var(--radius-xs);
}

.ref-sanc-pts--warn {
  color: #fbbf24;
}

.ref-disqualify-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: linear-gradient(135deg, rgba(185, 28, 28, 0.8) 0%, rgba(127, 29, 29, 0.9) 100%);
  border: 1px solid rgba(239, 68, 68, 0.5);
  color: #ffffff;
  padding: 12px;
  border-radius: var(--radius-xs);
  font-weight: 800;
  font-size: 0.8125rem;
  cursor: pointer;
  margin-top: 4px;
}
.ref-disqualify-btn:hover {
  filter: brightness(1.15);
}

.ref-sanc-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 10px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  margin-top: 12px;
  font-size: 0.6875rem;
}

.ref-undo-sanc-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
}
.ref-undo-sanc-btn:hover { color: #ffffff; }

/* Column 3: Jury & Logs */
.ref-jury-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  font-size: 0.75rem;
}

.ref-jury-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 800;
  color: #e2e8f0;
}

.ref-badge-lock {
  background-color: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.35);
  color: #34d399;
  padding: 2px 8px;
  border-radius: var(--radius-xs);
  font-size: 0.625rem;
  font-weight: 800;
}

.ref-juries-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
}

.ref-jury-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background-color: rgba(10, 16, 30, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-xs);
  font-size: 0.6875rem;
}

.ref-jury-card__left {
  display: flex;
  flex-direction: column;
}

.ref-jury-card__right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ref-log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 0.6875rem;
  color: #94a3b8;
}

.ref-sync-live-tag {
  color: #34d399;
  font-size: 0.625rem;
  letter-spacing: 0.08em;
}

.ref-log-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 140px;
  overflow-y: auto;
  margin-bottom: 10px;
}

.ref-log-card {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 6px 10px;
  background-color: rgba(0, 0, 0, 0.3);
  border-radius: var(--radius-xs);
  font-size: 0.6875rem;
}

.ref-log-time {
  color: #38bdf8;
  font-weight: 700;
}

.ref-log-content {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
}

.ref-security-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding-top: 8px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  margin-top: auto;
  font-size: 0.625rem;
  color: #64748b;
}

/* --- Footer Strip --- */
.ref-footer-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  margin-top: 14px;
  font-size: 0.6875rem;
}

.ref-footer-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.ref-sep { opacity: 0.3; }

/* Modal */
.ref-modal-backdrop {
  position: fixed;
  inset: 0;
  background-color: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(6px);
}

.ref-modal-box {
  background-color: #0f172a;
  border: 1px solid #334155;
  border-radius: var(--radius-md);
  padding: 24px;
  max-width: 440px;
  width: 90%;
  box-shadow: var(--shadow-lg);
}

.ref-modal-title {
  font-size: 1.25rem;
  font-weight: 800;
  color: #ffffff;
  margin-bottom: 8px;
}

.ref-modal-msg {
  font-size: 0.875rem;
  color: #cbd5e1;
  margin-bottom: 20px;
  line-height: 1.5;
}

.ref-modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.ref-modal-cancel {
  padding: 8px 16px;
  background-color: #1e293b;
  border: 1px solid #334155;
  color: #cbd5e1;
  border-radius: var(--radius-xs);
  cursor: pointer;
  font-size: 0.8125rem;
}

.ref-modal-confirm {
  padding: 8px 16px;
  background-color: #dc2626;
  border: none;
  color: #ffffff;
  border-radius: var(--radius-xs);
  font-weight: 700;
  cursor: pointer;
  font-size: 0.8125rem;
}
</style>
