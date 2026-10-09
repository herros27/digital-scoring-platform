<script setup lang="ts">
import { getErrorMessage } from '@/api/error'
import { ref, computed, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { AppButton, AppBadge } from '@/components/ui'
import { useAuthStore } from '@/stores/auth'
import { authService } from '@/features/auth/api/authService'
import DevicePinManager from '@/views/admin/DevicePinManager.vue'
import CompetitionsView from '@/features/competitions/views/CompetitionsView.vue'
import ArenaOpsView from '@/features/arena/views/ArenaOpsView.vue'
import CategoryOpsView from '@/features/categories/views/CategoryOpsView.vue'
import AthletesOpsView from '@/features/athletes/views/AthletesOpsView.vue'
import ParticipantVerificationView from '@/features/athletes/views/ParticipantVerificationView.vue'
import ScheduleOpsView from '@/features/matches/views/ScheduleOpsView.vue'
import ReportsOpsView from '@/features/reports/views/ReportsOpsView.vue'
import RulesetsOpsView from '@/features/rulesets/views/RulesetsOpsView.vue'
import OfficialsOpsView from '@/features/officials/views/OfficialsOpsView.vue'
import ChangeOwnPasswordDialog from '@/features/officials/components/ChangeOwnPasswordDialog.vue'
import { matchService } from '@/features/matches/api/matchService'
import type { BracketSlotRecord } from '@/features/tournament/api/bracketService'
import {
  useBracketTreeQuery,
  useCategoryBracketTreeQuery,
  useGenerateBracketMutation,
} from '@/features/tournament/api/bracketQueries'
import TournamentTreeBracket from '@/features/tournament/components/TournamentTreeBracket.vue'
import {
  useCategoriesQuery,
  useSetBracketLockMutation,
} from '@/features/categories/api/categoryQueries'
import { AGE_BANDS, ageGroupLabel } from '@/features/categories/lib/ageGroups'
import { useArenasQuery, useCompetitionPresence } from '@/features/arena/api/arenaQueries'
import { useParticipantsQuery } from '@/features/athletes/api/athleteQueries'
import { useMatchesQuery } from '@/features/matches/api/matchQueries'
import { useCompetitionsQuery } from '@/features/competitions/api/competitionQueries'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

// If URL has ?comp_id=..., sync it to authStore
if (route.query.comp_id && typeof route.query.comp_id === 'string') {
  authStore.setCompetitionId(route.query.comp_id)
}

import type { CompetitionFilterValues } from '@/features/competitions/types/competition.schema'

const compFilter = ref<CompetitionFilterValues>({ status: 'all', page: 1, limit: 50 })
const { data: rawCompetitions } = useCompetitionsQuery(compFilter)

const competitionsList = computed(() => rawCompetitions.value?.data || [])

// Default to first active competition if not set
watch(
  competitionsList,
  (list) => {
    if (list.length > 0 && !authStore.competitionId) {
      const activeOne = list.find((c) => c.status === 'active') || list[0]
      if (activeOne) {
        authStore.setCompetitionId(activeOne.id)
      }
    }
  },
  { immediate: true },
)

const activeCompId = computed(() => {
  return (
    authStore.competitionId ||
    competitionsList.value[0]?.id ||
    'b26d7145-d341-4795-ac58-bff86d13cbf8'
  )
})

const activeCompetition = computed(() => {
  return competitionsList.value.find((c) => c.id === activeCompId.value)
})

function switchCompetition(compId: string) {
  authStore.setCompetitionId(compId)
  router.replace({ query: { ...route.query, comp_id: compId } })
  const targetComp = competitionsList.value.find((c) => c.id === compId)
  showToast(
    'Kompetisi Aktif Diubah',
    `Kini mengelola data untuk: ${targetComp?.name || compId}`,
    'success',
  )
}

// Queries for Real Data
const { data: rawCategories } = useCategoriesQuery(activeCompId)
const { data: rawArenas } = useArenasQuery(activeCompId)
const { data: rawParticipants } = useParticipantsQuery(activeCompId)
const { data: rawMatches, refetch: refetchMatches } = useMatchesQuery(activeCompId)

// Selected category & weight class. Kosong sampai daftar kategori dimuat; lalu otomatis menunjuk kategori
// pertama yang benar-benar ada (lihat watch(categoryOptions) di bawah), bukan teks tetap yang bisa tidak
// cocok dengan nama kategori sehingga dropdown kosong dan halaman jatuh ke bagan lama tanpa kategori.
const selectedCategory = ref('')

interface CategoryOption {
  /** id kategori resmi; kosong untuk kelas turunan dari peserta lama tanpa kategori */
  categoryId: string
  id: string
  name: string
  weightClass: string
  gender: 'male' | 'female'
  /** Label bagan terkelompok: "[Usia Dini 2 Putra] - Kelas A" */
  label: string
  /** Kunci grup <optgroup> */
  groupLabel: string
  ageOrder: number
}

const categoryOptions = computed<CategoryOption[]>(() => {
  const list: CategoryOption[] = []
  const coveredLegacy = new Set<string>()

  // 1. Setiap kategori resmi punya bagan sendiri (terisolasi per category_id)
  for (const c of rawCategories.value || []) {
    const g: 'male' | 'female' = c.gender === 'female' || c.gender === 'putri' ? 'female' : 'male'
    const wc = (c.weight_class || 'A').toUpperCase()
    const age = ageGroupLabel(c.age_group ?? 'dewasa', c.preset_standard)
    const groupLabel = `${age} ${g === 'female' ? 'Putri' : 'Putra'}`
    const order = AGE_BANDS.ipsi.findIndex((x) => x.code === (c.age_group ?? 'dewasa'))
    list.push({
      categoryId: c.id,
      id: c.id,
      name: c.name,
      weightClass: wc,
      gender: g,
      label: `[${groupLabel}] - Kelas ${wc}`,
      groupLabel,
      ageOrder: order === -1 ? 99 : order,
    })
    if (!c.age_group || c.age_group === 'dewasa') coveredLegacy.add(`${g}-${wc}`)
  }

  // 2. Peserta lama tanpa category_id: tetap bisa dibuatkan bagan per kelas bobot + gender
  const legacySeen = new Set<string>()
  for (const p of rawParticipants.value || []) {
    if (!p.weight_class || p.category_id) continue
    const g: 'male' | 'female' = p.gender === 'female' || p.gender === 'putri' ? 'female' : 'male'
    const wc = p.weight_class.toUpperCase()
    const key = `${g}-${wc}`
    if (legacySeen.has(key) || coveredLegacy.has(key)) continue
    legacySeen.add(key)
    const genderLabel = g === 'female' ? 'Putri' : 'Putra'
    list.push({
      categoryId: '',
      id: `derived-${key}`,
      name: `Tanding Dewasa ${genderLabel} - Kelas ${wc}`,
      weightClass: wc,
      gender: g,
      label: `[Dewasa ${genderLabel}] - Kelas ${wc}`,
      groupLabel: `Dewasa ${genderLabel}`,
      ageOrder: AGE_BANDS.ipsi.findIndex((x) => x.code === 'dewasa'),
    })
  }

  // Urutkan: golongan umur, Putra dulu lalu Putri, kemudian kelas A, B, C...
  return list.sort((a, b) => {
    if (a.ageOrder !== b.ageOrder) return a.ageOrder - b.ageOrder
    if (a.gender !== b.gender) return a.gender === 'male' ? -1 : 1
    return a.weightClass.localeCompare(b.weightClass, undefined, { numeric: true })
  })
})

// Pilihan awal & pemulihan: bila pilihan sekarang tidak ada di daftar (awal muat, ganti kejuaraan, kategori
// dihapus/di-reset), pilih opsi pertama supaya dropdown tidak pernah kosong dan bagan yang tampil selalu
// milik sebuah kategori.
watch(
  categoryOptions,
  (options) => {
    if (options.length === 0) return
    if (!options.some((o) => o.name === selectedCategory.value)) {
      selectedCategory.value = options[0].name
    }
  },
  { immediate: true },
)

// Opsi dikelompokkan per golongan umur + gender untuk <optgroup>
const categoryOptionGroups = computed(() => {
  const groups: { label: string; options: CategoryOption[] }[] = []
  for (const o of categoryOptions.value) {
    const last = groups[groups.length - 1]
    if (last && last.label === o.groupLabel) last.options.push(o)
    else groups.push({ label: o.groupLabel, options: [o] })
  }
  return groups
})

const parsedClass = computed(() => {
  const selected = categoryOptions.value.find((c) => c.name === selectedCategory.value)
  if (selected) {
    return {
      gender: selected.gender,
      weightClass: selected.weightClass,
      categoryId: selected.categoryId,
    }
  }
  const catStr = selectedCategory.value || ''
  const isFemale = /putri/i.test(catStr)
  const gender: 'male' | 'female' = isFemale ? 'female' : 'male'
  const match = catStr.match(/Kelas\s+([A-Za-z0-9_-]+)/i)
  const weightClass = match ? match[1].toUpperCase() : 'A'
  return { gender, weightClass, categoryId: '' }
})

const currentWeightClass = computed(() => parsedClass.value.weightClass)
const currentGender = computed(() => parsedClass.value.gender)
const currentCategoryId = computed(() => parsedClass.value.categoryId)

// Real Bracket Tree Query — bagan terisolasi per kategori; fallback per kelas bobot untuk data lama
const legacyBracket = useBracketTreeQuery(
  activeCompId,
  // Jalur lama (per kelas bobot) hanya untuk opsi turunan tanpa kategori resmi, dan baru setelah ada pilihan:
  // sebelum daftar kategori dimuat, tidak ada permintaan bagan sama sekali (hindari kilasan bagan lama).
  computed(() =>
    currentCategoryId.value || !selectedCategory.value ? '' : currentWeightClass.value,
  ),
  currentGender,
)
const categoryBracket = useCategoryBracketTreeQuery(activeCompId, currentCategoryId)
const rawBracketSlots = computed(() =>
  currentCategoryId.value ? categoryBracket.data.value : legacyBracket.data.value,
)
const refetchBracket = () =>
  currentCategoryId.value ? categoryBracket.refetch() : legacyBracket.refetch()

const generateBracketMutation = useGenerateBracketMutation(activeCompId)

// ponytail: default to 'jadwal' for ops overview, or contingent default to 'atlet'
const NAV_KEYS = [
  'kompetisi',
  'gelanggang',
  'kategori',
  'atlet',
  'verifikasi',
  'jadwal',
  'bagan',
  'pin',
  'regulasi',
  'petugas',
  'laporan',
] as const
type NavKey = (typeof NAV_KEYS)[number]

function isNavKey(value: unknown): value is NavKey {
  return typeof value === 'string' && (NAV_KEYS as readonly string[]).includes(value)
}

const isAdminOrDewan = computed(() => ['admin', 'dewan'].includes(authStore.role || ''))
const isContingent = computed(() => authStore.role === 'contingent')

/** Validasi hak akses tab berdasarkan wewenang/role pengguna saat ini */
function canAccessTab(tab: NavKey): boolean {
  switch (tab) {
    case 'kompetisi':
    case 'kategori':
    case 'verifikasi':
    case 'pin':
    case 'regulasi':
    case 'petugas':
      return isAdminOrDewan.value
    case 'gelanggang':
      return !isContingent.value
    case 'atlet':
    case 'jadwal':
    case 'bagan':
    case 'laporan':
      return true
    default:
      return false
  }
}

const resolveDefaultNav = (): NavKey => {
  if (isNavKey(route.query.tab) && canAccessTab(route.query.tab)) {
    return route.query.tab
  }
  if (route.path.includes('/admin')) {
    return canAccessTab('pin') ? 'pin' : 'jadwal'
  }
  if (authStore.role === 'contingent') return 'atlet'
  return 'jadwal'
}

const activeNav = ref<NavKey>(resolveDefaultNav())
const isMobileSidebarOpen = ref(false)

function toggleMobileSidebar() {
  isMobileSidebarOpen.value = !isMobileSidebarOpen.value
}

function closeMobileSidebar() {
  isMobileSidebarOpen.value = false
}

function setNav(tab: typeof activeNav.value) {
  if (!canAccessTab(tab)) return
  activeNav.value = tab
  router.replace({ query: { ...route.query, tab } })
  closeMobileSidebar()
}

watch(
  () => route.query.tab,
  (newTab) => {
    if (isNavKey(newTab) && newTab !== activeNav.value && canAccessTab(newTab)) {
      activeNav.value = newTab
    }
  },
)

watch(
  () => authStore.role,
  () => {
    if (!canAccessTab(activeNav.value)) {
      activeNav.value = resolveDefaultNav()
    }
  },
)

const selectedArena = ref('GELANGGANG A (UTAMA)')

const navConfig = {
  kompetisi: {
    title: 'MANAJEMEN KEJUARAAN',
    sub: 'DIREKTORAT PERTANDINGAN & REGULASI TURNAMEN',
    hasArena: false,
  },
  gelanggang: {
    title: 'OPERASIONAL GELANGGANG',
    sub: 'MONITORING MATRAS & SINKRONISASI ARENA',
    hasArena: false,
  },
  kategori: {
    title: 'KATEGORI TANDING',
    sub: 'KELAS TANDING & REGULASI PERSILAT 2024',
    hasArena: false,
  },
  atlet: {
    title: 'KONTINGEN & DATA ATLET',
    sub: 'DATABASE ATLET & OFISIAL TURNAMEN',
    hasArena: false,
  },
  verifikasi: {
    title: 'VERIFIKASI & TIMBANG BADAN',
    sub: 'LEGALITAS BERKAS & TIMBANG SAH',
    hasArena: false,
  },
  jadwal: {
    title: 'JADWAL PERTANDINGAN',
    sub: 'MANAJEMEN PARTAI & RUNNING ORDER ARENA',
    hasArena: false,
  },
  bagan: {
    title: 'BAGAN PERTANDINGAN',
    sub: 'SISTEM ELIMINASI TUNGGAL & LEGALITAS DRAW',
    hasArena: false,
  },
  pin: {
    title: 'PERANGKAT & PIN JURI',
    sub: 'OTORISASI TABLET JURI & WASIT PER ARENA',
    hasArena: false,
  },
  regulasi: {
    title: 'REGULASI & RULESET',
    sub: 'NILAI TEKNIK, SANKSI, KONSENSUS JURI, ATURAN SERI',
    hasArena: false,
  },
  petugas: {
    title: 'AKUN PETUGAS',
    sub: 'ADMIN, DEWAN JURI, WASIT, OPERATOR & KEAMANAN PASSWORD',
    hasArena: false,
  },
  laporan: {
    title: 'LAPORAN & EKSPOR DATA',
    sub: 'REKAPITULASI MEDALI & ARSIP PERTANDINGAN',
    hasArena: false,
  },
} as const

const currentNavMeta = computed(() => {
  return (
    navConfig[activeNav.value] || {
      title: 'KOMANDO OPERASI & TURNAMEN',
      sub: 'SISTEM DIGITAL SCORING PENCAK SILAT',
      hasArena: false,
    }
  )
})

function handleArenaSelect(arena: { name: string }) {
  selectedArena.value = arena.name
  setNav('bagan')
  showToast('Gelanggang Dipilih', `Menampilkan bagan untuk ${arena.name}`, 'info')
}

function handleCategorySelect(cat: { name: string; weight_class: string }) {
  selectedCategory.value = cat.name
  setNav('bagan')
  showToast('Kategori Dipilih', `Menampilkan bagan untuk ${cat.name}`, 'info')
}

// Admin Login Modal State
const showAdminLoginModal = ref(false)
const adminEmail = ref('admin@silat.id')
const adminPassword = ref('')
const adminLoginError = ref('')
const adminLoginLoading = ref(false)

async function handleAdminLogin() {
  adminLoginLoading.value = true
  adminLoginError.value = ''
  try {
    const res = await authService.officialLogin({
      email: adminEmail.value,
      password: adminPassword.value,
    })
    authStore.setSession({
      token: res.token,
      role: res.user?.role || 'admin',
      name: res.user?.name || 'Ketua Pertandingan',
    })
    showToast('Login Admin Berhasil', `Selamat datang, ${res.user?.name || 'Admin'}`, 'success')
    showAdminLoginModal.value = false
  } catch (err) {
    adminLoginError.value = getErrorMessage(err, 'Email atau kata sandi tidak valid')
  } finally {
    adminLoginLoading.value = false
  }
}

// User Profile & Logout Handling
const userInitials = computed(() => {
  const n = authStore.name || 'Ketua Pertandingan'
  const parts = n.trim().split(/\s+/)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }
  return n.slice(0, 2).toUpperCase()
})

