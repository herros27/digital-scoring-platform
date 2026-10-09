<script setup lang="ts">
import { getErrorMessage } from '@/api/error'
import { ref, reactive, computed, watch } from 'vue'
import { AppButton } from '@/components/ui'
import {
  useParticipantsQuery,
  useUpdateAthleteProfileMutation,
  useVerifyParticipantMutation,
} from '../api/athleteQueries'
import type { Participant } from '../types/athlete.schema'
import { useCategoriesQuery } from '@/features/categories/api/categoryQueries'
import type { Category } from '@/features/categories/types/category.schema'
import {
  docChecklist,
  ageGroupLabel,
  checkWeight,
  missingDocs,
  type WeightVerdict,
} from '@/features/categories/lib/ageGroups'

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
const {
  data: rawParticipants,
  isLoading,
  isFetching,
  error,
  refetch,
} = useParticipantsQuery(activeCompId)
const verifyMutation = useVerifyParticipantMutation(activeCompId)
const profileMutation = useUpdateAthleteProfileMutation(activeCompId)

const participants = computed(() => rawParticipants.value || [])

const { data: rawCategories } = useCategoriesQuery(activeCompId)
const categoryById = computed(() => {
  const map = new Map<string, Category>()
  for (const c of rawCategories.value || []) map.set(c.id, c)
  return map
})
const sortedCategories = computed(() =>
  [...(rawCategories.value || [])].sort((a, b) =>
    a.name.localeCompare(b.name, undefined, { numeric: true }),
  ),
)

// ---- Meja timbang & checklist berkas (state lokal per peserta) ----
interface DeskState {
  weight: string
  docs: Record<string, boolean>
}
const desk = reactive<Record<string, DeskState>>({})

// Isi awal dari data server; tidak menimpa input yang sedang diketik.
watch(
  participants,
  (list) => {
    for (const p of list) {
      if (!desk[p.id]) {
        desk[p.id] = {
          weight: p.actual_weight != null ? String(p.actual_weight) : '',
          docs: { ...(p.docs_verified || {}) },
        }
      }
    }
  },
  { immediate: true },
)

function deskOf(p: Participant): DeskState {
  return desk[p.id] ?? (desk[p.id] = { weight: '', docs: {} })
}

function parsedWeight(p: Participant): number | null {
  const raw = deskOf(p).weight.trim().replace(',', '.')
  if (!raw) return null
  const n = Number(raw)
  return Number.isFinite(n) && n > 0 ? n : null
}

function weightVerdict(p: Participant): WeightVerdict | null {
  const w = parsedWeight(p)
  const cat = p.category_id ? categoryById.value.get(p.category_id) : undefined
  if (w === null || !cat) return null
  return checkWeight(w, cat.min_weight, cat.max_weight)
}

function categoryHasWeightBounds(p: Participant): boolean {
  const cat = p.category_id ? categoryById.value.get(p.category_id) : undefined
  return !!cat && (cat.min_weight != null || cat.max_weight != null)
}

function weightRangeLabel(p: Participant): string {
  const cat = p.category_id ? categoryById.value.get(p.category_id) : undefined
  if (!cat || !categoryHasWeightBounds(p)) return 'Tanpa batas bobot'
  const min = cat.min_weight ?? null
  const max = cat.max_weight ?? null
  if (min === null) return `≤ ${max} kg`
  if (max === null) return `> ${min} kg`
  return `${min}–${max} kg`
}

/** Alasan tombol "Loloskan" ditahan di UI (server tetap memvalidasi ulang). */
function checklistOf(p: Participant) {
  // Peserta lama tanpa kategori tidak diperiksa berkasnya (server juga melewatkan).
  if (!p.category_id) return []
  const standard = categoryById.value.get(p.category_id)?.preset_standard ?? 'ipsi'
  return docChecklist(p.age_group, p.gender, standard)
}

// ---- Tinjauan umur: tanggal lahir belum diisi / tidak sesuai golongan (umumnya peserta lama) ----
const birthDraft = reactive<Record<string, string>>({})
const savingBirthId = ref<string | null>(null)

function ageReviewMessage(p: Participant): string | null {
  if (p.age_check === 'missing_birth_date') {
    return 'Tanggal lahir belum diisi, sehingga golongan umur tidak dapat dipastikan.'
  }
  if (p.age_check === 'age_mismatch') {
    return `Usia atlet tidak sesuai golongan ${ageGroupLabel(p.age_group)}. Periksa tanggal lahir atau pindahkan ke kategori yang tepat.`
  }
  return null
}

