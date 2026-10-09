<script setup lang="ts">
import { getErrorMessage } from '@/api/error'
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { AppButton } from '@/components/ui'
import {
  useMatchesQuery,
  useLoadArenaMutation,
  useStartMatchMutation,
  usePauseMatchMutation,
  useResumeMatchMutation,
  useFinishMatchMutation,
} from '../api/matchQueries'
import { useArenasQuery } from '@/features/arena/api/arenaQueries'
import { useParticipantsQuery } from '@/features/athletes/api/athleteQueries'
import { useAuthStore } from '@/stores/auth'
import type { MatchDTO } from '../api/matchService'
import CreateMatchDialog from '../components/CreateMatchDialog.vue'
import MatchScoreDialog from '../components/MatchScoreDialog.vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const canControlMatch = computed(() => ['admin', 'dewan', 'wasit'].includes(authStore.role || ''))
// Server hanya mengizinkan admin/dewan membuat partai manual.
const canCreateMatch = computed(() => authStore.isAdminOrDewan)

const props = withDefaults(
  defineProps<{
    competitionId?: string
  }>(),
  {
    competitionId: 'b26d7145-d341-4795-ac58-bff86d13cbf8',
  },
)

const activeCompId = computed(() => props.competitionId || 'b26d7145-d341-4795-ac58-bff86d13cbf8')

// Queries
const activeStatusFilter = ref<string>('all')
const { data: rawMatches, isLoading, isFetching, error, refetch } = useMatchesQuery(activeCompId)
const { data: rawArenas } = useArenasQuery(activeCompId)
const { data: rawParticipants } = useParticipantsQuery(activeCompId)

// Mutations
const loadArenaMutation = useLoadArenaMutation(activeCompId)
const startMatchMutation = useStartMatchMutation(activeCompId)
const pauseMatchMutation = usePauseMatchMutation(activeCompId)
const resumeMatchMutation = useResumeMatchMutation(activeCompId)
const finishMatchMutation = useFinishMatchMutation(activeCompId)

const matches = computed(() => rawMatches.value || [])
const arenas = computed(() => rawArenas.value || [])
const participants = computed(() => rawParticipants.value || [])

// Lookup maps
const arenaMap = computed(() => {
  const map: Record<string, string> = {}
  for (const a of arenas.value) {
    map[a.id] = a.name
  }
  return map
})

const participantMap = computed(() => {
  const map: Record<string, { name: string; club: string; gender: string; weightClass: string }> =
    {}
  for (const p of participants.value) {
    map[p.id] = {
      name: p.name || 'Pesilat',
      club: p.club || 'Kontingen Mandiri',
      gender: p.gender || 'male',
      weightClass: p.weight_class || 'A',
    }
  }
  return map
})

// UI Filter
const selectedArenaFilter = ref<string>('all')
const searchQuery = ref('')

// Toast State
const toast = ref<{ title: string; message: string; type: 'success' | 'danger' | 'info' } | null>(
  null,
)
let toastTimer: ReturnType<typeof setTimeout> | null = null

function showToast(
  title: string,
  message: string,
  type: 'success' | 'danger' | 'info' = 'success',
) {
  if (toastTimer) clearTimeout(toastTimer)
  toast.value = { title, message, type }
  toastTimer = setTimeout(() => {
    toast.value = null
  }, 4000)
}

// Operational Actions
const actionLoadingId = ref<string | null>(null)

async function handleLoadToArena(m: MatchDTO) {
  if (!m.arena_id) {
    showToast('Gelanggang Belum Diatur', 'Tentukan gelanggang terlebih dahulu.', 'info')
    return
  }
  actionLoadingId.value = m.id
  try {
    await loadArenaMutation.mutateAsync({ matchId: m.id, arenaId: m.arena_id })
    showToast(
      'Dimuat ke Gelanggang',
      `Partai berhasil dimuat ke ${arenaMap.value[m.arena_id] || 'Gelanggang'}.`,
      'success',
    )
  } catch (err) {
    showToast('Gagal Memuat', getErrorMessage(err, 'Gagal memuat partai.'), 'danger')
  } finally {
    actionLoadingId.value = null
  }
}

async function handleStartMatch(m: MatchDTO) {
  actionLoadingId.value = m.id
  try {
    await startMatchMutation.mutateAsync(m.id)
    showToast('Pertandingan Dimulai', 'Timer dan konsol wasit telah diaktifkan.', 'success')
  } catch (err) {
    showToast('Gagal Memulai', getErrorMessage(err, 'Gagal memulai pertandingan.'), 'danger')
  } finally {
    actionLoadingId.value = null
  }
}

async function handlePauseMatch(m: MatchDTO) {
  actionLoadingId.value = m.id
  try {
    await pauseMatchMutation.mutateAsync(m.id)
    showToast('Pertandingan Dijeda', 'Pertandingan sementara dijeda (break/time out).', 'info')
  } catch (err) {
    showToast('Gagal Menjeda', getErrorMessage(err, 'Gagal menjeda.'), 'danger')
  } finally {
    actionLoadingId.value = null
  }
}