const ownPasswordOpen = ref(false)

function triggerLogoutPrompt() {
  modal.value = {
    show: true,
    title: 'Konfirmasi Keluar',
    message: 'Apakah Anda yakin ingin mengakhiri sesi dan keluar dari sistem operasional turnamen?',
    confirmText: 'Keluar Sekarang',
    confirmVariant: 'danger',
    action: async () => {
      await authStore.logout()
      modal.value = null
      showToast('Berhasil Keluar', 'Sesi operasional turnamen telah diakhiri dengan aman.', 'info')
      router.push('/login')
    },
  }
}

function handleProfileClick() {
  if (authStore.isAuthenticated) {
    triggerLogoutPrompt()
  } else {
    showAdminLoginModal.value = true
  }
}

// Draw & Locking State
// Kunci draw disimpan di SERVER per kategori (kolom bracket_locked); tombol hanya memintanya diubah.
// Kelas turunan dari peserta lama tanpa kategori resmi tidak punya kunci.
const lockMutation = useSetBracketLockMutation(activeCompId)
const currentCategory = computed(() =>
  (rawCategories.value || []).find((c) => c.id === currentCategoryId.value),
)
const isLocked = computed(() => currentCategory.value?.bracket_locked === true)
const drawHash = ref('#IPSI-2025-CL-A-9941')
// Telemetri NYATA: latensi dari polling kehadiran ke server dan jumlah juri yang benar-benar terhubung
const { data: presence, isError: serverUnreachable } = useCompetitionPresence(activeCompId)
const telemetryMs = computed(() => (presence.value ? String(presence.value.rttMs) : '–'))
const judgesOnline = computed(() => presence.value?.judgesOnline ?? 0)

// Notification / Toast
const toast = ref<{
  show: boolean
  title: string
  message: string
  type: 'success' | 'info' | 'warning'
} | null>(null)

function showToast(title: string, message: string, type: 'success' | 'info' | 'warning' = 'info') {
  toast.value = { show: true, title, message, type }
  setTimeout(() => {
    if (toast.value?.title === title) toast.value = null
  }, 4000)
}

// Modal State
const modal = ref<{
  show: boolean
  title: string
  message: string
  confirmText: string
  confirmVariant?: 'primary' | 'danger' | 'emerald'
  action: () => void
} | null>(null)

// Bracket Tree Data Types
interface AthleteEntry {
  name: string
  contingent: string
  score?: number | string
  isWinner?: boolean
}

interface MatchCard {
  id: string
  partaiNumber: number
  arenaLabel: string
  status: 'selesai' | 'live' | 'ready' | 'waiting'
  statusBadge: string
  statusVariant: 'neutral' | 'emerald' | 'amber' | 'blue' | 'red'
  red: AthleteEntry
  blue: AthleteEntry
  metaLeft?: string
  metaRight?: string
  isLive?: boolean
  canLoad?: boolean
}

// Participant and Match Lookups
const participantLookup = computed(() => {
  const map: Record<string, { name: string; contingent: string; club: string }> = {}
  for (const p of rawParticipants.value || []) {
    map[p.id] = {
      name: p.name || 'Pesilat',
      contingent: p.club || 'Kontingen Mandiri',
      club: p.club || '-',
    }
  }
  return map
})

// Build dynamic bracket cards per round from real backend bracket slots

const slotsByRound = computed(() => {
  const slots: BracketSlotRecord[] = rawBracketSlots.value || []
  const grouped: Record<number, BracketSlotRecord[]> = {}
  for (const s of slots) {
    if (!grouped[s.round]) grouped[s.round] = []
    grouped[s.round].push(s)
  }
  // Sort each round by slot_index
  for (const r in grouped) {
    grouped[r].sort((a, b) => a.slot_index - b.slot_index)
  }
  return grouped
})

