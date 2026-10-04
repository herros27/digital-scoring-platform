<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useWebSocket } from '@/composables/useWebSocket'
import { useTimer } from '@/composables/useTimer'
import { api } from '@/lib/api'

const route = useRoute()
const arenaId = computed(() => (route.params.arena_id as string) || 'gelanggang-1')

// --- Match State ---
const competitionTitle = ref('KEJUARAAN NASIONAL PENCAK SILAT 2025')
const tournamentSeries = ref('PENCAK SILAT WORLD CHAMPIONSHIPS')
const arenaName = ref('GELANGGANG 1')
const categoryName = ref('DEWASA PUTRA — KELAS C (55–60 KG)')
const matchNumber = ref('MATCH #M-2025-04')
const rulesetLabel = ref('PERSILAT RULES 2024')

// Round & State
const currentRound = ref(2)
const matchStatus = ref<'idle' | 'ready' | 'running' | 'break' | 'paused' | 'finished'>('running')

// Athletes
const redAthlete = ref({
  name: 'AHMAD FAUZI',
  team: 'KONTINGEN JAWA TENGAH',
  idTag: 'ID: INA-204',
})

const blueAthlete = ref({
  name: 'BAYU PRATAMA',
  team: 'KONTINGEN JAWA TIMUR',
  idTag: 'ID: INA-188',
})

// Scores per round
const roundScores = ref([
  { round: 1, red: 4, blue: 5, redPen: 0, bluePen: 0 },
  { round: 2, red: 8, blue: 4, redPen: 1, bluePen: 2 },
  { round: 3, red: 2, blue: 2, redPen: 0, bluePen: 1 },
])

// Penalties
const redPenalties = ref({ binaan: 0, teguran: 1, peringatan: 0 })
const bluePenalties = ref({ binaan: 1, teguran: 2, peringatan: 1 })

// Totals
const redTotal = computed(() => {
  return roundScores.value.reduce((acc, r) => acc + r.red, 0)
})

const blueTotal = computed(() => {
  return roundScores.value.reduce((acc, r) => acc + r.blue, 0)
})

// Jury Consensus status banner
const consensusStatus = computed(() => {
  if (redTotal.value > blueTotal.value) return 'MERAH MEMIMPIN (3/5)'
  if (blueTotal.value > redTotal.value) return 'BIRU MEMIMPIN (3/5)'
  return 'SKOR IMBANG (5/5)'
})

// Floating Score Animation Toasts
interface ScoreToast {
  id: number
  side: 'red' | 'blue'
  text: string
}
const activeToasts = ref<ScoreToast[]>([])
let toastCounter = 0

function triggerScoreToast(side: 'red' | 'blue', text: string) {
  const id = ++toastCounter
  activeToasts.value.push({ id, side, text })
  setTimeout(() => {
    activeToasts.value = activeToasts.value.filter(t => t.id !== id)
  }, 1600)
}

// Winner Announcement Modal
const showWinnerModal = ref(false)
const winnerData = ref<{ side: 'red' | 'blue'; name: string; resultType: string } | null>(null)

// --- Timer ---
const timer = useTimer()

// Fullscreen toggle
const isFullscreen = ref(false)
function toggleFullscreen() {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen().catch(() => {})
    isFullscreen.value = true
  } else {
    document.exitFullscreen().catch(() => {})
    isFullscreen.value = false
  }
}

// --- Real-Time WebSocket Connection ---
const { isConnected, onMessage } = useWebSocket(arenaId.value)