async function saveBirthDate(p: Participant) {
  const value = (birthDraft[p.id] ?? '').trim()
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    showToast(
      'Tanggal Lahir Tidak Valid',
      'Pilih tanggal lahir lengkap (hari, bulan, tahun).',
      'danger',
    )
    return
  }
  savingBirthId.value = p.id
  try {
    await profileMutation.mutateAsync({ athleteId: p.athlete_id, birth_date: value })
    delete birthDraft[p.id]
    showToast('Tanggal Lahir Disimpan', `${p.name}: golongan umur diperiksa ulang.`, 'success')
  } catch (err) {
    showToast('Gagal Menyimpan', getErrorMessage(err, 'Terjadi kesalahan sistem.'), 'danger')
  } finally {
    savingBirthId.value = null
  }
}

function approveBlocker(p: Participant): string | null {
  if (p.age_check && p.category_id) return ageReviewMessage(p)
  const missing = p.category_id ? missingDocs(p.age_group, p.gender, deskOf(p).docs) : []
  if (missing.length > 0) return `Berkas wajib belum lengkap (${missing.length})`
  if (categoryHasWeightBounds(p) && parsedWeight(p) === null) return 'Isi hasil timbang badan'
  return null
}

function needsDesk(p: Participant): boolean {
  return p.registration_status !== 'verified' && p.registration_status !== 'rejected'
}

// UI Filters
const activeGenderTab = ref<'all' | 'male' | 'female'>('all')
const searchQuery = ref('')
const selectedClassFilter = ref<string>('all')

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

// Verification Actions
const processingId = ref<string | null>(null)

async function handleSetStatus(participant: Participant, newStatus: 'verified' | 'rejected') {
  processingId.value = participant.id
  try {
    const state = deskOf(participant)
    const weight = parsedWeight(participant)
    const result = await verifyMutation.mutateAsync({
      participantId: participant.id,
      status: newStatus,
      ...(weight !== null ? { actual_weight: weight } : {}),
      docs_verified: { ...state.docs },
    })
    if (result.status === 'overweight' || result.status === 'underweight') {
      // Server menerapkan aturan 0 kg toleransi: peserta tidak lolos timbang.
      showToast(
        result.status === 'overweight' ? 'OVERWEIGHT — Tidak Lolos' : 'UNDERWEIGHT — Tidak Lolos',
        `${participant.name}: ${result.reason || 'bobot di luar batas kelas'}.`,
        'danger',
      )
    } else {
      const actionLabel = newStatus === 'verified' ? 'dinyatakan Lolos Verifikasi' : 'ditolak'
      showToast(
        newStatus === 'verified' ? 'Verifikasi Berhasil' : 'Peserta Ditolak',
        `Peserta ${participant.name} ${actionLabel}.`,
        newStatus === 'verified' ? 'success' : 'danger',
      )
    }
  } catch (err) {
    showToast(
      'Gagal Memperbarui Status',
      getErrorMessage(err, 'Terjadi kesalahan sistem.'),
      'danger',
    )
  } finally {
    processingId.value = null
  }
}

// Stats
const stats = computed(() => {
  const all = participants.value
  const verified = all.filter((p) => p.registration_status === 'verified').length
  const pending = all.filter(
    (p) => p.registration_status === 'pending' || p.registration_status === 'registered',
  ).length
  const rejected = all.filter((p) => p.registration_status === 'rejected').length
  const weighFail = all.filter(
    (p) => p.registration_status === 'overweight' || p.registration_status === 'underweight',
  ).length
  return {
    total: all.length,
    verified,
    pending,
    rejected,
    weighFail,
  }
})

// Filtered List
const filteredParticipants = computed(() => {
  let list = [...participants.value]

  // Status Filter
  if (activeStatusFilter.value === 'verified') {
    list = list.filter((p) => p.registration_status === 'verified')
  } else if (activeStatusFilter.value === 'pending') {
    list = list.filter(
      (p) => p.registration_status === 'pending' || p.registration_status === 'registered',
    )
  } else if (activeStatusFilter.value === 'rejected') {
    list = list.filter((p) => p.registration_status === 'rejected')
  } else if (activeStatusFilter.value === 'weighfail') {
    list = list.filter(
      (p) => p.registration_status === 'overweight' || p.registration_status === 'underweight',
    )
  }

  // Gender Filter
  if (activeGenderTab.value === 'male') {
    list = list.filter((p) => p.gender === 'male' || p.gender === 'putra')
  } else if (activeGenderTab.value === 'female') {
    list = list.filter((p) => p.gender === 'female' || p.gender === 'putri')
  }

  // Class Filter
  if (selectedClassFilter.value !== 'all') {
    list = list.filter((p) => p.category_id === selectedClassFilter.value)
  }

  // Search Query
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase().trim()
    list = list.filter(
      (p) =>
        (p.name && p.name.toLowerCase().includes(q)) ||
        (p.club && p.club.toLowerCase().includes(q)) ||
        (p.category_name && p.category_name.toLowerCase().includes(q)) ||
        p.weight_class.toLowerCase().includes(q),
    )
  }

  return list
})

function isMaleGender(gender: string): boolean {
  return gender === 'male' || gender === 'putra'
}