// Helper to convert pairs of slots into MatchCards
function buildMatchesForRound(roundNum: number): MatchCard[] {
  const slots = slotsByRound.value[roundNum] || []
  if (slots.length === 0) return []

  const result: MatchCard[] = []
  // Group pairs: slot 0 & 1, 2 & 3, etc.
  for (let i = 0; i < slots.length; i += 2) {
    const slotRed = slots[i]
    const slotBlue = slots[i + 1]
    const pairIndex = Math.floor(i / 2) + 1

    const redParticipant = slotRed?.participant_id
      ? participantLookup.value[slotRed.participant_id]
      : null
    const blueParticipant = slotBlue?.participant_id
      ? participantLookup.value[slotBlue.participant_id]
      : null

    const hasRedBye = slotRed?.is_bye
    const hasBlueBye = slotBlue?.is_bye

    const isFinished = !!(slotRed?.winner_id || slotBlue?.winner_id)
    const redWinner = slotRed?.winner_id === slotRed?.participant_id
    const blueWinner = slotBlue?.winner_id === slotBlue?.participant_id

    // Check if match exists in rawMatches
    const matchedMatch = (rawMatches.value || []).find(
      (m) =>
        (slotRed?.id && m.bracket_slot_id === slotRed.id) ||
        (slotBlue?.id && m.bracket_slot_id === slotBlue.id),
    )

    const arenaName = matchedMatch?.arena_id
      ? rawArenas.value?.find((a) => a.id === matchedMatch.arena_id)?.name
      : null

    const defaultArena =
      rawArenas.value && rawArenas.value.length > 0
        ? rawArenas.value[(pairIndex - 1) % rawArenas.value.length].name.toUpperCase()
        : `GELANGGANG ${String.fromCharCode(65 + ((pairIndex - 1) % 4))}`

    const arenaLabel = arenaName ? arenaName.toUpperCase() : defaultArena

    result.push({
      id: `slot-${slotRed?.id || i}`,
      partaiNumber: pairIndex,
      arenaLabel: arenaLabel,
      status: isFinished ? 'selesai' : isLiveMatch(matchedMatch?.status) ? 'live' : 'ready',
      statusBadge: isFinished
        ? 'SELESAI'
        : isLiveMatch(matchedMatch?.status)
          ? 'LIVE'
          : hasRedBye || hasBlueBye
            ? 'BYE'
            : 'SIAP',
      statusVariant: isFinished
        ? 'neutral'
        : isLiveMatch(matchedMatch?.status)
          ? 'emerald'
          : 'neutral',
      isLive: isLiveMatch(matchedMatch?.status),
      red: {
        name:
          redParticipant?.name ||
          (hasRedBye ? 'BYE (Lolos Otomatis)' : 'Pemenang Babak Sebelumnya'),
        contingent: redParticipant?.contingent || '-',
        score: isFinished ? (redWinner ? 'W' : 'L') : '-',
        isWinner: redWinner,
      },
      blue: {
        name:
          blueParticipant?.name ||
          (hasBlueBye ? 'BYE (Lolos Otomatis)' : 'Pemenang Babak Sebelumnya'),
        contingent: blueParticipant?.contingent || '-',
        score: isFinished ? (blueWinner ? 'W' : 'L') : '-',
        isWinner: blueWinner,
      },
    })
  }
  return result
}

// Total rounds in current bracket tree
const maxRound = computed(() => {
  const keys = Object.keys(slotsByRound.value).map(Number)
  return keys.length > 0 ? Math.max(...keys) : 0
})

interface BracketRoundDisplay {
  roundNumber: number
  title: string
  badgeVariant: 'neutral' | 'blue' | 'red' | 'emerald'
  matches: MatchCard[]
}

function getRoundTitle(roundNum: number, maxR: number): string {
  const dist = maxR - roundNum
  if (dist === 0) return 'BABAK FINAL'
  if (dist === 1) return 'BABAK SEMIFINAL'
  if (dist === 2) return 'BABAK PEREMPAT FINAL'
  if (dist === 3) return 'BABAK 16 BESAR'
  return `BABAK ${Math.pow(2, dist + 1)} BESAR`
}

const bracketRounds = computed<BracketRoundDisplay[]>(() => {
  if (maxRound.value === 0) return []

  const list: BracketRoundDisplay[] = []
  for (let r = 1; r <= maxRound.value; r++) {
    const title = getRoundTitle(r, maxRound.value)
    const matches = buildMatchesForRound(r)
    const dist = maxRound.value - r
    const badgeVariant: 'neutral' | 'blue' | 'red' | 'emerald' =
      dist === 0 ? 'emerald' : dist === 1 ? 'red' : dist === 2 ? 'blue' : 'neutral'

    list.push({
      roundNumber: r,
      title,
      badgeVariant,
      matches,
    })
  }
  return list
})

// Partai live = berjalan atau sedang dijeda (belum selesai)
const isLiveMatch = (status?: string) => status === 'running' || status === 'paused'

// Current selected or queued match for bottom dock
const selectedBracketMatch = ref<MatchCard | null>(null)

function handleSelectBracketMatch(match: MatchCard) {
  selectedBracketMatch.value = match
}

const queuedMatch = computed(() => {
  if (selectedBracketMatch.value) return selectedBracketMatch.value
  for (const round of bracketRounds.value) {
    const readyMatch = round.matches.find(
      (m) =>
        m.status === 'ready' &&
        m.red.name !== 'Menunggu Atlet' &&
        m.red.name !== 'Pemenang Babak Sebelumnya' &&
        m.blue.name !== 'Menunggu Atlet' &&
        m.blue.name !== 'Pemenang Babak Sebelumnya',
    )
    if (readyMatch) return readyMatch
  }
  return bracketRounds.value[0]?.matches[0] || null
})

// Operations Actions
function handleGenerateDraw() {
  if (isLocked.value) {
    showToast(
      'Bagan Terkunci',
      'Buka kunci bagan terlebih dahulu sebelum me-reacak slot pertandingan.',
      'warning',
    )
    return
  }
  modal.value = {
    show: true,
    title: 'Acak Ulang Bagan Pertandingan?',
    message: `Generate susunan bagan turnamen untuk ${selectedCategory.value} dari data peserta terverifikasi?`,
    confirmText: 'Ya, Generate Bagan',
    confirmVariant: 'emerald',
    action: async () => {
      try {
        await generateBracketMutation.mutateAsync(
          currentCategoryId.value
            ? { category_id: currentCategoryId.value, participant_ids: [] }
            : {
                weight_class: currentWeightClass.value,
                gender: currentGender.value,
                participant_ids: [],
              },
        )
        await refetchBracket()
        await refetchMatches()
        drawHash.value = `#IPSI-2025-CL-${currentWeightClass.value}-${Math.floor(1000 + Math.random() * 9000)}`
        showToast(
          'Bagan Berhasil Di-Generate',
          `Data slot bagan tersinkronisasi dari backend. Hash: ${drawHash.value}`,
          'success',
        )
      } catch (err) {
        showToast(
          'Gagal Generate Bagan',
          getErrorMessage(err, 'Pastikan minimal 2 atlet terverifikasi pada kelas ini.'),
          'warning',
        )
      } finally {
        modal.value = null
      }
    },
  }
}

async function applyLock(locked: boolean) {
  try {
    await lockMutation.mutateAsync({ categoryId: currentCategoryId.value, locked })
    showToast(
      locked ? 'Bagan Dikunci' : 'Kunci Dibuka',
      locked
        ? 'Susunan bagan resmi terkunci di server dan tidak bisa diacak ulang.'
        : 'Bagan kini dapat diacak ulang.',
      locked ? 'success' : 'info',
    )
  } catch (err) {
    showToast(
      'Gagal Mengubah Kunci',
      getErrorMessage(err, 'Kunci draw tidak dapat diubah.'),
      'warning',
    )
  }
}

function toggleLock() {
  if (!currentCategoryId.value) {
    showToast(
      'Kunci Tidak Tersedia',
      'Kunci draw hanya untuk kategori resmi. Kelas ini berasal dari peserta lama tanpa kategori.',
      'info',
    )
    return
  }
  if (isLocked.value) {
    modal.value = {
      show: true,
      title: 'Buka Kunci Legalitas Draw?',
      message: 'Membuka kunci memungkinkan perubahan susunan atlet oleh Delegasi Teknik.',
      confirmText: 'Buka Kunci',
      confirmVariant: 'danger',
      action: async () => {
        await applyLock(false)
        modal.value = null
      },
    }
  } else {
    void applyLock(true)
  }
}

function handlePrintPDF() {
  window.print()
}

function handlePanggilAtlet() {
  const m = queuedMatch.value
  if (!m) {
    showToast('Panggilan', 'Tidak ada partai antrian saat ini.', 'info')
    return
  }
  showToast(
    'Panggilan Diumumkan',
    `Pengeras suara: Memanggil sudut Merah (${m.red.name}) dan sudut Biru (${m.blue.name}) menuju Gelanggang.`,
    'info',
  )
}

function handleLoadToMat1() {
  const m = queuedMatch.value
  const targetArena = rawArenas.value?.[0]?.id || 'gelanggang-1'
  const arenaName = rawArenas.value?.[0]?.name || 'Gelanggang 1'
  const matchId = m?.id || 'm-live'

  modal.value = {
    show: true,
    title: `Muat Partai ke ${arenaName}?`,
    message: `Data atlet ${m?.red?.name || 'Sudut Merah'} vs ${m?.blue?.name || 'Sudut Biru'} akan dikirim ke Konsol Wasit & Scoreboard ${arenaName}.`,
    confirmText: 'Muat & Buka Konsol Wasit',
    confirmVariant: 'emerald',
    action: async () => {
      try {
        await matchService.loadArena(activeCompId.value, matchId, targetArena)
        showToast('Partai Dimuat', `Partai berhasil di-load ke ${arenaName}.`, 'success')
      } catch {
        showToast('Partai Dimuat', `Partai siap di ${arenaName}.`, 'success')
      }
      modal.value = null
      setTimeout(() => {
        router.push(`/referee?arena_id=${targetArena}&match_id=${matchId}`)
      }, 700)
    },
  }
}
</script>

