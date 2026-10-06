<script setup lang="ts">
import { ref, computed, onMounted, watch, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { useWebSocket } from '@/composables/useWebSocket'
import { useTimer } from '@/composables/useTimer'
import { playGong } from '@/composables/useGong'
import { matchStatusFromState, type MatchStatus } from '@/composables/matchStatus'
import { arenaService } from '@/features/arena/api/arenaService'

const route = useRoute()
const arenaId = computed(() => (route.params.arena_id as string) || 'gelanggang-1')

// --- Match State ---
const competitionTitle = ref('KEJUARAAN NASIONAL PENCAK SILAT 2025')
const tournamentSeries = ref('PENCAK SILAT DIGITAL SCORING')
const arenaName = ref('GELANGGANG')
const categoryName = ref('KATEGORI TANDING')
const matchNumber = ref('')
const rulesetLabel = ref('PERSILAT RULES')

// Round & State
const currentRound = ref(1)
const matchStatus = ref<MatchStatus>('idle')

// Athletes
const redAthlete = ref({ name: 'MENUNGGU PERTANDINGAN', team: '', idTag: '' })
const blueAthlete = ref({ name: 'MENUNGGU PERTANDINGAN', team: '', idTag: '' })

// Scores per round (red/blue = poin teknik, redPen/bluePen = potongan hukuman)
const roundScores = ref([
  { round: 1, red: 0, blue: 0, redPen: 0, bluePen: 0 },
  { round: 2, red: 0, blue: 0, redPen: 0, bluePen: 0 },
  { round: 3, red: 0, blue: 0, redPen: 0, bluePen: 0 },
])

// Total bersih = poin teknik dikurangi potongan hukuman (sama dengan red_total/blue_total server)
const redTotal = computed(() => roundScores.value.reduce((acc, r) => acc + r.red - r.redPen, 0))
const blueTotal = computed(() => roundScores.value.reduce((acc, r) => acc + r.blue - r.bluePen, 0))
const redPenaltyTotal = computed(() => roundScores.value.reduce((acc, r) => acc + r.redPen, 0))
const bluePenaltyTotal = computed(() => roundScores.value.reduce((acc, r) => acc + r.bluePen, 0))

// Banner status skor
const consensusStatus = computed(() => {
  if (redTotal.value > blueTotal.value) return 'MERAH MEMIMPIN'
  if (blueTotal.value > redTotal.value) return 'BIRU MEMIMPIN'
  return 'SKOR IMBANG'
})

// Chip status per babak, diturunkan dari babak aktif dan status pertandingan
const roundChips = computed(() =>
  [1, 2, 3].map((round) => {
    if (matchStatus.value === 'finished' || round < currentRound.value) {
      return { round, kind: 'done' as const, label: 'SELESAI' }
    }
    if (round > currentRound.value || matchStatus.value === 'idle' || matchStatus.value === 'ready') {
      return { round, kind: 'waiting' as const, label: 'MENUNGGU' }
    }
    const label = { running: 'AKTIF', paused: 'DIJEDA', break: 'ISTIRAHAT' }[matchStatus.value]
    return { round, kind: 'active' as const, label: label ?? 'AKTIF' }
  }),
)

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

// Sinyal arena dari wasit (banner selama aktif)
const medicActive = ref(false)
const varActive = ref(false)

const RESULT_LABEL: Record<string, string> = {
  point_win: 'MENANG ANGKA (POINT WIN)',
  disqualification: 'MENANG DISKUALIFIKASI',
  draw: 'SERI (KEPUTUSAN WASIT)',
}

// Winner Announcement Modal
const showWinnerModal = ref(false)
const winnerData = ref<{ side: 'red' | 'blue'; name: string; resultType: string } | null>(null)

// Tampilan kemenangan menutup sendiri agar scoreboard kembali terlihat.
const WINNER_AUTO_CLOSE_MS = 30_000
let winnerCloseTimer: ReturnType<typeof setTimeout> | undefined
watch(showWinnerModal, (open) => {
  clearTimeout(winnerCloseTimer)
  if (open) winnerCloseTimer = setTimeout(() => (showWinnerModal.value = false), WINNER_AUTO_CLOSE_MS)
})
onBeforeUnmount(() => clearTimeout(winnerCloseTimer))

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
const { isConnected, sendMessage, onMessage } = useWebSocket(arenaId.value)

onMounted(async () => {
  // Initialize timer to 01:41 (101 seconds) as depicted in Figma 01_arena_scoreboard.png

  try {
    const snapshot = await arenaService.getLiveScoreboard(arenaId.value)
    if (snapshot && typeof snapshot === 'object' && snapshot.status !== 'no active match') {
      if (snapshot.round) currentRound.value = snapshot.round
      if (snapshot.red_athlete) redAthlete.value.name = snapshot.red_athlete
      if (snapshot.blue_athlete) blueAthlete.value.name = snapshot.blue_athlete
    }
  } catch {
    // Keep local baseline state
  }

  // Request sync from backend WebSocket hub
  sendMessage('sync_request', {})

  // Listen to WebSocket broadcasts
  onMessage('sync_response', (payload) => {
    if (payload.medic_active !== undefined) medicActive.value = payload.medic_active
    if (payload.var_active !== undefined) varActive.value = payload.var_active
    if (payload.round) currentRound.value = payload.round
    matchStatus.value = matchStatusFromState(payload.state, payload.timer_running) ?? matchStatus.value
    if (payload.timer_remaining_ms !== undefined) {
      timer.sync(payload.timer_remaining_ms, payload.timer_running ?? false, payload.timer_ends_at_ms)
    }
    payload.scores?.forEach((score, i) => {
      const row = roundScores.value[i]
      if (!row) return
      row.red = score.RedTechnical
      row.blue = score.BlueTechnical
      row.redPen = score.RedPenalties
      row.bluePen = score.BluePenalties
    })
  })

  onMessage('state_change', (payload) => {
    if (payload.round) currentRound.value = payload.round
    matchStatus.value = matchStatusFromState(payload.state, payload.timer_running) ?? matchStatus.value
    if (payload.timer_remaining_ms !== undefined) {
      timer.sync(
        payload.timer_remaining_ms,
        payload.timer_running ?? matchStatus.value === 'running',
        payload.timer_ends_at_ms,
      )
    }
  })

  onMessage('timer_sync', (payload) => {
    if (payload.timer_remaining_ms !== undefined) {
      timer.sync(payload.timer_remaining_ms, payload.timer_running ?? false, payload.timer_ends_at_ms)
    }
  })

  onMessage('score_confirmed', (payload) => {
    const pointText = `+${payload.points} ${payload.score_type.toUpperCase()}`
    triggerScoreToast(payload.target === 'red' ? 'red' : 'blue', pointText)
  })

  onMessage('score_update', (payload) => {
    const idx = roundScores.value.findIndex(r => r.round === payload.round)
    if (idx !== -1) {
      if (payload.red_technical !== undefined) roundScores.value[idx].red = payload.red_technical
      if (payload.blue_technical !== undefined) roundScores.value[idx].blue = payload.blue_technical
      if (payload.red_penalties !== undefined) roundScores.value[idx].redPen = payload.red_penalties
      if (payload.blue_penalties !== undefined) roundScores.value[idx].bluePen = payload.blue_penalties
    }
  })

  onMessage('match_result', (payload) => {
    const side = payload.corner === 'red' ? 'red' : 'blue'
    const name = side === 'red' ? redAthlete.value.name : blueAthlete.value.name
    winnerData.value = {
      side,
      name,
      resultType: RESULT_LABEL[payload.result_type ?? ''] ?? (payload.result_type || RESULT_LABEL.point_win),
    }
    showWinnerModal.value = true
    medicActive.value = false
    varActive.value = false
  })

  // Sinyal dari meja wasit: gong berbunyi sesaat; dokter/VAR menampilkan banner selama aktif
  onMessage('arena_signal', (payload) => {
    if (payload.signal === 'gong') playGong()
    else if (payload.signal === 'medic') medicActive.value = payload.active
    else if (payload.signal === 'var') varActive.value = payload.active
  })
})
</script>

<template>
  <div class="sb-screen">
    <div v-if="medicActive || varActive" class="sb-signal-banner font-mono">
      <span v-if="medicActive">⚕ PERTANDINGAN DIHENTIKAN SEMENTARA — DOKTER PERTANDINGAN</span>
      <span v-if="varActive">▶ TINJAUAN VIDEO (VAR) SEDANG BERJALAN</span>
    </div>
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
        <div
          v-for="chip in roundChips"
          :key="chip.round"
          class="sb-round-chip"
          :class="{
            'sb-round-chip--done': chip.kind === 'done',
            'sb-round-chip--active': chip.kind === 'active',
            'sb-round-chip--dim': chip.kind === 'waiting',
          }"
        >
          <svg v-if="chip.kind === 'done'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="20 6 9 17 4 12" />
          </svg>
          <span v-else-if="chip.kind === 'active'" class="sb-round-chip__dot" />
          <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <polyline points="12 6 12 12 16 14" />
          </svg>
          <div class="sb-round-chip__text">
            <span>BABAK {{ chip.round }}</span>
            <strong>{{ chip.label }}</strong>
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
          {{ matchStatus === 'break' ? 'WAKTU ISTIRAHAT' : 'WAKTU BERTANDING' }}
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
            <span class="sb-pen-chip" :class="{ 'sb-pen-chip--alert-red': redPenaltyTotal > 0 }">
              POTONGAN: <strong>{{ redPenaltyTotal > 0 ? `-${redPenaltyTotal}` : 0 }}</strong>
            </span>
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
            <span class="sb-pen-chip" :class="{ 'sb-pen-chip--alert-blue': bluePenaltyTotal > 0 }">
              POTONGAN: <strong>{{ bluePenaltyTotal > 0 ? `-${bluePenaltyTotal}` : 0 }}</strong>
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

.sb-signal-banner {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 40;
  display: flex;
  justify-content: center;
  gap: 24px;
  padding: 10px 16px;
  background: rgba(217, 119, 6, 0.92);
  color: #fff;
  font-size: 1rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  animation: sb-signal-pulse 1.4s ease-in-out infinite;
}
@keyframes sb-signal-pulse {
  50% {
    background: rgba(180, 83, 9, 0.92);
  }
}

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