onMounted(async () => {
  // Initialize timer to 01:41 (101 seconds) as depicted in Figma 01_arena_scoreboard.png
  timer.sync(101000, true)

  try {
    const snapshot = await api.get<any>(`/api/v1/live/scoreboard/${arenaId.value}`)
    if (snapshot && typeof snapshot === 'object' && snapshot.status !== 'no active match') {
      if (snapshot.round) currentRound.value = snapshot.round
      if (snapshot.red_athlete) redAthlete.value.name = snapshot.red_athlete
      if (snapshot.blue_athlete) blueAthlete.value.name = snapshot.blue_athlete
    }
  } catch {
    // Keep local baseline state
  }

  // Listen to WebSocket broadcasts
  onMessage('sync_response', (payload) => {
    if (payload.round) currentRound.value = payload.round
    if (payload.timer_remaining_ms !== undefined) {
      timer.sync(payload.timer_remaining_ms, payload.timer_running)
    }
  })

  onMessage('state_change', (payload) => {
    if (payload.round) currentRound.value = payload.round
    if (payload.state) {
      if (payload.state.includes('round')) matchStatus.value = 'running'
      else if (payload.state.includes('break')) matchStatus.value = 'break'
      else if (payload.state === 'paused') matchStatus.value = 'paused'
      else if (payload.state === 'finished') matchStatus.value = 'finished'
    }
    if (payload.timer_remaining_ms !== undefined) {
      timer.sync(payload.timer_remaining_ms, matchStatus.value === 'running')
    }
  })

  onMessage('timer_sync', (payload) => {
    if (payload.remaining_ms !== undefined) {
      timer.sync(payload.remaining_ms, payload.running)
    }
  })

  onMessage('score_confirmed', (payload) => {
    const pointText = `+${payload.points} ${payload.score_type.toUpperCase()}`
    triggerScoreToast(payload.target === 'red' ? 'red' : 'blue', pointText)
  })

  onMessage('score_update', (payload) => {
    const idx = roundScores.value.findIndex(r => r.round === payload.round)
    if (idx !== -1) {
      roundScores.value[idx].red = payload.red_technical
      roundScores.value[idx].blue = payload.blue_technical
    }
  })

  onMessage('match_result', (payload) => {
    const side = payload.corner === 'red' ? 'red' : 'blue'
    const name = side === 'red' ? redAthlete.value.name : blueAthlete.value.name
    winnerData.value = {
      side,
      name,
      resultType: payload.result_type || 'MENANG ANGKA (POINT WIN)',
    }
    showWinnerModal.value = true
  })
})
</script>