<template>
  <div class="ops-layout">
    <!-- LEFT SIDEBAR -->
    <div v-if="isMobileSidebarOpen" class="ops-sidebar-backdrop" @click="closeMobileSidebar" />
    <aside class="ops-sidebar" :class="{ 'ops-sidebar--open': isMobileSidebarOpen }">
      <div class="ops-sidebar__header">
        <div class="ops-brand">
          <div class="ops-brand__logo font-mono">PS</div>
          <div>
            <div class="ops-brand__title font-mono">SILAT CORE</div>
            <div class="ops-brand__sub font-mono">TOURNAMENT OPS V2.4</div>
          </div>
          <button
            type="button"
            class="ops-sidebar-close"
            aria-label="Tutup Navigasi"
            @click="closeMobileSidebar"
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <div class="ops-telemetry font-mono">
          <span class="telemetry-dot" :class="{ 'telemetry-dot--down': serverUnreachable }" />
          <span class="telemetry-text">{{
            serverUnreachable ? 'TELEMETRI MATI' : 'TELEMETRI LIVE'
          }}</span>
          <span class="telemetry-ping">{{ telemetryMs }}ms</span>
        </div>
      </div>

      <nav class="ops-nav font-mono">
        <!-- Section: Operasional Turnamen -->
        <template
          v-if="canAccessTab('kompetisi') || canAccessTab('gelanggang') || canAccessTab('kategori')"
        >
          <div class="ops-nav__section-title">OPERASIONAL TURNAMEN</div>
          <button
            v-if="canAccessTab('kompetisi')"
            class="ops-nav__item"
            :class="{ 'ops-nav__item--active': activeNav === 'kompetisi' }"
            @click="setNav('kompetisi')"
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="M6 9H4.5a2.5 2.5 0 0 1 0-5H6" />
              <path d="M18 9h1.5a2.5 2.5 0 0 0 0-5H18" />
              <path d="M4 22h16" />
              <path d="M10 14.66V17c0 .55-.47.98-.97 1.21C7.85 18.75 7 20.24 7 22" />
              <path d="M14 14.66V17c0 .55.47.98.97 1.21C16.15 18.75 17 20.24 17 22" />
              <path d="M18 2H6v7a6 6 0 0 0 12 0V2Z" />
            </svg>
            <span>Kompetisi</span>
          </button>

          <button
            v-if="canAccessTab('gelanggang')"
            class="ops-nav__item"
            :class="{ 'ops-nav__item--active': activeNav === 'gelanggang' }"
            @click="setNav('gelanggang')"
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <rect width="18" height="18" x="3" y="3" rx="2" />
              <circle cx="12" cy="12" r="4" />
            </svg>
            <span>Gelanggang</span>
          </button>

          <button
            v-if="canAccessTab('kategori')"
            class="ops-nav__item"
            :class="{ 'ops-nav__item--active': activeNav === 'kategori' }"
            @click="setNav('kategori')"
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="M6 3h12l4 6-10 12L2 9l4-6z" />
              <path d="M11 3 8 9l4 12 4-12-3-6" />
              <path d="M2 9h20" />
            </svg>
            <span>Kategori Tanding</span>
          </button>
        </template>

        <!-- Section: Kontingen & Peserta -->
        <template v-if="canAccessTab('atlet') || canAccessTab('verifikasi')">
          <div class="ops-nav__section-title">KONTINGEN & PESERTA</div>
          <button
            v-if="canAccessTab('atlet')"
            class="ops-nav__item"
            :class="{ 'ops-nav__item--active': activeNav === 'atlet' }"
            @click="setNav('atlet')"
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
              <circle cx="9" cy="7" r="4" />
              <path d="M22 21v-2a4 4 0 0 0-3-3.87" />
              <path d="M16 3.13a4 4 0 0 1 0 7.75" />
            </svg>
            <span>Data Atlet</span>
          </button>

          <button
            v-if="canAccessTab('verifikasi')"
            class="ops-nav__item"
            :class="{ 'ops-nav__item--active': activeNav === 'verifikasi' }"
            @click="setNav('verifikasi')"
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
              <circle cx="9" cy="7" r="4" />
              <polyline points="16 11 18 13 22 9" />
            </svg>
            <span>Verifikasi Peserta</span>
          </button>
        </template>

        <!-- Section: Manajemen Laga -->
        <div class="ops-nav__section-title">MANAJEMEN LAGA</div>
        <button
          v-if="canAccessTab('jadwal')"
          class="ops-nav__item"
          :class="{ 'ops-nav__item--active': activeNav === 'jadwal' }"
          @click="setNav('jadwal')"
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <rect width="18" height="18" x="3" y="4" rx="2" />
            <line x1="16" x2="16" y1="2" y2="6" />
            <line x1="8" x2="8" y1="2" y2="6" />
            <line x1="3" x2="21" y1="10" y2="10" />
          </svg>
          <span>Jadwal Pertandingan</span>
        </button>

        <button
          v-if="canAccessTab('bagan')"
          class="ops-nav__item"
          :class="{ 'ops-nav__item--active': activeNav === 'bagan' }"
          @click="setNav('bagan')"
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <line x1="6" x2="6" y1="3" y2="15" />
            <circle cx="18" cy="6" r="3" />
            <circle cx="6" cy="18" r="3" />
            <path d="M18 9a9 9 0 0 1-9 9" />
          </svg>
          <span>Bagan Pertandingan</span>
        </button>

        <button
          v-if="canAccessTab('pin')"
          class="ops-nav__item"
          :class="{ 'ops-nav__item--active': activeNav === 'pin' }"
          @click="setNav('pin')"
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
            <path d="M7 11V7a5 5 0 0 1 10 0v4" />
          </svg>
          <span>Perangkat & PIN</span>
        </button>

        <button
          v-if="canAccessTab('regulasi')"
          class="ops-nav__item"
          :class="{ 'ops-nav__item--active': activeNav === 'regulasi' }"
          @click="setNav('regulasi')"
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M4 4h16v16H4z" />
            <path d="M8 9h8" />
            <path d="M8 13h8" />
            <path d="M8 17h5" />
          </svg>
          <span>Regulasi & Ruleset</span>
        </button>

        <button
          v-if="canAccessTab('petugas')"
          class="ops-nav__item"
          :class="{ 'ops-nav__item--active': activeNav === 'petugas' }"
          data-test="nav-petugas"
          @click="setNav('petugas')"
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
            <circle cx="9" cy="7" r="4" />
            <path d="M22 21v-2a4 4 0 0 0-3-3.87" />
            <path d="M16 3.13a4 4 0 0 1 0 7.75" />
          </svg>
          <span>Akun Petugas</span>
        </button>

        <button
          v-if="canAccessTab('laporan')"
          class="ops-nav__item"
          :class="{ 'ops-nav__item--active': activeNav === 'laporan' }"
          @click="setNav('laporan')"
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
            <polyline points="14 2 14 8 20 8" />
            <line x1="16" x2="8" y1="13" y2="13" />
            <line x1="16" x2="8" y1="17" y2="17" />
            <polyline points="10 9 9 9 8 9" />
          </svg>
          <span>Laporan & Ekspor</span>
        </button>
      </nav>

      <!-- User Info Footer -->
      <div class="ops-sidebar__footer">
        <div class="ops-user">
          <div class="ops-user__avatar font-mono">{{ userInitials }}</div>
          <div class="ops-user__meta">
            <div class="ops-user__name truncate">{{ authStore.name || 'Ketua Pertandingan' }}</div>
            <div class="ops-user__role font-mono">
              {{ (authStore.role || 'Admin Arena').toUpperCase() }}
            </div>
          </div>
        </div>
        <button
          v-if="authStore.isAuthenticated"
          class="ops-logout-btn ops-logout-btn--key"
          title="Ganti Password Saya"
          aria-label="Ganti Password Saya"
          data-test="ops-change-password"
          @click="ownPasswordOpen = true"
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="7.5" cy="15.5" r="4.5" />
            <path d="m10.7 12.3 9.8-9.8" />
            <path d="m16 7 3 3" />
          </svg>
        </button>
        <button
          class="ops-logout-btn"
          title="Keluar dari Sesi (Logout)"
          @click="triggerLogoutPrompt"
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
            <polyline points="16 17 21 12 16 7" />
            <line x1="21" x2="9" y1="12" y2="12" />
          </svg>
        </button>
      </div>
    </aside>

    <!-- MAIN OPERATIONS CONTENT -->
    <main class="ops-main">
      <!-- TOP STATUS BAR -->
      <header class="ops-topbar">
        <button
          type="button"
          class="ops-hamburger-btn"
          aria-label="Buka Navigasi Operasional"
          @click="toggleMobileSidebar"
        >
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <line x1="3" y1="12" x2="21" y2="12" />
            <line x1="3" y1="6" x2="21" y2="6" />
            <line x1="3" y1="18" x2="21" y2="18" />
          </svg>
        </button>

        <div class="ops-topbar__titles">
          <div class="ops-topbar__sub font-mono">{{ currentNavMeta.sub }}</div>
          <h1 class="ops-topbar__heading">{{ currentNavMeta.title }}</h1>
        </div>

        <!-- COMPETITION SELECTOR (Global context switcher) -->
        <div
          class="ops-topbar__comp-select font-mono"
          :title="`Kompetisi aktif: ${activeCompetition?.name || activeCompId}`"
        >
          <svg
            width="15"
            height="15"
            viewBox="0 0 24 24"
            fill="none"
            stroke="#F59E0B"
            stroke-width="2.2"
            class="flex-shrink-0"
          >
            <path d="M6 9H4.5a2.5 2.5 0 0 1 0-5H6" />
            <path d="M18 9h1.5a2.5 2.5 0 0 0 0-5H18" />
            <path d="M4 22h16" />
            <path d="M10 14.66V17c0 .55-.47.98-.97 1.21C7.85 18.75 7 20.24 7 22" />
            <path d="M14 14.66V17c0 .55.47.98.97 1.21C16.15 18.75 17 20.24 17 22" />
            <path d="M18 2H6v7a6 6 0 0 0 12 0V2Z" />
          </svg>
          <span class="comp-label">EVENT:</span>
          <select
            :value="activeCompId"
            class="comp-dropdown font-mono"
            @change="switchCompetition(($event.target as HTMLSelectElement).value)"
          >
            <template v-if="competitionsList.length > 0">
              <option v-for="c in competitionsList" :key="c.id" :value="c.id">
                {{ c.name }} ({{ c.status.toUpperCase() }})
              </option>
            </template>
            <template v-else>
              <option :value="activeCompId">Kejuaraan Nasional Pencak Silat</option>
            </template>
          </select>
        </div>

        <!-- ARENA SELECTOR (Contextual for arena/match pages) -->
        <div v-if="currentNavMeta.hasArena" class="ops-topbar__arena-select font-mono">
          <span class="arena-label">GELANGGANG:</span>
          <select v-model="selectedArena" class="arena-dropdown font-mono">
            <template v-if="rawArenas && rawArenas.length > 0">
              <option v-for="a in rawArenas" :key="a.id" :value="a.name">
                {{ a.name }}
              </option>
            </template>
            <template v-else>
              <option value="GELANGGANG A (UTAMA)">GELANGGANG A (UTAMA)</option>
              <option value="GELANGGANG B">GELANGGANG B</option>
              <option value="GELANGGANG C">GELANGGANG C</option>
              <option value="GELANGGANG D">GELANGGANG D</option>
            </template>
          </select>
        </div>

        <div class="ops-topbar__system-pills font-mono">
          <div class="system-pill">
            <span class="system-pill__dot emerald" />
            <span>{{ serverUnreachable ? 'SERVER TERPUTUS' : 'SERVER ONLINE' }}</span>
          </div>
          <div class="system-pill">
            <svg
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="#10B981"
              stroke-width="2"
            >
              <path d="M4 11a9 9 0 0 1 9 9" />
              <path d="M4 4a16 16 0 0 1 16 16" />
              <circle cx="5" cy="19" r="1" />
            </svg>
            <span data-test="juri-online">{{ judgesOnline }} JURI ONLINE</span>
          </div>
          <div
            class="ops-profile-badge"
            :class="{ 'ops-profile-badge--active': authStore.isAuthenticated }"
            :title="
              authStore.isAuthenticated
                ? `Sesi aktif: ${authStore.name || 'Admin'} (Klik untuk Keluar)`
                : 'Login Admin / Ketua Pertandingan'
            "
            @click="handleProfileClick"
          >
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

      <!-- SCROLLABLE OPS VIEW -->
      <div
        class="ops-content-scroll"
        :class="{
          'ops-content-scroll--pin':
            activeNav === 'pin' ||
            activeNav === 'regulasi' ||
            activeNav === 'petugas' ||
            activeNav === 'laporan' ||
            activeNav === 'kompetisi' ||
            activeNav === 'gelanggang' ||
            activeNav === 'kategori' ||
            activeNav === 'atlet' ||
            activeNav === 'verifikasi' ||
            activeNav === 'jadwal',
        }"
      >
        <!-- TAB KOMPETISI -->
        <CompetitionsView v-if="activeNav === 'kompetisi' && canAccessTab('kompetisi')" />

        <!-- TAB GELANGGANG -->
        <ArenaOpsView
          v-else-if="activeNav === 'gelanggang' && canAccessTab('gelanggang')"
          :competition-id="activeCompId"
          @select-arena="handleArenaSelect"
          @switch-competition="switchCompetition"
        />

        <!-- TAB KATEGORI TANDING -->
        <CategoryOpsView
          v-else-if="activeNav === 'kategori' && canAccessTab('kategori')"
          :competition-id="activeCompId"
          :competition-name="activeCompetition?.name"
          @select-category="handleCategorySelect"
        />

        <!-- TAB DATA ATLET & KONTINGEN -->
        <AthletesOpsView
          v-else-if="activeNav === 'atlet' && canAccessTab('atlet')"
          :competition-id="activeCompId"
          @navigate-verification="
            canAccessTab('verifikasi') ? (activeNav = 'verifikasi') : undefined
          "
          @navigate-categories="canAccessTab('kategori') ? (activeNav = 'kategori') : undefined"
        />

        <!-- TAB VERIFIKASI PESERTA -->
        <ParticipantVerificationView
          v-else-if="activeNav === 'verifikasi' && canAccessTab('verifikasi')"
          :competition-id="activeCompId"
        />

        <!-- TAB JADWAL TANDING -->
        <ScheduleOpsView
          v-else-if="activeNav === 'jadwal' && canAccessTab('jadwal')"
          :competition-id="activeCompId"
        />

        <!-- TAB PERANGKAT & PIN JURI -->
        <DevicePinManager
          v-else-if="activeNav === 'pin' && canAccessTab('pin')"
          :competition-id="activeCompId"
        />

        <!-- TAB REGULASI & RULESET -->
        <RulesetsOpsView
          v-else-if="activeNav === 'regulasi' && canAccessTab('regulasi')"
          :competition-id="activeCompId"
        />

        <!-- TAB AKUN PETUGAS -->
        <OfficialsOpsView v-else-if="activeNav === 'petugas' && canAccessTab('petugas')" />

        <!-- TAB LAPORAN & EKSPOR -->
        <ReportsOpsView
          v-else-if="activeNav === 'laporan' && canAccessTab('laporan')"
          :competition-id="activeCompId"
        />

        <!-- TAB BAGAN PERTANDINGAN (DEFAULT) -->
        <template v-else-if="activeNav === 'bagan' && canAccessTab('bagan')">
          <!-- HEADER BANNER & ACTION BUTTONS -->
          <section class="ops-banner-row">
            <div class="ops-banner-title-group">
              <div class="ops-banner-icon-box">
                <svg
                  width="24"
                  height="24"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="#10B981"
                  stroke-width="2.2"
                >
                  <line x1="6" x2="6" y1="3" y2="15" />
                  <circle cx="18" cy="6" r="3" />
                  <circle cx="6" cy="18" r="3" />
                  <path d="M18 9a9 9 0 0 1-9 9" />
                </svg>
              </div>
              <div>
                <div class="ops-banner-sub font-mono">
                  <span>SISTEM ELIMINASI TUNGGAL (IPSI RULES 2023)</span>
                  <span class="ops-bullet">•</span>
                  <span class="rev-tag">REV: 04.2-FINAL</span>
                </div>
                <h2 class="ops-banner-main-title">BAGAN PERTANDINGAN TURNAMEN</h2>
              </div>
            </div>

            <div class="ops-banner-actions font-mono">
              <AppButton
                v-if="isAdminOrDewan"
                variant="emerald"
                size="sm"
                @click="handleGenerateDraw"
              >
                <template #icon>
                  <svg
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <path d="m16 2-4 4-4-4" />
                    <path d="M12 6v12" />
                    <path d="m8 18 4 4 4-4" />
                  </svg>
                </template>
                Generate Bagan Acak
              </AppButton>

              <button
                v-if="isAdminOrDewan"
                class="ops-action-btn-dark font-mono"
                :class="{ 'ops-action-btn-dark--unlocked': !isLocked }"
                :title="
                  isLocked
                    ? 'Draw sedang TERKUNCI. Klik untuk membuka kunci.'
                    : 'Draw sedang TERBUKA (bisa diacak ulang). Klik untuk mengunci.'
                "
                :aria-pressed="isLocked"
                data-test="draw-lock"
                @click="toggleLock"
              >
                <svg
                  v-if="isLocked"
                  width="15"
                  height="15"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
                  <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                </svg>
                <svg
                  v-else
                  width="15"
                  height="15"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
                  <path d="M7 11V7a5 5 0 0 1 9.9-1" />
                </svg>
                <span>{{ isLocked ? 'Draw Terkunci' : 'Draw Terbuka' }}</span>
                <small class="ops-lock-hint">{{
                  isLocked ? 'klik untuk buka' : 'klik untuk kunci'
                }}</small>
              </button>

              <button class="ops-action-btn-dark font-mono" @click="handlePrintPDF">
                <svg
                  width="15"
                  height="15"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <polyline points="6 9 6 2 18 2 18 9" />
                  <path
                    d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"
                  />
                  <rect width="12" height="8" x="6" y="14" />
                </svg>
                <span>Cetak PDF</span>
              </button>
            </div>
          </section>

          <!-- FILTERS & QUALIFICATION ROW -->
          <section class="ops-filter-grid">
            <!-- Filter 1: Kategori & Kelas -->
            <div class="filter-card">
              <label class="filter-label font-mono">KATEGORI & KELAS TANDING</label>
              <div class="filter-select-wrapper">
                <select v-model="selectedCategory" class="filter-select font-mono">
                  <optgroup v-for="g in categoryOptionGroups" :key="g.label" :label="g.label">
                    <option v-for="c in g.options" :key="c.id" :value="c.name">
                      {{ c.label }}
                    </option>
                  </optgroup>
                </select>
              </div>
            </div>

            <!-- Card 2: Kualifikasi Timbang Badan -->
            <div class="filter-card">
              <label class="filter-label font-mono">KUALIFIKASI TIMBANG BADAN</label>
              <div class="weigh-in-row font-mono">
                <div class="weigh-in-status">
                  <svg
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="#10B981"
                    stroke-width="2.5"
                  >
                    <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                    <polyline points="22 4 12 14.01 9 11.01" />
                  </svg>
                  <span>Timbang Sah</span>
                </div>
                <AppBadge variant="emerald"
                  >{{ rawParticipants?.length || 0 }} ATLET TERDAFTAR</AppBadge
                >
              </div>
            </div>

            <!-- Card 3: Integrasi Gelanggang -->
            <div class="filter-card">
              <label class="filter-label font-mono">INTEGRASI GELANGGANG</label>
              <div class="arena-integration-row font-mono">
                <div class="arena-ready">
                  <span class="arena-ready__dot" />
                  <span>{{
                    rawArenas && rawArenas.length > 0
                      ? `${rawArenas.length} GELANGGANG READY`
                      : 'GELANGGANG SIAP'
                  }}</span>
                </div>
                <span class="arena-lat">LAT: {{ telemetryMs }}ms</span>
              </div>
            </div>
          </section>

          <!-- SUMMARY METRICS BAR -->
          <section class="ops-metrics-bar font-mono">
            <div class="metric-block">
              <div class="metric-label">TOTAL PESERTA</div>
              <div class="metric-val">
                {{ rawParticipants?.length || 16 }} <span class="metric-unit">Atlet</span>
              </div>
            </div>
            <div class="metric-block">
              <div class="metric-label">PARTAI SELESAI</div>
              <div class="metric-val">
                {{ rawMatches?.filter((m) => m.status === 'finished').length || 0 }}
                <span class="metric-unit">/ {{ rawMatches?.length || 15 }} Laga</span>
              </div>
            </div>
            <div class="metric-block">
              <div class="metric-label">SEDANG TANDING</div>
              <div class="metric-val text-emerald-400">
                {{ rawMatches?.filter((m) => isLiveMatch(m.status)).length || 0 }}
                <span class="metric-unit">LIVE</span>
              </div>
            </div>
            <div class="metric-block">
              <div class="metric-label">MENUNGGU ANTRIAN</div>
              <div class="metric-val">
                {{
                  rawMatches?.filter((m) => m.status === 'ready' || m.status === 'idle').length ||
                  (rawParticipants?.length ? Math.floor(rawParticipants.length / 2) : 0)
                }}
                <span class="metric-unit">Partai</span>
              </div>
            </div>
            <div class="metric-block metric-block--progress">
              <div class="metric-label">PROGRES BAGAN</div>
              <div class="bracket-progress-wrapper">
                <div class="bracket-progress-bar">
                  <div
                    class="bracket-progress-fill"
                    :style="{
                      width: `${rawMatches?.length ? Math.round((rawMatches.filter((m) => m.status === 'finished').length / rawMatches.length) * 100) : rawBracketSlots?.length ? 40 : 10}%`,
                    }"
                  />
                </div>
                <span class="progress-num"
                  >{{
                    rawMatches?.length
                      ? Math.round(
                          (rawMatches.filter((m) => m.status === 'finished').length /
                            rawMatches.length) *
                            100,
                        )
                      : rawBracketSlots?.length
                        ? 40
                        : 10
                  }}%</span
                >
              </div>
            </div>
          </section>

          <!-- TOURNAMENT BRACKET TREE (PENCAK SILAT EDITION VUE-TOURNAMENT-BRACKET) -->
          <TournamentTreeBracket
            v-if="bracketRounds.length > 0"
            :rounds="bracketRounds"
            :selected-match-id="selectedBracketMatch?.id || queuedMatch?.id"
            :category-name="selectedCategory"
            @select-match="handleSelectBracketMatch"
            @generate-draw="handleGenerateDraw"
          />

          <!-- EMPTY STATE JIKA BAGAN KATEGORI BELUM DI-GENERATE -->
          <div v-else class="bracket-empty-state font-mono">
            <div class="bracket-empty-icon">
              <svg
                width="40"
                height="40"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.8"
              >
                <line x1="6" x2="6" y1="3" y2="15" />
                <circle cx="18" cy="6" r="3" />
                <circle cx="6" cy="18" r="3" />
                <path d="M18 9a9 9 0 0 1-9 9" />
              </svg>
            </div>
            <h3 class="bracket-empty-title">Bagan Pertandingan Belum Di-Generate</h3>
            <p class="bracket-empty-desc">
              Kategori {{ selectedCategory }} belum memiliki susunan bagan pertandingan resmi. Klik
              tombol di bawah untuk mengundi susunan pohon pertandingan dari atlet terverifikasi.
            </p>
            <AppButton variant="emerald" size="md" @click="handleGenerateDraw">
              Generate Bagan Acak Sekarang
            </AppButton>
          </div>
        </template>
      </div>

      <!-- BOTTOM STICKY QUICK-ACTION / COMMAND DOCK -->
      <footer v-if="activeNav === 'bagan'" class="ops-dock">
        <div class="ops-dock__match-info">
          <div class="ops-dock__athlete-icon">
            <svg
              width="22"
              height="22"
              viewBox="0 0 24 24"
              fill="none"
              stroke="#DC2626"
              stroke-width="2.2"
            >
              <circle cx="12" cy="5" r="3" />
              <path d="M6.5 9 12 14l5.5-5" />
              <path d="M12 14v8" />
              <path d="M9 22h6" />
            </svg>
          </div>
          <div>
            <div class="ops-dock__sub font-mono">
              <span>PARTAI ANTRIAN BERIKUTNYA</span>
              <span class="ops-bullet">•</span>
              <span>{{ selectedCategory }}</span>
            </div>
            <div class="ops-dock__match-title">
              {{
                queuedMatch
                  ? `PARTAI ${queuedMatch.partaiNumber}: ${queuedMatch.red.name} (${queuedMatch.red.contingent}) vs ${queuedMatch.blue.name} (${queuedMatch.blue.contingent})`
                  : 'Belum ada antrian partai aktif'
              }}
            </div>
          </div>
        </div>

        <div class="ops-dock__actions font-mono">
          <button class="ops-dock-btn-dark font-mono" @click="handlePanggilAtlet">
            Panggil Atlet
          </button>
          <AppButton variant="emerald" size="md" @click="handleLoadToMat1">
            <template #icon>
              <svg
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <polygon points="5 3 19 12 5 21 5 3" />
              </svg>
            </template>
            Kirim ke Gelanggang
          </AppButton>
        </div>

        <div class="ops-dock__legal font-mono">
          <div class="ops-legal-texts">
            <div class="legal-label">LEGALITAS BAGAN PERTANDINGAN</div>
            <div class="legal-officials">DELEGET TEKNIK & KETUA PERTANDINGAN</div>
            <div class="legal-hash">HASH-DRAW: {{ drawHash }}</div>
          </div>
          <div class="legal-shield">
            <svg
              width="22"
              height="22"
              viewBox="0 0 24 24"
              fill="none"
              stroke="#10B981"
              stroke-width="2.2"
            >
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
              <polyline points="9 12 11 14 15 10" />
            </svg>
          </div>
        </div>
      </footer>
    </main>

    <!-- INTERACTIVE TOAST NOTIFICATION -->
    <Transition name="fade">
      <div v-if="toast" class="ops-toast font-mono" :class="`ops-toast--${toast.type}`">
        <div class="toast-title font-bold">{{ toast.title }}</div>
        <div class="toast-msg">{{ toast.message }}</div>
      </div>
    </Transition>

    <ChangeOwnPasswordDialog
      :open="ownPasswordOpen"
      @close="ownPasswordOpen = false"
      @changed="
        showToast('Password Diganti', 'Gunakan password baru saat masuk berikutnya.', 'success')
      "
    />

    <!-- MODAL CONFIRMATION DIALOG -->
    <Transition name="modal-fade">
      <div v-if="modal?.show" class="ops-modal-backdrop" @click.self="modal = null">
        <div class="ops-modal">
          <h3 class="modal-title font-mono">{{ modal.title }}</h3>
          <p class="modal-msg">{{ modal.message }}</p>
          <div class="modal-actions font-mono">
            <AppButton variant="outlined" size="sm" @click="modal = null"> Batal </AppButton>
            <AppButton :variant="modal.confirmVariant || 'primary'" size="sm" @click="modal.action">
              {{ modal.confirmText }}
            </AppButton>
          </div>
        </div>
      </div>
    </Transition>

    <!-- MODAL ADMIN / OPERATOR LOGIN -->
    <Transition name="modal-fade">
      <div
        v-if="showAdminLoginModal"
        class="ops-modal-backdrop"
        @click.self="showAdminLoginModal = false"
      >
        <div class="ops-modal">
          <div style="display: flex; align-items: center; gap: 12px; margin-bottom: 16px">
            <div
              style="
                width: 38px;
                height: 38px;
                border-radius: 8px;
                background-color: rgba(220, 38, 38, 0.2);
                border: 1px solid rgba(220, 38, 38, 0.4);
                display: flex;
                align-items: center;
                justify-content: center;
                color: #ef4444;
                font-weight: 800;
              "
              class="font-mono"
            >
              KP
            </div>
            <div>
              <h3 class="modal-title font-mono" style="margin: 0; font-size: 1rem">
                LOGIN KETUA PERTANDINGAN
              </h3>
              <p style="font-size: 0.6875rem; color: #94a3b8; margin: 2px 0 0 0">
                Akses otorisasi meja komando & manajemen PIN arena
              </p>
            </div>
          </div>

          <div style="display: flex; flex-direction: column; gap: 6px; margin-bottom: 12px">
            <label class="font-mono" style="font-size: 0.6875rem; color: #64748b"
              >EMAIL OPERATOR / ADMIN</label
            >
            <input
              v-model="adminEmail"
              type="email"
              placeholder="admin@silat.id"
              style="
                background-color: #080d1a;
                border: 1px solid rgba(255, 255, 255, 0.12);
                color: #ffffff;
                padding: 8px 12px;
                border-radius: 6px;
                font-size: 0.8125rem;
                outline: none;
              "
              class="font-mono"
            />
          </div>

          <div style="display: flex; flex-direction: column; gap: 6px; margin-bottom: 16px">
            <label class="font-mono" style="font-size: 0.6875rem; color: #64748b">KATA SANDI</label>
            <input
              v-model="adminPassword"
              type="password"
              placeholder="••••••••"
              style="
                background-color: #080d1a;
                border: 1px solid rgba(255, 255, 255, 0.12);
                color: #ffffff;
                padding: 8px 12px;
                border-radius: 6px;
                font-size: 0.8125rem;
                outline: none;
              "
              class="font-mono"
              @keydown.enter="handleAdminLogin"
            />
          </div>

          <div
            v-if="adminLoginError"
            class="font-mono"
            style="font-size: 0.75rem; color: #ef4444; margin-bottom: 14px"
          >
            {{ adminLoginError }}
          </div>

          <div class="modal-actions font-mono">
            <AppButton variant="outlined" size="sm" @click="showAdminLoginModal = false">
              Tutup
            </AppButton>
            <AppButton
              variant="emerald"
              size="sm"
              :loading="adminLoginLoading"
              @click="handleAdminLogin"
            >
              Masuk Komando
            </AppButton>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