async function handleResumeMatch(m: MatchDTO) {
  actionLoadingId.value = m.id
  try {
    await resumeMatchMutation.mutateAsync(m.id)
    showToast('Pertandingan Dilanjutkan', 'Timer pertandingan kembali berjalan.', 'success')
  } catch (err) {
    showToast('Gagal Melanjutkan', getErrorMessage(err, 'Gagal melanjutkan.'), 'danger')
  } finally {
    actionLoadingId.value = null
  }
}

async function handleFinishMatch(m: MatchDTO) {
  if (!confirm('Apakah Anda yakin ingin menyelesaikan partai ini?')) return
  actionLoadingId.value = m.id
  try {
    await finishMatchMutation.mutateAsync(m.id)
    showToast('Partai Selesai', 'Partai resmi dinyatakan selesai dan skor tercatat.', 'success')
  } catch (err) {
    showToast('Gagal Menyelesaikan', getErrorMessage(err, 'Gagal menyelesaikan partai.'), 'danger')
  } finally {
    actionLoadingId.value = null
  }
}

// Tambah partai manual & rincian skor
const isCreateOpen = ref(false)
const scoreMatch = ref<MatchDTO | null>(null)

function handleMatchCreated() {
  showToast(
    'Partai Dibuat',
    'Partai manual ditambahkan ke jadwal dan siap dimuat ke gelanggang.',
    'success',
  )
}

// External Links
function openScoreboard(arenaId?: string) {
  const url = arenaId ? `/scoreboard/${arenaId}` : '/scoreboard'
  window.open(url, '_blank')
}

function openRefereeConsole() {
  window.open('/referee', '_blank')
}

// Stats
// Partai yang sedang dijeda masih "live" (belum selesai, sudah di gelanggang): tidak masuk antrean
const isLive = (status: string) => status === 'running' || status === 'paused'

const stats = computed(() => {
  const all = matches.value
  const running = all.filter((m) => isLive(m.status)).length
  const ready = all.filter((m) => m.status === 'ready' || m.status === 'idle').length
  const finished = all.filter((m) => m.status === 'finished').length
  return {
    total: all.length,
    running,
    ready,
    finished,
  }
})

// Filtered Matches
const filteredMatches = computed(() => {
  let list = [...matches.value]

  // Status Filter
  if (activeStatusFilter.value === 'running') {
    list = list.filter((m) => isLive(m.status))
  } else if (activeStatusFilter.value === 'ready') {
    list = list.filter((m) => m.status === 'ready' || m.status === 'idle')
  } else if (activeStatusFilter.value === 'finished') {
    list = list.filter((m) => m.status === 'finished')
  }

  // Arena Filter
  if (selectedArenaFilter.value !== 'all') {
    list = list.filter((m) => m.arena_id === selectedArenaFilter.value)
  }

  // Search
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase().trim()
    list = list.filter((m) => {
      const red = participantMap.value[m.athlete_red_id]?.name.toLowerCase() || ''
      const blue = participantMap.value[m.athlete_blue_id]?.name.toLowerCase() || ''
      const arena = (arenaMap.value[m.arena_id] || '').toLowerCase()
      return red.includes(q) || blue.includes(q) || arena.includes(q)
    })
  }

  return list
})
</script>