<template>
  <div class="sb-screen">
    <!-- TOP BROADCAST NAVBAR -->
    <header class="sb-topbar">
      <div class="sb-brand">
        <div class="sb-logo-box">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#DC2626" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M18 6 6 18" />
            <path d="m6 6 12 12" />
          </svg>
        </div>
        <span class="sb-brand-name">PERSILAT</span>
        <span class="sb-pill-tag">OFFICIAL ARENA SYSTEM</span>
      </div>

      <div class="sb-top-status">
        <span class="sb-onair-chip" :class="{ 'sb-onair-chip--offline': !isConnected }">
          <span class="sb-pulse-dot" :class="{ 'sb-pulse-dot--offline': !isConnected }" />
          {{ arenaName }} &bull; {{ isConnected ? 'ON-AIR' : 'OFFICIAL LIVE' }}
        </span>
        <span class="sb-feed-chip">&bull; 4K 60FPS FEED</span>
        <button class="sb-fs-btn" title="Toggle Fullscreen" @click="toggleFullscreen">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3" />
          </svg>
        </button>
      </div>
    </header>

    <!-- TOURNAMENT BANNER STRIP -->
    <section class="sb-tournament-banner">
      <div class="sb-banner-left">
        <div class="sb-accent-bar" />
        <div class="sb-banner-titles">
          <span class="sb-tourney-series font-mono">{{ tournamentSeries }}</span>
          <h1 class="sb-tourney-title">{{ competitionTitle }}</h1>
        </div>
      </div>

      <div class="sb-banner-center">
        <div class="sb-arena-badge font-mono">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#34D399" stroke-width="2">
            <rect width="18" height="18" x="3" y="3" rx="2" />
            <path d="M3 9h18M9 21V9" />
          </svg>
          <span>{{ arenaName }}</span>
        </div>
      </div>

      <div class="sb-banner-right">
        <span class="sb-cat-label font-mono">KATEGORI TANDING</span>
        <span class="sb-cat-value font-mono">{{ categoryName }}</span>
      </div>
    </section>

    <!-- CENTER TIMER & ROUND CONTROL STRIP -->
    <section class="sb-timer-strip">
      <!-- Round Badges (Left) -->
      <div class="sb-round-chips">
        <div class="sb-round-chip" :class="{ 'sb-round-chip--done': currentRound > 1 }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="20 6 9 17 4 12" />
          </svg>
          <div class="sb-round-chip__text">
            <span>BABAK 1</span>
            <strong>SELESAI</strong>
          </div>
        </div>

        <div class="sb-round-chip sb-round-chip--active">
          <span class="sb-round-chip__dot" />
          <div class="sb-round-chip__text">
            <span>BABAK 2</span>
            <strong>AKTIF</strong>
          </div>
        </div>

        <div class="sb-round-chip" :class="{ 'sb-round-chip--dim': currentRound < 3 }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <polyline points="12 6 12 12 16 14" />
          </svg>
          <div class="sb-round-chip__text">
            <span>BABAK 3</span>
            <strong>MENUNGGU</strong>
          </div>
        </div>
      </div>

      <!-- Giant Digital Clock (Center) -->
      <div class="sb-clock-box">
        <div class="sb-giant-timer font-mono">
          {{ timer.formattedTime.value }}
        </div>
        <div class="sb-clock-caption font-mono">
          <span class="sb-pulse-dot" />
          WAKTU BERTANDING
        </div>
      </div>

      <!-- Consensus Badge (Right) -->
      <div class="sb-consensus-box">
        <div class="sb-consensus-chip">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#F87171" stroke-width="2">
            <path d="m14 12-8.5 8.5a2.12 2.12 0 1 1-3-3L11 9" />
            <path d="M15 13 9 7l4-4 6 6z" />
          </svg>
          <div class="sb-consensus-text font-mono">
            <span>KONSENSUS JURI</span>
            <strong>{{ consensusStatus }}</strong>
          </div>
        </div>
      </div>
    </section>

    <!-- MAIN BODY: RED CORNER VS BLUE CORNER -->
    <main class="sb-main-grid">
      <!-- SUDUT MERAH (RED CORNER) -->
      <section class="sb-corner sb-corner--red">
        <div class="sb-corner__header">
          <div class="sb-corner__title font-mono">
            <span class="sb-corner-dot sb-corner-dot--red" />
            SUDUT MERAH
          </div>
          <div class="sb-contingent-tag font-mono">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z" />
              <circle cx="12" cy="10" r="3" />
            </svg>
            {{ redAthlete.team }}
          </div>
        </div>

        <div class="sb-athlete-row">
          <div class="sb-athlete-meta">
            <span class="sb-athlete-role font-mono">PESILAT</span>
            <h2 class="sb-athlete-name">{{ redAthlete.name }}</h2>
          </div>
          <span class="sb-athlete-id font-mono">{{ redAthlete.idTag }}</span>
        </div>

        <!-- Giant Score Box Red -->
        <div class="sb-score-box sb-score-box--red">
          <!-- Floating Point Toast -->
          <transition-group name="toast-pop">
            <div
              v-for="toast in activeToasts.filter(t => t.side === 'red')"
              :key="toast.id"
              class="sb-floating-toast sb-floating-toast--red font-mono"
            >
              {{ toast.text }}
            </div>
          </transition-group>

          <div class="sb-giant-score font-mono">{{ redTotal }}</div>
          <div class="sb-score-label font-mono">
            <strong>PTS</strong>
            <span>TOTAL SKOR</span>
          </div>
        </div>

        <!-- Round Score Breakdown (Babak 1, 2, 3) -->
        <div class="sb-breakdown-row">
          <div
            v-for="r in roundScores"
            :key="r.round"
            class="sb-round-card"
            :class="{ 'sb-round-card--active-red': r.round === currentRound }"
          >
            <div class="sb-round-card__label font-mono">
              BABAK {{ r.round }}
              <span v-if="r.round === currentRound" class="sb-round-card__dot sb-round-card__dot--red" />
            </div>
            <div class="sb-round-card__val font-mono">{{ r.red }}</div>
          </div>
        </div>

        <!-- Hukuman / Penalties Strip -->
        <footer class="sb-penalties-bar">
          <div class="sb-penalties-label font-mono">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10" />
            </svg>
            <span>HUKUMAN :</span>
          </div>
          <div class="sb-penalties-chips font-mono">
            <span class="sb-pen-chip">BINAAN: <strong>{{ redPenalties.binaan }}</strong></span>
            <span class="sb-pen-chip sb-pen-chip--alert-red">
              TEGURAN: <strong>{{ redPenalties.teguran }} (-{{ redPenalties.teguran }})</strong>
            </span>
            <span class="sb-pen-chip">PERINGATAN: <strong>{{ redPenalties.peringatan }}</strong></span>
          </div>
        </footer>
      </section>

      <!-- SUDUT BIRU (BLUE CORNER) -->
      <section class="sb-corner sb-corner--blue">
        <div class="sb-corner__header">
          <div class="sb-corner__title font-mono">
            <span class="sb-corner-dot sb-corner-dot--blue" />
            SUDUT BIRU
          </div>
          <div class="sb-contingent-tag font-mono">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z" />
              <circle cx="12" cy="10" r="3" />
            </svg>
            {{ blueAthlete.team }}
          </div>
        </div>

        <div class="sb-athlete-row">
          <div class="sb-athlete-meta">
            <span class="sb-athlete-role font-mono">PESILAT</span>
            <h2 class="sb-athlete-name">{{ blueAthlete.name }}</h2>
          </div>
          <span class="sb-athlete-id font-mono">{{ blueAthlete.idTag }}</span>
        </div>

        <!-- Giant Score Box Blue -->
        <div class="sb-score-box sb-score-box--blue">
          <!-- Floating Point Toast -->
          <transition-group name="toast-pop">
            <div
              v-for="toast in activeToasts.filter(t => t.side === 'blue')"
              :key="toast.id"
              class="sb-floating-toast sb-floating-toast--blue font-mono"
            >
              {{ toast.text }}
            </div>
          </transition-group>

          <div class="sb-giant-score font-mono">{{ blueTotal }}</div>
          <div class="sb-score-label font-mono">
            <strong>PTS</strong>
            <span>TOTAL SKOR</span>
          </div>
        </div>

        <!-- Round Score Breakdown (Babak 1, 2, 3) -->
        <div class="sb-breakdown-row">
          <div
            v-for="r in roundScores"
            :key="r.round"
            class="sb-round-card"
            :class="{ 'sb-round-card--active-blue': r.round === currentRound }"
          >
            <div class="sb-round-card__label font-mono">
              BABAK {{ r.round }}
              <span v-if="r.round === currentRound" class="sb-round-card__dot sb-round-card__dot--blue" />
            </div>
            <div class="sb-round-card__val font-mono">{{ r.blue }}</div>
          </div>
        </div>

        <!-- Hukuman / Penalties Strip -->
        <footer class="sb-penalties-bar">
          <div class="sb-penalties-label font-mono">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10" />
            </svg>
            <span>HUKUMAN :</span>
          </div>
          <div class="sb-penalties-chips font-mono">
            <span class="sb-pen-chip">BINAAN: <strong>{{ bluePenalties.binaan }}</strong></span>
            <span class="sb-pen-chip sb-pen-chip--alert-blue">
              TEGURAN: <strong>{{ bluePenalties.teguran }} (-{{ bluePenalties.teguran }})</strong>
            </span>
            <span class="sb-pen-chip sb-pen-chip--alert-red">
              PERINGATAN: <strong>{{ bluePenalties.peringatan }} (-5)</strong>
            </span>
          </div>
        </footer>
      </section>
    </main>

    <!-- BROADCAST TICKER FOOTER -->
    <footer class="sb-bottom-bar font-mono">
      <div class="sb-bottom-left">
        <span class="sb-live-indicator">((&bull;))</span>
        <span>PENCAK SILAT DIGITAL SCORING SYSTEM</span>
        <span class="sb-bar-sep">&bull;</span>
        <span class="text-emerald-400">OFFICIAL LIVE TIMING</span>
        <span class="sb-bar-sep">&bull;</span>
        <span>{{ arenaName.toUpperCase() }}</span>
      </div>
      <div class="sb-bottom-right">
        <span>&copy; {{ rulesetLabel }}</span>
        <span class="sb-bar-sep">&bull;</span>
        <span>{{ matchNumber }}</span>
      </div>
    </footer>

    <!-- WINNER CELEBRATION MODAL OVERLAY -->
    <transition name="fade">
      <div v-if="showWinnerModal && winnerData" class="sb-winner-overlay" :class="`sb-winner-overlay--${winnerData.side}`">
        <div class="sb-winner-card">
          <div class="sb-winner-badge font-mono">
            PEMENANG PERTANDINGAN
          </div>
          <h2 class="sb-winner-name">{{ winnerData.name }}</h2>
          <div class="sb-winner-corner font-mono" :class="`text-${winnerData.side}-400`">
            SUDUT {{ winnerData.side.toUpperCase() }}
          </div>
          <div class="sb-winner-type font-mono">
            {{ winnerData.resultType }}
          </div>
          <div class="sb-winner-scores font-mono">
            <span class="text-red-400 font-bold">{{ redTotal }}</span>
            <span class="text-slate-500">VS</span>
            <span class="text-blue-400 font-bold">{{ blueTotal }}</span>
          </div>
          <button class="sb-winner-close font-mono" @click="showWinnerModal = false">
            Tutup Tampilan Kemenangan
          </button>
        </div>
      </div>
    </transition>
  </div>