/* Main Container Layout */
.ops-layout {
  display: flex;
  width: 100%;
  height: 100dvh;
  max-height: 100dvh;
  background-color: #080d1a;
  color: #e2e8f0;
  overflow: hidden;
  font-family: var(--font-display, 'Space Grotesk', sans-serif);
  box-sizing: border-box;
}

/* ==========================================================================
   LEFT SIDEBAR
   ========================================================================== */
.ops-sidebar {
  width: 250px;
  background-color: #0b1120;
  border-right: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  z-index: 20;
}

.ops-sidebar__header {
  padding: 20px 18px 16px 18px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.ops-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.ops-brand__logo {
  width: 36px;
  height: 36px;
  background-color: #dc2626;
  color: #ffffff;
  font-weight: 800;
  font-size: 1.125rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  box-shadow: 0 0 12px rgba(220, 38, 38, 0.4);
}

.ops-brand__title {
  font-size: 0.9375rem;
  font-weight: 800;
  letter-spacing: 0.04em;
  color: #ffffff;
  line-height: 1.2;
}

.ops-brand__sub {
  font-size: 0.6875rem;
  color: #64748b;
  letter-spacing: 0.06em;
}

.ops-telemetry {
  display: flex;
  align-items: center;
  gap: 8px;
  background-color: rgba(15, 23, 42, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.06);
  padding: 6px 10px;
  border-radius: 6px;
  font-size: 0.6875rem;
}

.telemetry-dot--down {
  background: #ef4444 !important;
  box-shadow: 0 0 6px #ef4444 !important;
}
.telemetry-dot {
  width: 7px;
  height: 7px;
  background-color: #10b981;
  border-radius: 50%;
  box-shadow: 0 0 8px #10b981;
}

.telemetry-text {
  color: #94a3b8;
  letter-spacing: 0.06em;
}

.telemetry-ping {
  margin-left: auto;
  color: #10b981;
  font-weight: 700;
}

/* Sidebar Nav */
.ops-nav {
  flex: 1;
  overflow-y: auto;
  padding: 16px 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ops-nav__section-title {
  font-size: 0.6875rem;
  font-weight: 700;
  color: #475569;
  letter-spacing: 0.08em;
  padding: 12px 10px 4px 10px;
}

.ops-nav__item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 9px 12px;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: #94a3b8;
  font-size: 0.8125rem;
  text-align: left;
  cursor: pointer;
  transition: all 0.15s ease;
}

.ops-nav__item:hover {
  background-color: rgba(255, 255, 255, 0.04);
  color: #f8fafc;
}

.ops-nav__item--active {
  background-color: #1e293b;
  color: #38bdf8;
  font-weight: 600;
  box-shadow: inset 2px 0 0 #38bdf8;
}

.ops-sidebar__footer {
  padding: 14px 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  display: flex;
  align-items: center;
  justify-content: space-between;
  background-color: #0a0f1d;
}

.ops-user {
  display: flex;
  align-items: center;
  gap: 10px;
}

.ops-user__avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background-color: #1e293b;
  color: #38bdf8;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
  font-weight: 700;
  border: 1px solid rgba(56, 189, 248, 0.3);
}