<template>
  <div
    class="sch-ops @container"
    role="region"
    aria-label="Manajemen Jadwal dan Partai Pertandingan"
  >
    <!-- Toast Notification -->
    <transition name="sch-fade">
      <div
        v-if="toast"
        class="sch-toast"
        :class="`sch-toast--${toast.type}`"
        role="status"
        aria-live="polite"
      >
        <div class="sch-toast__dot" />
        <div class="sch-toast__content">
          <strong class="sch-toast__title">{{ toast.title }}</strong>
          <span class="sch-toast__msg">{{ toast.message }}</span>
        </div>
        <button class="sch-toast__close" aria-label="Tutup notifikasi" @click="toast = null">
          ✕
        </button>
      </div>
    </transition>

    <!-- Header Banner -->
    <header class="sch-header">
      <div class="sch-header__left">
        <div class="sch-header__icon-box">
          <svg
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.2"
          >
            <rect width="18" height="18" x="3" y="4" rx="2" ry="2" />
            <line x1="16" y1="2" x2="16" y2="6" />
            <line x1="8" y1="2" x2="8" y2="6" />
            <line x1="3" y1="10" x2="21" y2="10" />
          </svg>
        </div>
        <div>
          <div class="sch-header__sub font-mono">
            <span>JADWAL PARTAI & TERTIB PERTANDINGAN</span>
            <span class="sch-bullet">•</span>
            <span class="sch-tag">IPSI RULES 2023</span>
            <span class="sch-bullet">•</span>
            <span class="sch-pulse-wrapper">
              <span class="sch-pulse-dot" />
              <span>MEJA OPERATOR RESMI</span>
            </span>
          </div>
          <h2 class="sch-header__title">JADWAL PERTANDINGAN TURNAMEN</h2>
        </div>
      </div>

      <div class="sch-header__actions font-mono">
        <AppButton variant="secondary" size="sm" :loading="isFetching" @click="() => refetch()">
          <template #icon>
            <svg
              width="15"
              height="15"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" />
              <path d="M21 3v5h-5" />
              <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" />
              <path d="M3 21v-5h5" />
            </svg>
          </template>
          Segarkan Jadwal
        </AppButton>
        <AppButton
          v-if="canCreateMatch"
          variant="emerald"
          size="sm"
          data-test="match-add"
          @click="isCreateOpen = true"
        >
          Tambah Partai
        </AppButton>
      </div>
    </header>

    <!-- Metrics Strip -->
    <section class="sch-metrics" aria-label="Statistik Partai">
      <div class="sch-metric-card">
        <span class="sch-metric-label">TOTAL PARTAI TERJADWAL</span>
        <div class="sch-metric-value-row">
          <span class="sch-metric-val tabular-nums">{{ stats.total }}</span>
          <span class="sch-metric-badge font-mono">PARTAI</span>
        </div>
        <span class="sch-metric-desc">Semua gelanggang aktif</span>
      </div>

      <div class="sch-metric-card sch-metric-card--running">
        <div class="sch-metric-corner-tag font-mono">LIVE</div>
        <span class="sch-metric-label">SEDANG BERLANGSUNG</span>
        <div class="sch-metric-value-row">
          <span class="sch-metric-val sch-val-emerald tabular-nums">{{ stats.running }}</span>
          <span class="sch-metric-badge sch-badge-emerald font-mono">LIVE RING</span>
        </div>
        <span class="sch-metric-desc">Sedang tanding di gelanggang</span>
      </div>

      <div class="sch-metric-card sch-metric-card--ready">
        <div class="sch-metric-corner-tag font-mono">STANDBY</div>
        <span class="sch-metric-label">SIAP & MENUNGGU</span>
        <div class="sch-metric-value-row">
          <span class="sch-metric-val sch-val-amber tabular-nums">{{ stats.ready }}</span>
          <span class="sch-metric-badge sch-badge-amber font-mono">ANTREAN</span>
        </div>
        <span class="sch-metric-desc">Pesilat siap masuk matras</span>
      </div>

      <div class="sch-metric-card sch-metric-card--finished">
        <div class="sch-metric-corner-tag font-mono">SELESAI</div>
        <span class="sch-metric-label">PARTAI SELESAI</span>
        <div class="sch-metric-value-row">
          <span class="sch-metric-val sch-val-blue tabular-nums">{{ stats.finished }}</span>
          <span class="sch-metric-badge sch-badge-blue font-mono">REKAP</span>
        </div>
        <span class="sch-metric-desc">Pemenang telah disahkan dewan</span>
      </div>
    </section>

    <!-- Controls Toolbar -->
    <section class="sch-toolbar">
      <div class="sch-toolbar__tabs font-mono" role="tablist">
        <button
          class="sch-tab-btn"
          :class="{ 'sch-tab-btn--active': activeStatusFilter === 'all' }"
          role="tab"
          :aria-selected="activeStatusFilter === 'all'"
          @click="activeStatusFilter = 'all'"
        >
          Semua ({{ stats.total }})
        </button>

        <button
          class="sch-tab-btn"
          :class="{ 'sch-tab-btn--active-running': activeStatusFilter === 'running' }"
          role="tab"
          :aria-selected="activeStatusFilter === 'running'"
          @click="activeStatusFilter = 'running'"
        >
          <span class="sch-tab-dot sch-dot-emerald" />
          Live ({{ stats.running }})
        </button>

        <button
          class="sch-tab-btn"
          :class="{ 'sch-tab-btn--active-ready': activeStatusFilter === 'ready' }"
          role="tab"
          :aria-selected="activeStatusFilter === 'ready'"
          @click="activeStatusFilter = 'ready'"
        >
          <span class="sch-tab-dot sch-dot-amber" />
          Antrean ({{ stats.ready }})
        </button>

        <button
          class="sch-tab-btn"
          :class="{ 'sch-tab-btn--active-finished': activeStatusFilter === 'finished' }"
          role="tab"
          :aria-selected="activeStatusFilter === 'finished'"
          @click="activeStatusFilter = 'finished'"
        >
          <span class="sch-tab-dot sch-dot-blue" />
          Selesai ({{ stats.finished }})
        </button>
      </div>

      <div class="sch-toolbar__search-group">
        <!-- Arena Select -->
        <div class="sch-filter-wrapper">
          <select
            v-model="selectedArenaFilter"
            class="sch-select font-mono"
            aria-label="Filter Gelanggang"
          >
            <option value="all">Semua Gelanggang</option>
            <option v-for="a in arenas" :key="a.id" :value="a.id">
              {{ a.name }}
            </option>
          </select>
        </div>

        <!-- Search Input -->
        <div class="sch-search-box">
          <svg
            class="sch-search-icon"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            v-model="searchQuery"
            type="search"
            class="sch-search-input font-mono"
            placeholder="Cari pesilat / gelanggang..."
            aria-label="Cari partai"
          />
          <button
            v-if="searchQuery"
            class="sch-search-clear"
            aria-label="Hapus filter"
            @click="searchQuery = ''"
          >
            ✕
          </button>
        </div>
      </div>
    </section>

    <!-- Skeleton Loading -->
    <div v-if="isLoading" class="sch-grid" aria-busy="true">
      <div v-for="i in 4" :key="i" class="sch-skeleton-card">
        <div class="skeleton-line skeleton-header" />
        <div class="skeleton-line skeleton-matchup" />
        <div class="skeleton-line skeleton-btn" />
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="sch-empty-state sch-empty-state--error">
      <div class="sch-empty-icon sch-empty-icon--error">
        <svg
          width="32"
          height="32"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
      </div>
      <h3 class="sch-empty-title">Gagal Memuat Jadwal Pertandingan</h3>
      <p class="sch-empty-desc">
        {{ getErrorMessage(error, 'Terjadi gangguan jaringan saat memuat jadwal.') }}
      </p>
      <AppButton variant="secondary" size="md" @click="() => refetch()"> Coba Lagi </AppButton>
    </div>

    <!-- Empty State -->
    <div v-else-if="filteredMatches.length === 0" class="sch-empty-state">
      <div class="sch-empty-icon">
        <svg
          width="36"
          height="36"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.8"
        >
          <rect width="18" height="18" x="3" y="4" rx="2" ry="2" />
          <line x1="16" y1="2" x2="16" y2="6" />
          <line x1="8" y1="2" x2="8" y2="6" />
          <line x1="3" y1="10" x2="21" y2="10" />
        </svg>
      </div>
      <h3 class="sch-empty-title">
        {{ searchQuery ? 'Tidak Ada Partai yang Cocok' : 'Belum Ada Jadwal Partai' }}
      </h3>
      <p class="sch-empty-desc">
        {{
          searchQuery
            ? `Pencarian "${searchQuery}" tidak menghasilkan jadwal partai manapun.`
            : 'Belum ada partai pertandingan yang dijadwalkan pada kompetisi ini. Silakan undi susunan pohon bagan di menu Bagan Pertandingan.'
        }}
      </p>

      <div class="sch-empty-actions font-mono">
        <AppButton v-if="searchQuery" variant="secondary" size="md" @click="searchQuery = ''">
          Reset Pencarian
        </AppButton>
        <AppButton
          v-else
          variant="emerald"
          size="md"
          @click="router.push({ query: { ...route.query, tab: 'bagan' } })"
        >
          Buka Bagan Pertandingan
        </AppButton>
      </div>
    </div>

    <!-- Responsive Cards Grid -->
    <div v-else class="sch-grid" role="list">
      <article
        v-for="(m, idx) in filteredMatches"
        :key="m.id"
        class="sch-card"
        :class="`sch-card--status-${m.status}`"
        role="listitem"
      >
        <!-- Accent stripe -->
        <div class="sch-card__accent" :class="`sch-accent--${m.status}`" />

        <!-- Header -->
        <header class="sch-card__header font-mono">
          <div class="sch-card__tag-group">
            <span class="sch-partai-badge">PARTAI {{ String(idx + 1).padStart(2, '0') }}</span>
            <span class="sch-arena-tag">{{ arenaMap[m.arena_id] || 'GELANGGANG' }}</span>
          </div>

          <div class="sch-status-pill" :class="`sch-status-pill--${m.status}`">
            <span v-if="isLive(m.status)" class="sch-pulse-dot-live" />
            <span>
              {{
                m.status === 'running'
                  ? 'LIVE PERTANDINGAN'
                  : m.status === 'finished'
                    ? 'SELESAI'
                    : m.status === 'paused'
                      ? 'DIJEDA'
                      : 'SIAP TANDING'
              }}
            </span>
          </div>
        </header>

        <!-- Matchup Body -->
        <div class="sch-card__body">
          <div class="sch-matchup-grid">
            <!-- Red Corner -->
            <div
              class="sch-fighter-box sch-fighter-box--red"
              :class="{
                'sch-fighter-box--winner': m.winner_id && m.winner_id === m.athlete_red_id,
              }"
            >
              <div class="sch-corner-bar sch-bar-red" />
              <div class="sch-fighter-info">
                <span class="sch-corner-label font-mono">SUDUT MERAH</span>
                <strong class="sch-fighter-name">
                  {{ participantMap[m.athlete_red_id]?.name || 'Pesilat Merah' }}
                </strong>
                <span class="sch-fighter-club font-mono">
                  {{ participantMap[m.athlete_red_id]?.club || 'Perguruan' }}
                </span>
              </div>
              <span
                v-if="m.winner_id && m.winner_id === m.athlete_red_id"
                class="sch-win-tag font-mono"
                >MENANG</span
              >
            </div>

            <!-- VS Center Badge -->
            <div class="sch-vs-badge font-mono">
              <span class="sch-vs-text">VS</span>
              <span class="sch-round-sub">BABAK {{ m.current_round }}</span>
            </div>

            <!-- Blue Corner -->
            <div
              class="sch-fighter-box sch-fighter-box--blue"
              :class="{
                'sch-fighter-box--winner': m.winner_id && m.winner_id === m.athlete_blue_id,
              }"
            >
              <div class="sch-corner-bar sch-bar-blue" />
              <div class="sch-fighter-info">
                <span class="sch-corner-label font-mono">SUDUT BIRU</span>
                <strong class="sch-fighter-name">
                  {{ participantMap[m.athlete_blue_id]?.name || 'Pesilat Biru' }}
                </strong>
                <span class="sch-fighter-club font-mono">
                  {{ participantMap[m.athlete_blue_id]?.club || 'Perguruan' }}
                </span>
              </div>
              <span
                v-if="m.winner_id && m.winner_id === m.athlete_blue_id"
                class="sch-win-tag font-mono"
                >MENANG</span
              >
            </div>
          </div>
        </div>

        <!-- Footer Operations -->
        <footer class="sch-card__footer font-mono">
          <!-- Control Buttons based on status -->
          <div v-if="canControlMatch" class="sch-footer__ops-btns">
            <!-- If idle / ready -->
            <template v-if="m.status === 'idle' || m.status === 'ready'">
              <button
                type="button"
                class="sch-btn-action sch-btn--load"
                :disabled="actionLoadingId === m.id"
                @click="handleLoadToArena(m)"
              >
                Muat ke Gelanggang
              </button>
              <button
                type="button"
                class="sch-btn-action sch-btn--start"
                :disabled="actionLoadingId === m.id"
                @click="handleStartMatch(m)"
              >
                Mulai Partai
              </button>
            </template>

            <!-- If running -->
            <template v-else-if="m.status === 'running'">
              <button
                type="button"
                class="sch-btn-action sch-btn--pause"
                :disabled="actionLoadingId === m.id"
                @click="handlePauseMatch(m)"
              >
                Jeda (Pause)
              </button>
              <button
                type="button"
                class="sch-btn-action sch-btn--finish"
                :disabled="actionLoadingId === m.id"
                @click="handleFinishMatch(m)"
              >
                Selesaikan Partai
              </button>
            </template>

            <!-- If paused -->
            <template v-else-if="m.status === 'paused'">
              <button
                type="button"
                class="sch-btn-action sch-btn--start"
                :disabled="actionLoadingId === m.id"
                @click="handleResumeMatch(m)"
              >
                Lanjutkan Partai
              </button>
              <button
                type="button"
                class="sch-btn-action sch-btn--finish"
                :disabled="actionLoadingId === m.id"
                @click="handleFinishMatch(m)"
              >
                Selesaikan Partai
              </button>
            </template>

            <!-- If finished -->
            <template v-else>
              <div class="sch-finished-badge">
                <svg
                  width="13"
                  height="13"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2.5"
                >
                  <polyline points="20 6 9 17 4 12" />
                </svg>
                <span>Partai Resmi Selesai</span>
              </div>
            </template>
          </div>

          <!-- Quick Navigation Links -->
          <div class="sch-footer__links">
            <button
              type="button"
              class="sch-link-btn"
              title="Lihat skor per babak dan riwayat kejadian"
              data-test="match-scores"
              @click="scoreMatch = m"
            >
              Rincian Skor
            </button>
            <button
              type="button"
              class="sch-link-btn"
              title="Buka Scoreboard Gelanggang"
              @click="openScoreboard(m.arena_id)"
            >
              Layar Scoreboard
            </button>
            <button
              v-if="canControlMatch"
              type="button"
              class="sch-link-btn"
              title="Buka Konsol Wasit"
              @click="openRefereeConsole"
            >
              Konsol Wasit
            </button>
          </div>
        </footer>
      </article>
    </div>

    <CreateMatchDialog
      :open="isCreateOpen"
      :competition-id="activeCompId"
      :participants="participants"
      :arenas="arenas"
      @close="isCreateOpen = false"
      @created="handleMatchCreated"
    />

    <MatchScoreDialog
      :open="!!scoreMatch"
      :competition-id="activeCompId"
      :match="scoreMatch"
      :red-name="participantMap[scoreMatch?.athlete_red_id ?? '']?.name || 'Pesilat Merah'"
      :blue-name="participantMap[scoreMatch?.athlete_blue_id ?? '']?.name || 'Pesilat Biru'"
      :arena-name="arenaMap[scoreMatch?.arena_id ?? '']"
      @close="scoreMatch = null"
    />
  </div>