</template>

<style scoped>
/* ==========================================================================
   STADIUM BROADCAST SCOREBOARD (Figma Node 1:2 / 01_arena_scoreboard.png)
   ========================================================================== */

.sb-screen {
  width: 100vw;
  min-height: 100vh;
  background-color: #060913;
  color: #f8fafc;
  display: flex;
  flex-direction: column;
  padding: 18px 28px;
  overflow-x: hidden;
  user-select: none;
  font-family: var(--font-sans);
}

/* --- Topbar --- */
.sb-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 14px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.sb-brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.sb-logo-box {
  width: 32px;
  height: 32px;
  background-color: rgba(220, 38, 38, 0.15);
  border: 1px solid rgba(220, 38, 38, 0.35);
  border-radius: var(--radius-xs);
  display: flex;
  align-items: center;
  justify-content: center;
}

.sb-brand-name {
  font-size: 1.125rem;
  font-weight: 800;
  letter-spacing: -0.02em;
}

.sb-pill-tag {
  font-family: var(--font-mono);
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  padding: 3px 8px;
  border-radius: var(--radius-xs);
  background-color: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #94a3b8;
}

.sb-top-status {
  display: flex;
  align-items: center;
  gap: 12px;
}

.sb-onair-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background-color: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.35);
  color: #34d399;
  font-family: var(--font-mono);
  font-size: 0.75rem;
  font-weight: 700;
  padding: 4px 12px;
  border-radius: var(--radius-full);
}