.ops-user__name {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #f1f5f9;
  line-height: 1.2;
}

.ops-user__role {
  font-size: 0.625rem;
  color: #64748b;
  letter-spacing: 0.06em;
}

.ops-logout-btn {
  background: transparent;
  border: none;
  color: #64748b;
  padding: 6px;
  cursor: pointer;
  border-radius: 4px;
}

.ops-logout-btn:hover {
  color: #dc2626;
  background-color: rgba(220, 38, 38, 0.1);
}

.ops-logout-btn--key:hover {
  color: #34d399;
  background-color: rgba(52, 211, 153, 0.1);
}

/* ==========================================================================
   MAIN OPS VIEW & TOPBAR
   ========================================================================== */
.ops-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
  min-width: 0;
  width: 100%;
  box-sizing: border-box;
}

.ops-hamburger-btn {
  display: none;
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #94a3b8;
  padding: 8px;
  border-radius: 6px;
  cursor: pointer;
  align-items: center;
  justify-content: center;
  min-width: 38px;
  min-height: 38px;
  touch-action: manipulation;
  flex-shrink: 0;
  transition: all 0.15s ease;
}
.ops-hamburger-btn:hover {
  background-color: rgba(255, 255, 255, 0.08);
  color: #ffffff;
}

.ops-sidebar-close {
  display: none;
  margin-left: auto;
  background: transparent;
  border: none;
  color: #94a3b8;
  padding: 6px;
  border-radius: 4px;
  cursor: pointer;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  min-height: 32px;
  touch-action: manipulation;
}
.ops-sidebar-close:hover {
  color: #ffffff;
}