</template>

<style scoped>
.sch-ops {
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: 100%;
  color: var(--text-main, #f8fafc);
  box-sizing: border-box;
}

/* Toast */
.sch-toast {
  position: fixed;
  top: 24px;
  right: 24px;
  z-index: 1000;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 18px;
  border-radius: var(--radius-md, 8px);
  background-color: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.45);
}
.sch-toast--success {
  border-left: 4px solid var(--color-emerald-500, #10b981);
}
.sch-toast--danger {
  border-left: 4px solid var(--color-primary-600, #dc2626);
}
.sch-toast--info {
  border-left: 4px solid var(--color-secondary-600, #2563eb);
}
.sch-toast__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: currentColor;
}
.sch-toast__content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.sch-toast__title {
  font-size: 0.875rem;
  font-weight: 700;
}
.sch-toast__msg {
  font-size: 0.8125rem;
  color: var(--color-neutral-400, #94a3b8);
}
.sch-toast__close {
  background: none;
  border: none;
  color: var(--color-neutral-500, #64748b);
  cursor: pointer;
  font-size: 1rem;
}
.sch-fade-enter-active,
.sch-fade-leave-active {
  transition:
    opacity 0.2s ease,
    transform 0.2s ease;
}
.sch-fade-enter-from,
.sch-fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* Header */
.sch-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
  padding: 18px 24px;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #1e293b);
  border-radius: var(--radius-lg, 12px);
}
.sch-header__left {
  display: flex;
  align-items: center;
  gap: 16px;
}
.sch-header__icon-box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 10px;
  background: rgba(16, 185, 129, 0.12);
  color: var(--color-emerald-400, #34d399);
  border: 1px solid rgba(16, 185, 129, 0.25);
}
.sch-header__sub {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--color-neutral-400, #94a3b8);
  letter-spacing: 0.05em;
  margin-bottom: 2px;
}
.sch-bullet {
  color: var(--color-neutral-600, #475569);
}
.sch-tag {
  color: var(--color-primary-400, #f87171);
}
.sch-pulse-wrapper {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--color-emerald-400, #34d399);
}
.sch-pulse-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-emerald-400, #34d399);
  box-shadow: 0 0 8px var(--color-emerald-400, #34d399);
  animation: pulse-dot 1.8s infinite;
}
@keyframes pulse-dot {
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
.sch-header__title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.5rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #ffffff;
  line-height: 1.2;
}
.sch-header__actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

/* Metrics */
.sch-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 12px;
}
.sch-metric-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 14px 18px;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #1e293b);
  border-radius: var(--radius-md, 10px);
  overflow: hidden;
}
.sch-metric-card--running {
  border-left: 3px solid var(--color-emerald-500, #10b981);
}
.sch-metric-card--ready {
  border-left: 3px solid #f59e0b;
}
.sch-metric-card--finished {
  border-left: 3px solid var(--color-secondary-600, #2563eb);
}
.sch-metric-corner-tag {
  position: absolute;
  top: 10px;
  right: 12px;
  font-size: 0.625rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--color-neutral-500, #64748b);
}
.sch-metric-label {
  font-family: var(--font-mono, monospace);
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--color-neutral-400, #94a3b8);
}
.sch-metric-value-row {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-top: 2px;
}
.sch-metric-val {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.875rem;
  font-weight: 800;
  color: #ffffff;
  line-height: 1;
}
.sch-val-emerald {
  color: #34d399;
}
.sch-val-amber {
  color: #fbbf24;
}
.sch-val-blue {
  color: #93c5fd;
}
.sch-metric-badge {
  font-size: 0.6875rem;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--color-neutral-400, #94a3b8);
}
.sch-badge-emerald {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
}
.sch-badge-amber {
  background: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
}
.sch-badge-blue {
  background: rgba(37, 99, 235, 0.15);
  color: #60a5fa;
}
.sch-metric-desc {
  font-size: 0.75rem;
  color: var(--color-neutral-500, #64748b);
  margin-top: 2px;
}

/* Toolbar */
.sch-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 14px;
  padding: 12px 16px;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #1e293b);
  border-radius: var(--radius-md, 10px);
}
.sch-toolbar__tabs {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.sch-tab-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 7px 14px;
  border-radius: 6px;
  background: transparent;
  border: 1px solid transparent;
  color: var(--color-neutral-400, #94a3b8);
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}
.sch-tab-btn:hover {
  background: rgba(255, 255, 255, 0.05);
  color: #ffffff;
}
.sch-tab-btn--active {
  background: rgba(255, 255, 255, 0.1);
  color: #ffffff;
  border-color: rgba(255, 255, 255, 0.15);
}
.sch-tab-btn--active-running {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
  border-color: rgba(16, 185, 129, 0.4);
}
.sch-tab-btn--active-ready {
  background: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
  border-color: rgba(245, 158, 11, 0.4);
}
.sch-tab-btn--active-finished {
  background: rgba(37, 99, 235, 0.15);
  color: #93c5fd;
  border-color: rgba(37, 99, 235, 0.4);
}
.sch-tab-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}
.sch-dot-emerald {
  background-color: var(--color-emerald-400, #34d399);
}
.sch-dot-amber {
  background-color: #f59e0b;
}
.sch-dot-blue {
  background-color: var(--color-secondary-500, #3b82f6);
}

.sch-toolbar__search-group {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  max-width: 540px;
  justify-content: flex-end;
}
.sch-select {
  padding: 8px 12px;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  color: var(--color-neutral-300, #cbd5e1);
  font-size: 0.75rem;
  outline: none;
  cursor: pointer;
}
.sch-search-box {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 200px;
}
.sch-search-icon {
  position: absolute;
  left: 10px;
  color: var(--color-neutral-500, #64748b);
  pointer-events: none;
}
.sch-search-input {
  width: 100%;
  padding: 8px 32px 8px 34px;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  color: #ffffff;
  font-size: 0.75rem;
  outline: none;
}
.sch-search-input:focus {
  border-color: var(--color-emerald-400, #34d399);
}
.sch-search-clear {
  position: absolute;
  right: 10px;
  background: none;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  cursor: pointer;
  font-size: 0.8125rem;
}

/* Grid */
.sch-grid {
  display: grid;
  grid-template-columns: repeat(1, minmax(0, 1fr));
  gap: 16px;
}
@container (min-width: 900px) {
  .sch-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

/* Match Card */
.sch-card {
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #1e293b);
  border-radius: var(--radius-md, 10px);
  overflow: hidden;
  transition:
    transform 0.15s ease,
    border-color 0.15s ease,
    box-shadow 0.15s ease;
}
.sch-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.35);
}
.sch-card__accent {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
}
.sch-accent--running {
  background: linear-gradient(90deg, #059669, #10b981);
}
.sch-accent--ready,
.sch-accent--idle,
.sch-accent--paused {
  background: linear-gradient(90deg, #d97706, #f59e0b);
}
.sch-accent--finished {
  background: linear-gradient(90deg, #2563eb, #3b82f6);
}

.sch-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 18px 10px 18px;
}
.sch-card__tag-group {
  display: flex;
  align-items: center;
  gap: 8px;
}
.sch-partai-badge {
  font-size: 0.6875rem;
  font-weight: 800;
  padding: 2px 8px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.08);
  color: #ffffff;
}
.sch-arena-tag {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--color-emerald-400, #34d399);
}

.sch-status-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.625rem;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 4px;
}
.sch-status-pill--running {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
  border: 1px solid rgba(16, 185, 129, 0.3);
}
.sch-status-pill--ready,
.sch-status-pill--idle,
.sch-status-pill--paused {
  background: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
  border: 1px solid rgba(245, 158, 11, 0.3);
}
.sch-status-pill--finished {
  background: rgba(37, 99, 235, 0.15);
  color: #93c5fd;
  border: 1px solid rgba(37, 99, 235, 0.3);
}
.sch-pulse-dot-live {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: var(--color-emerald-400, #34d399);
  box-shadow: 0 0 6px var(--color-emerald-400, #34d399);
  animation: pulse-dot 1.5s infinite;
}

/* Matchup Grid */
.sch-card__body {
  padding: 0 18px 14px 18px;
}
.sch-matchup-grid {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 12px;
}
.sch-fighter-box {
  position: relative;
  display: flex;
  flex-direction: column;
  padding: 10px 12px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 8px;
  min-height: 80px;
}
.sch-corner-bar {
  position: absolute;
  top: 0;
  left: 0;
  bottom: 0;
  width: 4px;
  border-radius: 8px 0 0 8px;
}
.sch-bar-red {
  background-color: #dc2626;
}
.sch-bar-blue {
  background-color: #2563eb;
}
.sch-fighter-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-left: 6px;
}
.sch-corner-label {
  font-size: 0.5625rem;
  font-weight: 700;
  color: var(--color-neutral-400, #94a3b8);
  letter-spacing: 0.05em;
}
.sch-fighter-name {
  font-size: 0.9375rem;
  font-weight: 700;
  color: #ffffff;
  line-height: 1.25;
}
.sch-fighter-club {
  font-size: 0.6875rem;
  color: var(--color-neutral-400, #94a3b8);
}
.sch-fighter-box--winner {
  border-color: rgba(16, 185, 129, 0.4);
  background: rgba(16, 185, 129, 0.08);
}
.sch-win-tag {
  align-self: flex-start;
  margin-top: auto;
  margin-left: 6px;
  font-size: 0.5625rem;
  font-weight: 800;
  padding: 1px 6px;
  border-radius: 3px;
  background: rgba(16, 185, 129, 0.25);
  color: #34d399;
}

.sch-vs-badge {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
}
.sch-vs-text {
  font-size: 1rem;
  font-weight: 900;
  color: var(--color-neutral-400, #94a3b8);
  letter-spacing: 0.08em;
}
.sch-round-sub {
  font-size: 0.5625rem;
  color: var(--color-neutral-500, #64748b);
  font-weight: 700;
}

/* Footer */
.sch-card__footer {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 18px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(0, 0, 0, 0.15);
}
.sch-footer__ops-btns {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.sch-btn-action {
  flex: 1;
  padding: 7px 12px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
  border: 1px solid transparent;
  transition: all 0.15s ease;
}
.sch-btn--load {
  background: rgba(255, 255, 255, 0.08);
  color: #ffffff;
  border-color: rgba(255, 255, 255, 0.15);
}
.sch-btn--load:hover {
  background: rgba(255, 255, 255, 0.15);
}
.sch-btn--start {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
  border-color: rgba(16, 185, 129, 0.35);
}
.sch-btn--start:hover {
  background: rgba(16, 185, 129, 0.3);
  color: #ffffff;
}
.sch-btn--pause {
  background: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
  border-color: rgba(245, 158, 11, 0.35);
}
.sch-btn--pause:hover {
  background: rgba(245, 158, 11, 0.3);
  color: #ffffff;
}
.sch-btn--finish {
  background: rgba(37, 99, 235, 0.15);
  color: #93c5fd;
  border-color: rgba(37, 99, 235, 0.35);
}
.sch-btn--finish:hover {
  background: rgba(37, 99, 235, 0.3);
  color: #ffffff;
}

.sch-finished-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.6875rem;
  color: var(--color-emerald-400, #34d399);
  font-weight: 700;
}

.sch-footer__links {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
}
.sch-link-btn {
  background: none;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  font-size: 0.6875rem;
  font-weight: 600;
  cursor: pointer;
  text-decoration: underline;
}
.sch-link-btn:hover {
  color: #ffffff;
}

/* Skeleton */
.sch-skeleton-card {
  height: 220px;
  background: rgba(19, 29, 51, 0.6);
  border: 1px solid var(--border-card, #1e293b);
  border-radius: 10px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.skeleton-line {
  background: linear-gradient(
    90deg,
    rgba(255, 255, 255, 0.04) 25%,
    rgba(255, 255, 255, 0.08) 50%,
    rgba(255, 255, 255, 0.04) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 4px;
}
.skeleton-header {
  width: 40%;
  height: 20px;
}
.skeleton-matchup {
  width: 100%;
  height: 80px;
}
.skeleton-btn {
  width: 100%;
  height: 32px;
  margin-top: auto;
}
@keyframes shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

/* Empty State */
.sch-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 56px 24px;
  background: var(--bg-card, #131d33);
  border: 1px dashed var(--border-card, #334155);
  border-radius: 12px;
}
.sch-empty-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.05);
  color: var(--color-neutral-400, #94a3b8);
  margin-bottom: 16px;
}
.sch-empty-icon--error {
  background: rgba(220, 38, 38, 0.15);
  color: var(--color-primary-400, #f87171);
}
.sch-empty-title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.25rem;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 8px;
}
.sch-empty-desc {
  max-width: 520px;
  font-size: 0.875rem;
  color: var(--color-neutral-400, #94a3b8);
  line-height: 1.5;
  margin-bottom: 20px;
}
.sch-empty-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* Modal */
.sch-modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(4px);
}
.sch-modal {
  width: 100%;
  max-width: 560px;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
  overflow: hidden;
}
.sch-modal__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-card, #1e293b);
}
.sch-modal__tag {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--color-emerald-400, #34d399);
  margin-bottom: 4px;
}
.sch-modal__title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.25rem;
  font-weight: 800;
  color: #ffffff;
}
.sch-modal__close {
  background: none;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  cursor: pointer;
  font-size: 1.25rem;
}
.sch-modal__body {
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.sch-form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.sch-form-label {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--color-neutral-300, #cbd5e1);
}
.sch-req {
  color: var(--color-primary-500, #ef4444);
}
.sch-form-select {
  width: 100%;
  padding: 10px 14px;
  background: rgba(15, 23, 42, 0.8);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  color: #ffffff;
  font-size: 0.8125rem;
  outline: none;
  box-sizing: border-box;
}
.sch-form-select:focus {
  border-color: var(--color-emerald-400, #34d399);
}
.sch-form-error {
  font-size: 0.75rem;
  color: var(--color-primary-400, #f87171);
}

.sch-modal__footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 10px;
}
</style>