.sb-pulse-dot {
  width: 7px;
  height: 7px;
  background-color: #10b981;
  border-radius: 50%;
  animation: pulse 1.6s infinite;
}

.sb-feed-chip {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  color: #64748b;
}

.sb-fs-btn {
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #94a3b8;
  border-radius: var(--radius-xs);
  padding: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}
.sb-fs-btn:hover {
  color: #ffffff;
  border-color: #ffffff;
}

/* --- Tournament Banner Strip --- */
.sb-tournament-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background-color: rgba(15, 23, 42, 0.55);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: var(--radius-md);
  margin-top: 14px;
}

.sb-banner-left {
  display: flex;
  align-items: center;
  gap: 14px;
}

.sb-accent-bar {
  width: 4px;
  height: 38px;
  background: linear-gradient(180deg, #f97316 0%, #ef4444 100%);
  border-radius: 2px;
}

.sb-tourney-series {
  font-size: 0.6875rem;
  letter-spacing: 0.1em;
  color: #94a3b8;
  text-transform: uppercase;
}

.sb-tourney-title {
  font-size: 1.25rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #ffffff;
}

.sb-arena-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background-color: rgba(15, 23, 42, 0.9);
  border: 1px solid rgba(255, 255, 255, 0.12);
  padding: 6px 14px;
  border-radius: var(--radius-sm);
  font-size: 0.8125rem;
  font-weight: 700;
  color: #e2e8f0;
}