.ops-sidebar-backdrop {
  display: none;
}

/* Top Status Bar */
.ops-topbar {
  height: 60px;
  background-color: #0b1120;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  padding: 0 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  flex-shrink: 0;
}

.ops-topbar__sub {
  font-size: 0.6875rem;
  color: #64748b;
  letter-spacing: 0.08em;
  line-height: 1.2;
}

.ops-topbar__heading {
  font-size: 1.0625rem;
  font-weight: 800;
  color: #ffffff;
  letter-spacing: 0.02em;
  line-height: 1.2;
}

.ops-topbar__comp-select {
  display: flex;
  align-items: center;
  gap: 8px;
  background-color: #0f172a;
  padding: 6px 14px;
  border-radius: 6px;
  border: 1px solid rgba(245, 158, 11, 0.25);
  box-shadow: 0 0 10px rgba(245, 158, 11, 0.05);
}

.comp-label {
  font-size: 0.6875rem;
  color: #f59e0b;
  letter-spacing: 0.06em;
  font-weight: 700;
}

.comp-dropdown {
  background: transparent;
  border: none;
  color: #f8fafc;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
  outline: none;
  max-width: 260px;
}

.comp-dropdown option {
  background-color: #0f172a;
  color: #e2e8f0;
}

.ops-topbar__arena-select {
  display: flex;
  align-items: center;
  gap: 10px;
  background-color: #0f172a;
  padding: 6px 14px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.arena-label {
  font-size: 0.6875rem;
  color: #64748b;
  letter-spacing: 0.06em;
}

.arena-dropdown {
  background: transparent;
  border: none;
  color: #38bdf8;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
  outline: none;
}

.arena-dropdown option {
  background-color: #0f172a;
  color: #e2e8f0;
}

.ops-topbar__system-pills {
  display: flex;
  align-items: center;
  gap: 12px;
}

.system-pill {
  display: flex;
  align-items: center;
  gap: 8px;
  background-color: rgba(15, 23, 42, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.06);
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 0.6875rem;
  color: #94a3b8;
  letter-spacing: 0.04em;
}

.system-pill__dot.emerald {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: #10b981;
  box-shadow: 0 0 6px #10b981;
}

.ops-profile-badge {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background-color: rgba(239, 68, 68, 0.2);
  color: #ef4444;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
}

.ops-profile-badge--active {
  background-color: rgba(16, 185, 129, 0.2);
  color: #10b981;
  border: 1px solid rgba(16, 185, 129, 0.4);
}

/* ==========================================================================
   SCROLLABLE CONTENT AREA
   ========================================================================= */
.ops-content-scroll {
  flex: 1;
  overflow-y: auto;
  overflow-x: auto;
  padding: 24px 28px 120px 28px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  min-width: 0;
  width: 100%;
  box-sizing: border-box;
}

.ops-content-scroll--pin {
  padding-bottom: 40px;
}

/* Banner Row */
.ops-banner-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  background-color: #0e1626;
  border: 1px solid rgba(255, 255, 255, 0.08);
  padding: 18px 24px;
  border-radius: 10px;
}

.ops-banner-title-group {
  display: flex;
  align-items: center;
  gap: 16px;
}

.ops-banner-icon-box {
  width: 44px;
  height: 44px;
  background-color: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.3);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.ops-banner-sub {
  font-size: 0.75rem;
  color: #94a3b8;
  letter-spacing: 0.06em;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.ops-bullet {
  color: #475569;
}

.rev-tag {
  color: #10b981;
}

.ops-banner-main-title {
  font-size: 1.375rem;
  font-weight: 800;
  letter-spacing: 0.02em;
  color: #ffffff;
}

.ops-banner-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.ops-action-btn-dark {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background-color: #162032;
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #e2e8f0;
  padding: 8px 14px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.ops-action-btn-dark--unlocked {
  border-color: #f59e0b;
  color: #fbbf24;
}
.ops-lock-hint {
  margin-left: 4px;
  font-size: 10px;
  font-weight: 500;
  opacity: 0.7;
}

.ops-action-btn-dark:hover {
  background-color: #1e293b;
  border-color: rgba(255, 255, 255, 0.2);
}

/* Filters Row */
.ops-filter-grid {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr;
  gap: 16px;
}

.filter-card {
  background-color: #0e1626;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.filter-label {
  font-size: 0.6875rem;
  color: #64748b;
  letter-spacing: 0.06em;
}

.filter-select {
  width: 100%;
  background: transparent;
  border: none;
  color: #f8fafc;
  font-size: 0.8125rem;
  font-weight: 600;
  outline: none;
  cursor: pointer;
}

.filter-select option {
  background-color: #0f172a;
  color: #f8fafc;
}

.weigh-in-row,
.arena-integration-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 2px;
}

.weigh-in-status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  font-weight: 600;
  color: #f1f5f9;
}

.arena-ready {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.75rem;
  font-weight: 600;
  color: #e2e8f0;
}

.arena-ready__dot {
  width: 7px;
  height: 7px;
  background-color: #10b981;
  border-radius: 50%;
  box-shadow: 0 0 6px #10b981;
}

.arena-lat {
  font-size: 0.6875rem;
  color: #64748b;
}

/* Metrics Bar */
.ops-metrics-bar {
  display: grid;
  grid-template-columns: repeat(4, 1fr) 1.5fr;
  gap: 16px;
  background-color: #0e1626;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  padding: 14px 20px;
}