function formatDate(iso: string): string {
  if (!iso) return '-'
  try {
    const d = new Date(iso)
    return d.toLocaleDateString('id-ID', {
      day: 'numeric',
      month: 'short',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return iso
  }
}
</script>

<template>
  <div
    class="verif-ops @container"
    role="region"
    aria-label="Meja Verifikasi Peserta & Timbang Badan"
  >
    <!-- Toast Notification -->
    <transition name="verif-fade">
      <div
        v-if="toast"
        class="verif-toast"
        :class="`verif-toast--${toast.type}`"
        role="status"
        aria-live="polite"
      >
        <div class="verif-toast__dot" />
        <div class="verif-toast__content">
          <strong class="verif-toast__title">{{ toast.title }}</strong>
          <span class="verif-toast__msg">{{ toast.message }}</span>
        </div>
        <button class="verif-toast__close" aria-label="Tutup notifikasi" @click="toast = null">
          ✕
        </button>
      </div>
    </transition>

    <!-- Header Banner -->
    <header class="verif-header">
      <div class="verif-header__left">
        <div class="verif-header__icon-box">
          <svg
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.2"
          >
            <path d="m9 11 3 3L22 4" />
            <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
          </svg>
        </div>
        <div>
          <div class="verif-header__sub font-mono">
            <span>TIMBANG BADAN & PEMERIKSAAN BERKAS</span>
            <span class="verif-bullet">•</span>
            <span class="verif-tag">IPSI 2023 / PERSILAT</span>
            <span class="verif-bullet">•</span>
            <span class="verif-pulse-wrapper">
              <span class="verif-pulse-dot" />
              <span>MEJA VERIFIKASI RESMI</span>
            </span>
          </div>
          <h2 class="verif-header__title">VERIFIKASI PESERTA TURNAMEN</h2>
        </div>
      </div>

      <div class="verif-header__actions font-mono">
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
          Segarkan Antrean
        </AppButton>
      </div>
    </header>

    <!-- Metrics Strip -->
    <section class="verif-metrics" aria-label="Statistik Verifikasi">
      <div class="verif-metric-card">
        <span class="verif-metric-label">TOTAL PESERTA</span>
        <div class="verif-metric-value-row">
          <span class="verif-metric-val tabular-nums">{{ stats.total }}</span>
          <span class="verif-metric-badge font-mono">PESERTA</span>
        </div>
        <span class="verif-metric-desc">Semua kontingen turnamen</span>
      </div>

      <div class="verif-metric-card verif-metric-card--verified">
        <div class="verif-metric-corner-tag font-mono">STATUS RESMI</div>
        <span class="verif-metric-label">LOLOS VERIFIKASI</span>
        <div class="verif-metric-value-row">
          <span class="verif-metric-val verif-val-emerald tabular-nums">{{ stats.verified }}</span>
          <span class="verif-metric-badge verif-badge-emerald font-mono">SIAP BAGAN</span>
        </div>
        <span class="verif-metric-desc">Lolos timbang & administrasi</span>
      </div>

      <div class="verif-metric-card verif-metric-card--pending">
        <div class="verif-metric-corner-tag font-mono">MENUNGGU</div>
        <span class="verif-metric-label">ANTREAN TIMBANG</span>
        <div class="verif-metric-value-row">
          <span class="verif-metric-val verif-val-amber tabular-nums">{{ stats.pending }}</span>
          <span class="verif-metric-badge verif-badge-amber font-mono">PENDING</span>
        </div>
        <span class="verif-metric-desc">Belum timbang badan</span>
      </div>

      <div class="verif-metric-card verif-metric-card--rejected">
        <div class="verif-metric-corner-tag font-mono">DISKUALIFIKASI</div>
        <span class="verif-metric-label">DITOLAK / DISKUALIFIKASI</span>
        <div class="verif-metric-value-row">
          <span class="verif-metric-val verif-val-red tabular-nums">{{ stats.rejected }}</span>
          <span class="verif-metric-badge verif-badge-red font-mono">REJECTED</span>
        </div>
        <span class="verif-metric-desc">Overweight / berkas tidak sesuai</span>
      </div>
    </section>

    <!-- Controls Toolbar -->
    <section class="verif-toolbar">
      <div class="verif-toolbar__tabs font-mono" role="tablist">
        <button
          class="verif-tab-btn"
          :class="{ 'verif-tab-btn--active': activeStatusFilter === 'all' }"
          role="tab"
          :aria-selected="activeStatusFilter === 'all'"
          @click="activeStatusFilter = 'all'"
        >
          Semua ({{ stats.total }})
        </button>

        <button
          class="verif-tab-btn"
          :class="{ 'verif-tab-btn--active-pending': activeStatusFilter === 'pending' }"
          role="tab"
          :aria-selected="activeStatusFilter === 'pending'"
          @click="activeStatusFilter = 'pending'"
        >
          <span class="verif-tab-dot verif-dot-pending" />
          Menunggu ({{ stats.pending }})
        </button>

        <button
          class="verif-tab-btn"
          :class="{ 'verif-tab-btn--active-verified': activeStatusFilter === 'verified' }"
          role="tab"
          :aria-selected="activeStatusFilter === 'verified'"
          @click="activeStatusFilter = 'verified'"
        >
          <span class="verif-tab-dot verif-dot-verified" />
          Lolos ({{ stats.verified }})
        </button>

        <button
          class="verif-tab-btn"
          :class="{ 'verif-tab-btn--active-rejected': activeStatusFilter === 'rejected' }"
          role="tab"
          :aria-selected="activeStatusFilter === 'rejected'"
          @click="activeStatusFilter = 'rejected'"
        >
          <span class="verif-tab-dot verif-dot-rejected" />
          Ditolak ({{ stats.rejected }})
        </button>

        <button
          class="verif-tab-btn"
          :class="{ 'verif-tab-btn--active-rejected': activeStatusFilter === 'weighfail' }"
          role="tab"
          :aria-selected="activeStatusFilter === 'weighfail'"
          @click="activeStatusFilter = 'weighfail'"
        >
          Gagal Timbang ({{ stats.weighFail }})
        </button>
      </div>

      <div class="verif-toolbar__search-group">
        <!-- Class Filter -->
        <div class="verif-filter-wrapper">
          <select
            v-model="selectedClassFilter"
            class="verif-select font-mono"
            aria-label="Filter Kategori"
          >
            <option value="all">Semua Kategori</option>
            <option v-for="c in sortedCategories" :key="c.id" :value="c.id">{{ c.name }}</option>
          </select>
        </div>

        <!-- Gender Filter -->
        <div class="verif-filter-wrapper">
          <select
            v-model="activeGenderTab"
            class="verif-select font-mono"
            aria-label="Filter Gender"
          >
            <option value="all">Semua Gender</option>
            <option value="male">Putra</option>
            <option value="female">Putri</option>
          </select>
        </div>

        <!-- Search Input -->
        <div class="verif-search-box">
          <svg
            class="verif-search-icon"
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
            class="verif-search-input font-mono"
            placeholder="Cari peserta / tim..."
            aria-label="Cari peserta"
          />
          <button
            v-if="searchQuery"
            class="verif-search-clear"
            aria-label="Hapus filter"
            @click="searchQuery = ''"
          >
            ✕
          </button>
        </div>
      </div>
    </section>

    <!-- Loading Skeleton -->
    <div v-if="isLoading" class="verif-grid" aria-busy="true">
      <div v-for="i in 6" :key="i" class="verif-skeleton-card">
        <div class="skeleton-line skeleton-header" />
        <div class="skeleton-line skeleton-title" />
        <div class="skeleton-line skeleton-specs" />
        <div class="skeleton-line skeleton-btn" />
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="verif-empty-state verif-empty-state--error">
      <div class="verif-empty-icon verif-empty-icon--error">
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
      <h3 class="verif-empty-title">Gagal Memuat Antrean Verifikasi</h3>
      <p class="verif-empty-desc">
        {{ getErrorMessage(error, 'Terjadi gangguan jaringan saat memuat antrean.') }}
      </p>
      <AppButton variant="secondary" size="md" @click="() => refetch()"> Coba Lagi </AppButton>
    </div>

    <!-- Empty State -->
    <div v-else-if="filteredParticipants.length === 0" class="verif-empty-state">
      <div class="verif-empty-icon">
        <svg
          width="36"
          height="36"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.8"
        >
          <circle cx="12" cy="12" r="10" />
          <path d="m9 12 2 2 4-4" />
        </svg>
      </div>
      <h3 class="verif-empty-title">
        {{ searchQuery ? 'Tidak Ada Peserta yang Cocok' : 'Tidak Ada Antrean Verifikasi' }}
      </h3>
      <p class="verif-empty-desc">
        {{
          searchQuery
            ? `Pencarian "${searchQuery}" tidak menghasilkan data peserta.`
            : 'Belum ada antrean peserta pada status ini atau semua peserta telah diverifikasi.'
        }}
      </p>
      <AppButton v-if="searchQuery" variant="secondary" size="md" @click="searchQuery = ''">
        Reset Pencarian
      </AppButton>
    </div>

    <!-- Responsive Cards Grid -->
    <div v-else class="verif-grid" role="list">
      <article
        v-for="part in filteredParticipants"
        :key="part.id"
        class="verif-card"
        :class="`verif-card--status-${part.registration_status}`"
        role="listitem"
      >
        <!-- Corner Accent Stripe -->
        <div class="verif-card__accent" :class="`verif-accent--${part.registration_status}`" />

        <!-- Card Header: Status & Class -->
        <header class="verif-card__header">
          <!-- Status Badge -->
          <div
            class="verif-status-badge font-mono"
            :class="`verif-status-badge--${part.registration_status}`"
          >
            <svg
              v-if="part.registration_status === 'verified'"
              width="12"
              height="12"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="3"
            >
              <polyline points="20 6 9 17 4 12" />
            </svg>
            <svg
              v-else-if="part.registration_status === 'rejected'"
              width="12"
              height="12"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="3"
            >
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
            <span v-else class="verif-status-dot-pending" />
            <span>
              {{
                part.registration_status === 'verified'
                  ? 'LOLOS VERIFIKASI'
                  : part.registration_status === 'rejected'
                    ? 'DITOLAK'
                    : part.registration_status === 'overweight'
                      ? 'OVERWEIGHT'
                      : part.registration_status === 'underweight'
                        ? 'UNDERWEIGHT'
                        : 'MENUNGGU VERIFIKASI'
              }}
            </span>
          </div>

          <!-- Class Code Badge -->
          <div class="verif-class-tag font-mono">
            <span>KELAS</span>
            <strong>{{ part.weight_class || '—' }}</strong>
          </div>
        </header>

        <!-- Card Content: Athlete Info -->
        <div class="verif-card__content">
          <div class="verif-card__gender-row font-mono">
            <span
              class="verif-gender-chip"
              :class="
                isMaleGender(part.gender) ? 'verif-gender-chip--male' : 'verif-gender-chip--female'
              "
            >
              {{ part.age_group ? ageGroupLabel(part.age_group).toUpperCase() : 'DEWASA' }}
              {{ isMaleGender(part.gender) ? 'PUTRA (MERAH)' : 'PUTRI (BIRU)' }}
            </span>
          </div>

          <h3 class="verif-card__title">{{ part.name || 'Nama Tidak Tercatat' }}</h3>

          <div class="verif-card__spec-box font-mono">
            <div class="verif-spec-item">
              <span class="verif-spec-label">PERGURUAN / KONTINGEN:</span>
              <strong class="verif-spec-value">{{ part.club || 'Kontingen Mandiri' }}</strong>
            </div>
            <div class="verif-spec-divider" />
            <div class="verif-spec-item">
              <span class="verif-spec-label">TANGGAL REGISTRASI:</span>
              <span class="verif-spec-sub tabular-nums">{{ formatDate(part.created_at) }}</span>
            </div>
          </div>

          <!-- Tinjauan umur (data lama tanpa tanggal lahir / usia tidak sesuai golongan) -->
          <section
            v-if="part.age_check"
            class="verif-age font-mono"
            :data-test="`age-review-${part.age_check}`"
          >
            <p class="verif-age__msg">⚠ {{ ageReviewMessage(part) }}</p>
            <div class="verif-age__row">
              <label class="verif-age__label" :for="`birth-${part.id}`">TANGGAL LAHIR</label>
              <input
                :id="`birth-${part.id}`"
                v-model="birthDraft[part.id]"
                type="date"
                class="verif-age__input"
                :max="new Date().toISOString().slice(0, 10)"
              />
              <AppButton
                variant="secondary"
                size="sm"
                :loading="savingBirthId === part.id"
                :disabled="!birthDraft[part.id]"
                @click="saveBirthDate(part)"
              >
                Simpan
              </AppButton>
            </div>
          </section>

          <!-- Meja Timbang & Screening Berkas -->
          <section v-if="needsDesk(part)" class="verif-desk font-mono">
            <div v-if="part.category_name" class="verif-desk__cat">
              {{ part.category_name }}
            </div>
            <div class="verif-desk__row">
              <label class="verif-desk__label" :for="`weigh-${part.id}`">
                HASIL TIMBANG (KG)
                <span class="verif-desk__range">Batas: {{ weightRangeLabel(part) }}</span>
              </label>
              <input
                :id="`weigh-${part.id}`"
                v-model="deskOf(part).weight"
                type="text"
                inputmode="decimal"
                class="verif-desk__input tabular-nums"
                placeholder="0.0"
                autocomplete="off"
              />
              <span
                v-if="weightVerdict(part) === 'ok'"
                class="verif-weight-badge verif-weight-badge--ok"
                >🟢 SAH</span
              >
              <span
                v-else-if="weightVerdict(part) === 'overweight'"
                class="verif-weight-badge verif-weight-badge--bad"
                >🔴 OVERWEIGHT</span
              >
              <span
                v-else-if="weightVerdict(part) === 'underweight'"
                class="verif-weight-badge verif-weight-badge--bad"
                >🔴 UNDERWEIGHT</span
              >
            </div>
            <p class="verif-desk__note">Tanpa toleransi: selisih 0,1 kg pun diskualifikasi.</p>

            <fieldset v-if="checklistOf(part).length > 0" class="verif-desk__docs">
              <legend>Berkas verifikasi</legend>
              <label v-for="d in checklistOf(part)" :key="d.key" class="verif-desk__check">
                <input v-model="deskOf(part).docs[d.key]" type="checkbox" />
                <span
                  >{{ d.label }}<strong v-if="d.required" class="verif-desk__req"> *</strong></span
                >
              </label>
            </fieldset>
            <p v-if="approveBlocker(part)" class="verif-desk__blocker">
              ⚠ {{ approveBlocker(part) }}
            </p>
          </section>
        </div>

        <!-- Card Footer: Verification Actions -->
        <footer class="verif-card__footer font-mono">
          <!-- State 1: Verified (Lolos) -->
          <template v-if="part.registration_status === 'verified'">
            <div class="verif-verified-label">
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2.5"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
              <span>Sah & Masuk Bagan</span>
            </div>
            <button
              type="button"
              class="verif-btn-reject-action"
              :disabled="processingId === part.id"
              @click="handleSetStatus(part, 'rejected')"
            >
              <svg
                width="13"
                height="13"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2.5"
              >
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
              Tolak Peserta
            </button>
          </template>

          <!-- State 2: Rejected (Ditolak) -->
          <template v-else-if="part.registration_status === 'rejected'">
            <div class="verif-rejected-label">
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2.5"
              >
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
              <span>Status: Ditolak</span>
            </div>
            <button
              type="button"
              class="verif-btn-approve-action"
              :disabled="processingId === part.id"
              @click="handleSetStatus(part, 'verified')"
            >
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
              Loloskan Peserta
            </button>
          </template>

          <!-- State 3: Pending / Registered (Menunggu) -->
          <template v-else>
            <button
              type="button"
              class="verif-action-btn verif-btn-reject"
              :disabled="processingId === part.id"
              @click="handleSetStatus(part, 'rejected')"
            >
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2.5"
              >
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
              Tolak
            </button>

            <button
              type="button"
              class="verif-action-btn verif-btn-approve"
              :disabled="processingId === part.id || !!approveBlocker(part)"
              :title="approveBlocker(part) || undefined"
              @click="handleSetStatus(part, 'verified')"
            >
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2.5"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
              {{
                weightVerdict(part) === 'overweight' || weightVerdict(part) === 'underweight'
                  ? 'Catat Hasil Timbang'
                  : 'Loloskan'
              }}
            </button>
          </template>
        </footer>
      </article>
    </div>
  </div>
</template>

<style scoped>
.verif-ops {
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: 100%;
  color: var(--text-main, #f8fafc);
  box-sizing: border-box;
}

/* Toast */
.verif-toast {
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
.verif-toast--success {
  border-left: 4px solid var(--color-emerald-500, #10b981);
}
.verif-toast--danger {
  border-left: 4px solid var(--color-primary-600, #dc2626);
}
.verif-toast--info {
  border-left: 4px solid var(--color-secondary-600, #2563eb);
}
.verif-toast__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: currentColor;
}
.verif-toast__content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.verif-toast__title {
  font-size: 0.875rem;
  font-weight: 700;
}
.verif-toast__msg {
  font-size: 0.8125rem;
  color: var(--color-neutral-400, #94a3b8);
}
.verif-toast__close {
  background: none;
  border: none;
  color: var(--color-neutral-500, #64748b);
  cursor: pointer;
  font-size: 1rem;
}
.verif-fade-enter-active,
.verif-fade-leave-active {
  transition:
    opacity 0.2s ease,
    transform 0.2s ease;
}
.verif-fade-enter-from,
.verif-fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* Header */
.verif-header {
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
.verif-header__left {
  display: flex;
  align-items: center;
  gap: 16px;
}
.verif-header__icon-box {
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
.verif-header__sub {
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
.verif-bullet {
  color: var(--color-neutral-600, #475569);
}
.verif-tag {
  color: var(--color-emerald-400, #34d399);
}
.verif-pulse-wrapper {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--color-emerald-400, #34d399);
}
.verif-pulse-dot {
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
.verif-header__title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.5rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #ffffff;
  line-height: 1.2;
}
.verif-header__actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

/* Metrics */
.verif-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 12px;
}
.verif-metric-card {
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
.verif-metric-card--verified {
  border-left: 3px solid var(--color-emerald-500, #10b981);
}
.verif-metric-card--pending {
  border-left: 3px solid #f59e0b;
}
.verif-metric-card--rejected {
  border-left: 3px solid var(--color-primary-600, #dc2626);
}
.verif-metric-corner-tag {
  position: absolute;
  top: 10px;
  right: 12px;
  font-size: 0.625rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--color-neutral-500, #64748b);
}
.verif-metric-label {
  font-family: var(--font-mono, monospace);
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--color-neutral-400, #94a3b8);
}
.verif-metric-value-row {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-top: 2px;
}
.verif-metric-val {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.875rem;
  font-weight: 800;
  color: #ffffff;
  line-height: 1;
}
.verif-val-emerald {
  color: #34d399;
}
.verif-val-amber {
  color: #fbbf24;
}
.verif-val-red {
  color: #f87171;
}
.verif-metric-badge {
  font-size: 0.6875rem;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--color-neutral-400, #94a3b8);
}
.verif-badge-emerald {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
}
.verif-badge-amber {
  background: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
}
.verif-badge-red {
  background: rgba(220, 38, 38, 0.15);
  color: #f87171;
}
.verif-metric-desc {
  font-size: 0.75rem;
  color: var(--color-neutral-500, #64748b);
  margin-top: 2px;
}

/* Toolbar */
.verif-toolbar {
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
.verif-toolbar__tabs {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.verif-tab-btn {
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
.verif-tab-btn:hover {
  background: rgba(255, 255, 255, 0.05);
  color: #ffffff;
}
.verif-tab-btn--active {
  background: rgba(255, 255, 255, 0.1);
  color: #ffffff;
  border-color: rgba(255, 255, 255, 0.15);
}
.verif-tab-btn--active-pending {
  background: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
  border-color: rgba(245, 158, 11, 0.4);
}
.verif-tab-btn--active-verified {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
  border-color: rgba(16, 185, 129, 0.4);
}
.verif-tab-btn--active-rejected {
  background: rgba(220, 38, 38, 0.15);
  color: #fca5a5;
  border-color: rgba(220, 38, 38, 0.4);
}
.verif-tab-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}
.verif-dot-pending {
  background-color: #f59e0b;
}
.verif-dot-verified {
  background-color: var(--color-emerald-400, #34d399);
}
.verif-dot-rejected {
  background-color: var(--color-primary-500, #ef4444);
}

.verif-toolbar__search-group {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  flex: 1;
  max-width: 620px;
  justify-content: flex-end;
}
.verif-select {
  padding: 8px 12px;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  color: var(--color-neutral-300, #cbd5e1);
  font-size: 0.75rem;
  outline: none;
  cursor: pointer;
}
.verif-search-box {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 200px;
}
.verif-search-icon {
  position: absolute;
  left: 10px;
  color: var(--color-neutral-500, #64748b);
  pointer-events: none;
}
.verif-search-input {
  width: 100%;
  padding: 8px 32px 8px 34px;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  color: #ffffff;
  font-size: 0.75rem;
  outline: none;
}
.verif-search-input:focus {
  border-color: var(--color-emerald-400, #34d399);
}
.verif-search-clear {
  position: absolute;
  right: 10px;
  background: none;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  cursor: pointer;
  font-size: 0.8125rem;
}

/* Grid */
.verif-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(310px, 1fr));
  gap: 16px;
}
@media (max-width: 640px) {
  .verif-grid {
    grid-template-columns: 1fr;
  }
}

/* Card */
.verif-card {
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
.verif-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.35);
}
.verif-card__accent {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
}
.verif-accent--verified {
  background: linear-gradient(90deg, #059669, #10b981);
}
.verif-accent--pending,
.verif-accent--registered {
  background: linear-gradient(90deg, #d97706, #f59e0b);
}
.verif-accent--rejected {
  background: linear-gradient(90deg, #dc2626, #ef4444);
}

.verif-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 18px 10px 18px;
}
.verif-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.625rem;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 4px;
  letter-spacing: 0.05em;
}
.verif-status-badge--verified {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
  border: 1px solid rgba(16, 185, 129, 0.3);
}
.verif-status-badge--pending,
.verif-status-badge--registered {
  background: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
  border: 1px solid rgba(245, 158, 11, 0.3);
}
.verif-status-badge--rejected {
  background: rgba(220, 38, 38, 0.15);
  color: #f87171;
  border: 1px solid rgba(220, 38, 38, 0.3);
}
.verif-status-dot-pending {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #fbbf24;
}

.verif-class-tag {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  font-size: 0.5625rem;
  color: var(--color-neutral-400, #94a3b8);
}
.verif-class-tag strong {
  font-size: 1.125rem;
  font-weight: 800;
  color: #ffffff;
  line-height: 1;
}

.verif-card__content {
  padding: 0 18px 14px 18px;
}
.verif-card__gender-row {
  margin-bottom: 6px;
}
.verif-gender-chip {
  font-size: 0.625rem;
  font-weight: 700;
  letter-spacing: 0.05em;
}
.verif-gender-chip--male {
  color: #fca5a5;
}
.verif-gender-chip--female {
  color: #93c5fd;
}
.verif-card__title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.125rem;
  font-weight: 700;
  color: #ffffff;
  line-height: 1.3;
  margin-bottom: 8px;
}
.verif-card__spec-box {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 12px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 6px;
}
.verif-spec-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.verif-spec-label {
  font-size: 0.5625rem;
  color: var(--color-neutral-500, #64748b);
}
.verif-spec-value {
  font-size: 0.75rem;
  color: #ffffff;
  font-weight: 600;
}
.verif-spec-sub {
  font-size: 0.6875rem;
  color: var(--color-neutral-400, #94a3b8);
}
.verif-spec-divider {
  height: 1px;
  background: rgba(255, 255, 255, 0.06);
  margin: 2px 0;
}

.verif-card__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 18px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(0, 0, 0, 0.15);
}
.verif-verified-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--color-emerald-400, #34d399);
}
.verif-rejected-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.6875rem;
  font-weight: 700;
  color: #f87171;
}
.verif-btn-reject-action {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: rgba(239, 68, 68, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #fca5a5;
  font-size: 0.75rem;
  font-weight: 700;
  padding: 6px 14px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.verif-btn-reject-action:hover:not(:disabled) {
  background: rgba(239, 68, 68, 0.25);
  border-color: rgba(239, 68, 68, 0.55);
  color: #ffffff;
}
.verif-btn-approve-action {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.3);
  color: #6ee7b7;
  font-size: 0.75rem;
  font-weight: 700;
  padding: 6px 14px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.verif-btn-approve-action:hover:not(:disabled) {
  background: rgba(16, 185, 129, 0.25);
  border-color: rgba(16, 185, 129, 0.55);
  color: #ffffff;
}
.verif-action-btn {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 7px 12px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
  border: 1px solid transparent;
  transition: all 0.15s ease;
}
.verif-btn-reject {
  background: rgba(220, 38, 38, 0.12);
  color: #fca5a5;
  border-color: rgba(220, 38, 38, 0.25);
}
.verif-btn-reject:hover {
  background: rgba(220, 38, 38, 0.25);
  color: #ffffff;
  border-color: rgba(220, 38, 38, 0.5);
}
.verif-btn-approve {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
  border-color: rgba(16, 185, 129, 0.35);
}
.verif-btn-approve:hover {
  background: rgba(16, 185, 129, 0.3);
  color: #ffffff;
  border-color: rgba(16, 185, 129, 0.6);
}

/* Skeleton */
.verif-skeleton-card {
  height: 180px;
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
  width: 45%;
  height: 20px;
}
.skeleton-title {
  width: 75%;
  height: 24px;
}
.skeleton-specs {
  width: 100%;
  height: 36px;
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
.verif-empty-state {
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
.verif-empty-icon {
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
.verif-empty-icon--error {
  background: rgba(220, 38, 38, 0.15);
  color: var(--color-primary-400, #f87171);
}
.verif-empty-title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.25rem;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 8px;
}
.verif-empty-desc {
  max-width: 520px;
  font-size: 0.875rem;
  color: var(--color-neutral-400, #94a3b8);
  line-height: 1.5;
  margin-bottom: 20px;
}
.verif-desk {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 12px;
  padding: 12px;
  border-radius: var(--radius-md, 8px);
  border: 1px dashed var(--border-card, #334155);
  font-size: 0.75rem;
}
.verif-desk__cat {
  opacity: 0.75;
  font-size: 0.7rem;
}
.verif-desk__row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.verif-desk__label {
  display: flex;
  flex-direction: column;
  flex: 1 1 140px;
  gap: 2px;
  font-weight: 600;
}
.verif-desk__range {
  font-weight: 400;
  opacity: 0.7;
}
.verif-desk__input {
  width: 88px;
  padding: 6px 8px;
  border-radius: 6px;
  border: 1px solid var(--border-card, #334155);
  background: transparent;
  color: inherit;
  text-align: right;
}
.verif-weight-badge {
  padding: 3px 8px;
  border-radius: 999px;
  font-weight: 700;
  white-space: nowrap;
}
.verif-weight-badge--ok {
  background: rgba(16, 185, 129, 0.16);
  color: var(--color-emerald-500, #10b981);
}
.verif-weight-badge--bad {
  background: rgba(220, 38, 38, 0.16);
  color: var(--color-primary-600, #dc2626);
}
.verif-desk__note {
  margin: 0;
  opacity: 0.6;
}
.verif-desk__docs {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding: 8px 10px;
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
}
.verif-desk__docs legend {
  padding: 0 6px;
  opacity: 0.75;
}
.verif-desk__req {
  color: var(--color-primary-600, #dc2626);
}
.verif-desk__check {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  cursor: pointer;
}
.verif-age {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid rgba(245, 158, 11, 0.45);
  background: rgba(245, 158, 11, 0.08);
  font-size: 0.75rem;
}
.verif-age__msg {
  margin: 0;
  color: var(--color-amber-500, #f59e0b);
}
.verif-age__row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.verif-age__label {
  color: var(--color-slate-400, #94a3b8);
  letter-spacing: 0.06em;
}
.verif-age__input {
  min-height: 36px;
  padding: 4px 8px;
  border-radius: 6px;
  border: 1px solid var(--border-card, #334155);
  background: transparent;
  color: inherit;
  font: inherit;
}
.verif-desk__blocker {
  margin: 0;
  color: var(--color-amber-500, #f59e0b);
}
.verif-status-badge--overweight,
.verif-status-badge--underweight {
  background: rgba(220, 38, 38, 0.16);
  color: var(--color-primary-600, #dc2626);
}
</style>