.sb-banner-right {
  text-align: right;
  display: flex;
  flex-direction: column;
}

.sb-cat-label {
  font-size: 0.6875rem;
  letter-spacing: 0.1em;
  color: #94a3b8;
}

.sb-cat-value {
  font-size: 0.9375rem;
  font-weight: 700;
  color: #38bdf8;
}

/* --- Center Timer Strip --- */
.sb-timer-strip {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  padding: 16px 20px;
  background: radial-gradient(circle at center, rgba(16, 185, 129, 0.08) 0%, rgba(10, 16, 30, 0.4) 100%);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-md);
  margin-top: 14px;
}

.sb-round-chips {
  display: flex;
  gap: 10px;
}

.sb-round-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  background-color: rgba(15, 23, 42, 0.7);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-sm);
  color: #94a3b8;
  font-family: var(--font-mono);
}

.sb-round-chip__text {
  display: flex;
  flex-direction: column;
  line-height: 1.1;
}

.sb-round-chip__text span {
  font-size: 0.625rem;
  color: #64748b;
}

.sb-round-chip__text strong {
  font-size: 0.75rem;
}

.sb-round-chip--done {
  border-color: rgba(255, 255, 255, 0.2);
  color: #cbd5e1;
}

.sb-round-chip--active {
  background-color: rgba(16, 185, 129, 0.15);
  border-color: #10b981;
  color: #34d399;
  box-shadow: 0 0 15px rgba(16, 185, 129, 0.25);
}

.sb-round-chip__dot {
  width: 8px;
  height: 8px;
  background-color: #10b981;
  border-radius: 50%;
}

.sb-round-chip--dim {
  opacity: 0.45;
}

/* Giant Clock */
.sb-clock-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.sb-giant-timer {
  font-size: 4.75rem;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.03em;
  color: #34d399;
  text-shadow: 0 0 35px rgba(16, 185, 129, 0.55), 0 0 10px rgba(16, 185, 129, 0.8);
}

.sb-clock-caption {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  font-size: 0.6875rem;
  letter-spacing: 0.12em;
  color: #34d399;
  text-transform: uppercase;
}

/* Consensus Chip */
.sb-consensus-box {
  display: flex;
  justify-content: flex-end;
}

.sb-consensus-chip {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  background-color: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-sm);
}

.sb-consensus-text {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
}

.sb-consensus-text span {
  font-size: 0.625rem;
  color: #94a3b8;
  letter-spacing: 0.08em;
}

.sb-consensus-text strong {
  font-size: 0.8125rem;
  color: #f87171;
}

/* --- Main Split Arena Grid --- */
.sb-main-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-top: 18px;
  flex: 1;
}

.sb-corner {
  display: flex;
  flex-direction: column;
  border-radius: var(--radius-lg);
  padding: 24px;
  border: 1px solid transparent;
  position: relative;
}

.sb-corner--red {
  background: radial-gradient(ellipse at top left, rgba(220, 38, 38, 0.22) 0%, rgba(13, 20, 36, 0.95) 75%);
  border-color: rgba(220, 38, 38, 0.45);
  box-shadow: 0 10px 30px -10px rgba(220, 38, 38, 0.25);
}