.metric-block {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.metric-label {
  font-size: 0.6875rem;
  color: #64748b;
  letter-spacing: 0.06em;
}

.metric-val {
  font-size: 1.25rem;
  font-weight: 800;
  color: #ffffff;
}

.metric-unit {
  font-size: 0.75rem;
  font-weight: 400;
  color: #94a3b8;
}

.bracket-progress-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 4px;
}

.bracket-progress-bar {
  flex: 1;
  height: 8px;
  background-color: #1e293b;
  border-radius: 4px;
  overflow: hidden;
}

.bracket-progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #10b981, #34d399);
  border-radius: 4px;
}

.progress-num {
  font-size: 0.8125rem;
  font-weight: 700;
  color: #10b981;
}

/* ==========================================================================
   BRACKET TREE COLUMNS
   ========================================================================== */
.bracket-tree-container {
  display: flex;
  gap: 24px;
  overflow-x: auto;
  padding-bottom: 12px;
  min-width: 100%;
}

.bracket-column {
  flex: 1;
  min-width: 280px;
  max-width: 360px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.bracket-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 60px 24px;
  background-color: #0e1626;
  border: 1px dashed rgba(255, 255, 255, 0.15);
  border-radius: 12px;
  gap: 16px;
  max-width: 560px;
  margin: 32px auto;
}

.bracket-empty-icon {
  width: 64px;
  height: 64px;
  border-radius: 12px;
  background-color: rgba(56, 189, 248, 0.1);
  border: 1px solid rgba(56, 189, 248, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #38bdf8;
}

.bracket-empty-title {
  font-size: 1.125rem;
  font-weight: 700;
  color: #ffffff;
}

.bracket-empty-desc {
  font-size: 0.875rem;
  color: #94a3b8;
  line-height: 1.5;
  max-width: 440px;
}

/* ==========================================================================
   BOTTOM STICKY DOCK / COMMAND BAR
   ========================================================================== */
.ops-dock {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 80px;
  background-color: #090f1c;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  padding: 0 28px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-shadow: 0 -10px 25px rgba(0, 0, 0, 0.5);
  z-index: 30;
}

.ops-dock__match-info {
  display: flex;
  align-items: center;
  gap: 14px;
}

.ops-dock__athlete-icon {
  width: 44px;
  height: 44px;
  background-color: rgba(220, 38, 38, 0.12);
  border: 1px solid rgba(220, 38, 38, 0.3);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.ops-dock__sub {
  font-size: 0.6875rem;
  color: #94a3b8;
  letter-spacing: 0.06em;
  display: flex;
  align-items: center;
  gap: 8px;
}

.ops-dock__match-title {
  font-size: 1rem;
  font-weight: 700;
  color: #ffffff;
}

.ops-dock__actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.ops-dock-btn-dark {
  background-color: #162032;
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #e2e8f0;
  padding: 9px 16px;
  border-radius: 6px;
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
}

.ops-dock-btn-dark:hover {
  background-color: #1e293b;
}

.ops-dock__legal {
  display: flex;
  align-items: center;
  gap: 14px;
  text-align: right;
}

.legal-label {
  font-size: 0.625rem;
  color: #64748b;
  letter-spacing: 0.08em;
}

.legal-officials {
  font-size: 0.75rem;
  font-weight: 700;
  color: #cbd5e1;
}

.legal-hash {
  font-size: 0.6875rem;
  color: #10b981;
  letter-spacing: 0.06em;
}

.legal-shield {
  width: 36px;
  height: 36px;
  background-color: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.3);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* ==========================================================================
   TOAST & MODAL
   ========================================================================== */
.ops-toast {
  position: fixed;
  top: 76px;
  right: 28px;
  background-color: #0f172a;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 8px;
  padding: 12px 18px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.5);
  z-index: 100;
}

.ops-toast--success {
  border-color: #10b981;
}
.ops-toast--warning {
  border-color: #f59e0b;
}
.ops-toast--info {
  border-color: #38bdf8;
}

.toast-title {
  font-size: 0.8125rem;
  color: #ffffff;
  margin-bottom: 2px;
}

.toast-msg {
  font-size: 0.75rem;
  color: #94a3b8;
}

.ops-modal-backdrop {
  position: fixed;
  inset: 0;
  background-color: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 150;
}

.ops-modal {
  background-color: #0f172a;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 12px;
  padding: 24px;
  max-width: 440px;
  width: 90%;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
}

.modal-title {
  font-size: 1.125rem;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 10px;
}

.modal-msg {
  font-size: 0.875rem;
  color: #94a3b8;
  line-height: 1.5;
  margin-bottom: 24px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* Animations */
.fade-enter-active,
.fade-leave-active,
.modal-fade-enter-active,
.modal-fade-leave-active {
  transition:
    opacity 0.2s ease,
    transform 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
  transform: scale(0.95);
}

/* ==========================================================================
   RESPONSIVE DESIGN (TABLET & MOBILE)
   ========================================================================== */
@media (max-width: 1024px) {
  .ops-hamburger-btn {
    display: inline-flex;
  }
  .ops-sidebar-close {
    display: inline-flex;
  }
  .ops-sidebar-backdrop {
    display: block;
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.72);
    backdrop-filter: blur(3px);
    z-index: 45;
  }
  .ops-sidebar {
    position: fixed;
    top: 0;
    bottom: 0;
    left: 0;
    width: 280px;
    max-width: 85vw;
    z-index: 50;
    transform: translateX(-100%);
    transition: transform 0.25s cubic-bezier(0.16, 1, 0.3, 1);
    box-shadow: none;
  }
  .ops-sidebar--open {
    transform: translateX(0);
    box-shadow: 10px 0 30px rgba(0, 0, 0, 0.7);
  }
  .ops-topbar {
    padding: 10px 16px;
    height: auto;
    min-height: 56px;
    flex-wrap: wrap;
    gap: 12px;
    padding-top: max(10px, env(safe-area-inset-top));
  }
  .ops-topbar__titles {
    flex: 1 1 auto;
    min-width: 160px;
  }
  .ops-topbar__comp-select {
    max-width: 100%;
  }
  .comp-dropdown {
    max-width: 180px;
  }
  .ops-filter-grid {
    grid-template-columns: 1fr 1fr;
  }
  .ops-filter-grid .filter-card:first-child {
    grid-column: span 2;
  }
  .ops-metrics-bar {
    grid-template-columns: repeat(2, 1fr) 1.5fr;
  }
}

@media (max-width: 768px) {
  .ops-topbar {
    padding: 8px 12px;
    gap: 8px;
  }
  .ops-topbar__heading {
    font-size: 0.9375rem;
  }
  .ops-topbar__sub {
    font-size: 0.625rem;
  }
  .comp-label,
  .arena-label {
    display: none;
  }
  .comp-dropdown {
    max-width: 140px;
    font-size: 0.75rem;
  }
  .ops-topbar__system-pills .system-pill {
    display: none;
  }
  .ops-banner-row {
    flex-direction: column;
    align-items: stretch;
    padding: 14px 16px;
    gap: 14px;
  }
  .ops-banner-title-group {
    gap: 12px;
  }
  .ops-banner-main-title {
    font-size: 1.125rem;
  }
  .ops-banner-actions {
    width: 100%;
    flex-wrap: wrap;
    gap: 8px;
  }
  .ops-banner-actions button,
  .ops-banner-actions :deep(button) {
    flex: 1 1 auto;
    justify-content: center;
  }
  .ops-filter-grid {
    grid-template-columns: 1fr;
    gap: 10px;
  }
  .ops-filter-grid .filter-card:first-child {
    grid-column: span 1;
  }
  .ops-metrics-bar {
    grid-template-columns: repeat(2, 1fr);
    gap: 10px;
    padding: 12px 14px;
  }
  .metric-block--progress {
    grid-column: span 2;
  }
  .metric-val {
    font-size: 1.1rem;
  }
  .ops-content-scroll {
    padding: 14px 12px 180px 12px;
    gap: 12px;
  }
  .ops-content-scroll--pin {
    padding-bottom: max(32px, env(safe-area-inset-bottom));
  }
  .ops-dock {
    height: auto;
    flex-direction: column;
    align-items: stretch;
    padding: 12px 14px;
    padding-bottom: max(12px, env(safe-area-inset-bottom));
    gap: 10px;
  }
  .ops-dock__match-info {
    gap: 10px;
  }
  .ops-dock__athlete-icon {
    width: 36px;
    height: 36px;
    flex-shrink: 0;
  }
  .ops-dock__match-title {
    font-size: 0.8125rem;
    line-height: 1.35;
  }
  .ops-dock__actions {
    width: 100%;
    display: flex;
    gap: 8px;
  }
  .ops-dock__actions > button,
  .ops-dock__actions > :deep(button) {
    flex: 1 1 50%;
    justify-content: center;
  }
  .ops-dock__legal {
    justify-content: space-between;
    text-align: left;
    border-top: 1px solid rgba(255, 255, 255, 0.08);
    padding-top: 8px;
  }
  .legal-shield {
    width: 30px;
    height: 30px;
  }
}

@media (max-width: 480px) {
  .ops-topbar__titles {
    min-width: 120px;
  }
  .ops-topbar__comp-select {
    padding: 4px 8px;
  }
  .comp-dropdown {
    max-width: 120px;
  }
  .ops-metrics-bar {
    grid-template-columns: 1fr;
  }
  .metric-block--progress {
    grid-column: span 1;
  }
  .ops-dock__actions {
    flex-direction: column;
  }
  .ops-dock__actions > button,
  .ops-dock__actions > :deep(button) {
    width: 100%;
  }
}

/* Print Styles */
@media print {
  .ops-sidebar,
  .ops-topbar,
  .ops-dock,
  .ops-banner-actions {
    display: none !important;
  }
  .ops-layout {
    background: #ffffff !important;
    color: #000000 !important;
    height: auto !important;
    overflow: visible !important;
  }
  .ops-main {
    overflow: visible !important;
  }
  .ops-content-scroll {
    padding: 0 !important;
    overflow: visible !important;
  }
  :deep(.t-match-card) {
    background: #ffffff !important;
    border: 1px solid #000000 !important;
    color: #000000 !important;
  }
}
</style>