.sb-corner--blue {
  background: radial-gradient(ellipse at top right, rgba(37, 99, 235, 0.22) 0%, rgba(13, 20, 36, 0.95) 75%);
  border-color: rgba(37, 99, 235, 0.45);
  box-shadow: 0 10px 30px -10px rgba(37, 99, 235, 0.25);
}

.sb-corner__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.sb-corner__title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.875rem;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.sb-corner--red .sb-corner__title {
  color: #f87171;
}

.sb-corner--blue .sb-corner__title {
  color: #60a5fa;
}

.sb-corner-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
}
.sb-corner-dot--red { background-color: #ef4444; }
.sb-corner-dot--blue { background-color: #3b82f6; }

.sb-contingent-tag {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  color: #cbd5e1;
  background-color: rgba(0, 0, 0, 0.35);
  padding: 4px 10px;
  border-radius: var(--radius-full);
}

.sb-athlete-row {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  margin-bottom: 18px;
}

.sb-athlete-meta {
  display: flex;
  flex-direction: column;
}

.sb-athlete-role {
  font-size: 0.6875rem;
  letter-spacing: 0.1em;
  color: #94a3b8;
}

.sb-athlete-name {
  font-size: 1.875rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #ffffff;
  line-height: 1.1;
}

.sb-athlete-id {
  font-size: 0.8125rem;
  color: #94a3b8;
}

/* Giant Score Box */
.sb-score-box {
  background-color: rgba(6, 11, 23, 0.75);
  border-radius: var(--radius-md);
  padding: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 20px;
  position: relative;
  border: 1px solid rgba(255, 255, 255, 0.06);
  margin-bottom: 20px;
}

.sb-giant-score {
  font-size: 7.5rem;
  font-weight: 900;
  line-height: 0.9;
  letter-spacing: -0.05em;
}

.sb-score-box--red .sb-giant-score {
  color: #ef4444;
  text-shadow: 0 0 45px rgba(239, 68, 68, 0.65), 0 0 15px rgba(239, 68, 68, 0.9);
}

.sb-score-box--blue .sb-giant-score {
  color: #3b82f6;
  text-shadow: 0 0 45px rgba(59, 130, 246, 0.65), 0 0 15px rgba(59, 130, 246, 0.9);
}

.sb-score-label {
  display: flex;
  flex-direction: column;
  line-height: 1.1;
}

.sb-score-label strong {
  font-size: 1.125rem;
  color: #94a3b8;
}

.sb-score-label span {
  font-size: 0.6875rem;
  letter-spacing: 0.1em;
  color: #64748b;
}

/* Floating point toast */
.sb-floating-toast {
  position: absolute;
  top: 14px;
  right: 18px;
  padding: 6px 14px;
  border-radius: var(--radius-sm);
  font-size: 1.125rem;
  font-weight: 800;
  box-shadow: var(--shadow-lg);
  pointer-events: none;
}
.sb-floating-toast--red {
  background-color: #dc2626;
  color: #ffffff;
  box-shadow: 0 0 20px rgba(220, 38, 38, 0.8);
}
.sb-floating-toast--blue {
  background-color: #2563eb;
  color: #ffffff;
  box-shadow: 0 0 20px rgba(37, 99, 235, 0.8);
}

.toast-pop-enter-active,
.toast-pop-leave-active {
  transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.toast-pop-enter-from {
  opacity: 0;
  transform: translateY(15px) scale(0.85);
}
.toast-pop-leave-to {
  opacity: 0;
  transform: translateY(-20px) scale(1.1);
}

/* Round Breakdown */
.sb-breakdown-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-bottom: 20px;
}

.sb-round-card {
  background-color: rgba(10, 16, 30, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: var(--radius-sm);
  padding: 10px;
  text-align: center;
}

.sb-round-card__label {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 0.6875rem;
  color: #64748b;
  margin-bottom: 4px;
}

.sb-round-card__dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
}
.sb-round-card__dot--red { background-color: #ef4444; }
.sb-round-card__dot--blue { background-color: #3b82f6; }

.sb-round-card__val {
  font-size: 1.75rem;
  font-weight: 800;
  color: #ffffff;
  line-height: 1;
}

.sb-round-card--active-red {
  background-color: rgba(220, 38, 38, 0.18);
  border-color: rgba(220, 38, 38, 0.5);
}
.sb-round-card--active-blue {
  background-color: rgba(37, 99, 235, 0.18);
  border-color: rgba(37, 99, 235, 0.5);
}

/* Penalties Strip */
.sb-penalties-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 14px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  margin-top: auto;
}

.sb-penalties-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  color: #94a3b8;
}

.sb-penalties-chips {
  display: flex;
  gap: 8px;
}

.sb-pen-chip {
  padding: 4px 10px;
  background-color: rgba(0, 0, 0, 0.4);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-xs);
  font-size: 0.75rem;
  color: #cbd5e1;
}

.sb-pen-chip strong {
  color: #ffffff;
}

.sb-pen-chip--alert-red {
  background-color: rgba(220, 38, 38, 0.2);
  border-color: rgba(220, 38, 38, 0.5);
  color: #f87171;
}

.sb-pen-chip--alert-blue {
  background-color: rgba(37, 99, 235, 0.2);
  border-color: rgba(37, 99, 235, 0.5);
  color: #60a5fa;
}

/* --- Broadcast Footer --- */
.sb-bottom-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  margin-top: 18px;
  font-size: 0.75rem;
  color: #64748b;
}

.sb-bottom-left, .sb-bottom-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.sb-live-indicator {
  color: #10b981;
  font-weight: 700;
}

.sb-bar-sep {
  opacity: 0.4;
}

/* --- Winner Overlay --- */
.sb-winner-overlay {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: rgba(0, 0, 0, 0.88);
  backdrop-filter: blur(12px);
  z-index: 999;
}

.sb-winner-card {
  text-align: center;
  padding: 48px 60px;
  border-radius: var(--radius-xl);
  background-color: #0d1424;
  border: 2px solid rgba(255, 255, 255, 0.15);
  box-shadow: 0 0 60px rgba(0, 0, 0, 0.8);
  max-width: 600px;
  width: 90%;
}

.sb-winner-overlay--red .sb-winner-card {
  border-color: #dc2626;
  box-shadow: 0 0 80px rgba(220, 38, 38, 0.5);
}

.sb-winner-overlay--blue .sb-winner-card {
  border-color: #2563eb;
  box-shadow: 0 0 80px rgba(37, 99, 235, 0.5);
}

.sb-winner-badge {
  display: inline-block;
  padding: 6px 18px;
  background-color: rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-full);
  font-size: 0.875rem;
  letter-spacing: 0.12em;
  color: #e2e8f0;
  margin-bottom: 20px;
}

.sb-winner-name {
  font-size: 2.75rem;
  font-weight: 900;
  color: #ffffff;
  margin-bottom: 8px;
  line-height: 1.1;
}

.sb-winner-corner {
  font-size: 1.25rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  margin-bottom: 16px;
}

.sb-winner-type {
  font-size: 1rem;
  color: #94a3b8;
  margin-bottom: 28px;
}

.sb-winner-scores {
  font-size: 3rem;
  margin-bottom: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 20px;
}

.sb-winner-close {
  padding: 10px 24px;
  background-color: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: #ffffff;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 0.875rem;
  transition: all var(--transition-fast);
}
.sb-winner-close:hover {
  background-color: rgba(255, 255, 255, 0.2);
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(0.85); }
}

.text-red-400 { color: #f87171; }
.text-blue-400 { color: #60a5fa; }
.text-emerald-400 { color: #34d399; }
.text-slate-500 { color: #64748b; }
.font-bold { font-weight: 700; }
</style>
