<script setup lang="ts">
import { getErrorMessage } from '@/api/error'
import { ref, computed } from 'vue'
import { AppButton } from '@/components/ui'
import {
  useCompetitionAthletesQuery,
  useContingentsQuery,
  useRegisterToCompetitionMutation,
} from '../api/athleteQueries'
import { createAthleteSchema } from '../types/athlete.schema'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

const props = withDefaults(
  defineProps<{
    competitionId?: string
  }>(),
  {
    competitionId: 'b26d7145-d341-4795-ac58-bff86d13cbf8',
  },
)

const emit = defineEmits<{
  (e: 'navigateVerification'): void
}>()

const activeCompId = computed(() => props.competitionId || 'b26d7145-d341-4795-ac58-bff86d13cbf8')

// Queries
const {
  data: rawAthletes,
  isLoading,
  isFetching,
  error,
  refetch,
} = useCompetitionAthletesQuery(activeCompId)
const { data: rawContingents } = useContingentsQuery()

const registerMutation = useRegisterToCompetitionMutation(activeCompId)

const athletes = computed(() => rawAthletes.value || [])
const contingents = computed(() => rawContingents.value || [])

// UI Filters
const activeGenderTab = ref<'all' | 'male' | 'female'>('all')
const selectedContingentFilter = ref<string>('all')
const searchQuery = ref('')
const sortBy = ref<'name' | 'team'>('name')

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

// Modal Form State (Register New Athlete)
const isModalOpen = ref(false)
const formErrors = ref<Record<string, string>>({})
const formState = ref<{
  name: string
  gender: 'male' | 'female' | 'putra' | 'putri'
  team: string
  weight_class: string
}>({
  name: '',
  gender: 'male',
  team: '',
  weight_class: 'A',
})

function openCreateModal() {
  formState.value = {
    name: '',
    gender: 'male',
    team: contingents.value.length > 0 ? contingents.value[0].name : '',
    weight_class: 'A',
  }
  formErrors.value = {}
  isModalOpen.value = true
}

async function handleSubmitAthlete() {
  formErrors.value = {}
  const parseResult = createAthleteSchema.safeParse({
    name: formState.value.name,
    gender: formState.value.gender,
    team: formState.value.team,
  })

  if (!parseResult.success) {
    const errors: Record<string, string> = {}
    for (const issue of parseResult.error.issues) {
      const field = issue.path[0] as string
      if (!errors[field]) errors[field] = issue.message
    }
    formErrors.value = errors
    return
  }

  try {
    // 1. Create or register athlete directly to tournament
    await registerMutation.mutateAsync({
      name: formState.value.name.trim(),
      gender: formState.value.gender,
      team: formState.value.team.trim(),
      weight_class: formState.value.weight_class.trim().toUpperCase(),
    })

    isModalOpen.value = false
    showToast(
      'Atlet Berhasil Didaftarkan',
      `${formState.value.name} telah didaftarkan ke kompetisi.`,
      'success',
    )
    refetch()
  } catch (err) {
    showToast(
      'Gagal Mendaftarkan Atlet',
      getErrorMessage(err, 'Terjadi kesalahan sistem.'),
      'danger',
    )
  }
}

// Stats
const stats = computed(() => {
  const all = athletes.value
  const male = all.filter((a) => a.gender === 'male' || a.gender === 'putra').length
  const female = all.filter((a) => a.gender === 'female' || a.gender === 'putri').length
  const distinctTeams = new Set(all.map((a) => a.team).filter(Boolean)).size
  return {
    total: all.length,
    male,
    female,
    teams: distinctTeams,
  }
})

// Filtered List
const filteredAthletes = computed(() => {
  let list = [...athletes.value]

  // Gender
  if (activeGenderTab.value === 'male') {
    list = list.filter((a) => a.gender === 'male' || a.gender === 'putra')
  } else if (activeGenderTab.value === 'female') {
    list = list.filter((a) => a.gender === 'female' || a.gender === 'putri')
  }

  // Contingent
  if (selectedContingentFilter.value !== 'all') {
    list = list.filter((a) => a.team === selectedContingentFilter.value)
  }

  // Search
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase().trim()
    list = list.filter(
      (a) => a.name.toLowerCase().includes(q) || (a.team && a.team.toLowerCase().includes(q)),
    )
  }

  // Sort
  list.sort((a, b) => {
    if (sortBy.value === 'team') {
      return (a.team || '').localeCompare(b.team || '')
    }
    return a.name.localeCompare(b.name)
  })

  return list
})

function isMaleGender(gender: string): boolean {
  return gender === 'male' || gender === 'putra'
}
</script>

<template>
  <div class="atl-ops @container" role="region" aria-label="Manajemen Data Atlet dan Kontingen">
    <!-- Toast Notification -->
    <transition name="atl-fade">
      <div
        v-if="toast"
        class="atl-toast"
        :class="`atl-toast--${toast.type}`"
        role="status"
        aria-live="polite"
      >
        <div class="atl-toast__dot" />
        <div class="atl-toast__content">
          <strong class="atl-toast__title">{{ toast.title }}</strong>
          <span class="atl-toast__msg">{{ toast.message }}</span>
        </div>
        <button class="atl-toast__close" aria-label="Tutup notifikasi" @click="toast = null">
          ✕
        </button>
      </div>
    </transition>

    <!-- Header Banner -->
    <header class="atl-header">
      <div class="atl-header__left">
        <div class="atl-header__icon-box">
          <svg
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.2"
          >
            <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
            <circle cx="9" cy="7" r="4" />
            <path d="M22 21v-2a4 4 0 0 0-3-3.87" />
            <path d="M16 3.13a4 4 0 0 1 0 7.75" />
          </svg>
        </div>
        <div>
          <div class="atl-header__sub font-mono">
            <span>DATABASE KONTINGEN & PESERTA</span>
            <span class="atl-bullet">•</span>
            <span class="atl-tag">IPSI & PERSILAT</span>
            <span class="atl-bullet">•</span>
            <span class="atl-pulse-wrapper">
              <span class="atl-pulse-dot" />
              <span>LIVE REPO</span>
            </span>
          </div>
          <h2 class="atl-header__title">DATA ATLET & KONTINGEN</h2>
        </div>
      </div>

      <div class="atl-header__actions font-mono">
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
          Segarkan
        </AppButton>

        <AppButton
          v-if="authStore.isAdminOrDewan"
          variant="secondary"
          size="sm"
          @click="emit('navigateVerification')"
        >
          <template #icon>
            <svg
              width="15"
              height="15"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="m9 11 3 3L22 4" />
              <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
            </svg>
          </template>
          Buka Verifikasi Peserta
        </AppButton>

        <AppButton variant="emerald" size="sm" @click="openCreateModal">
          <template #icon>
            <svg
              width="15"
              height="15"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
            >
              <line x1="12" y1="5" x2="12" y2="19" />
              <line x1="5" y1="12" x2="19" y2="12" />
            </svg>
          </template>
          Daftarkan Atlet
        </AppButton>
      </div>
    </header>

    <!-- Metrics Strip -->
    <section class="atl-metrics" aria-label="Statistik Atlet">
      <div class="atl-metric-card">
        <span class="atl-metric-label">TOTAL ATLET TERDAFTAR</span>
        <div class="atl-metric-value-row">
          <span class="atl-metric-val tabular-nums">{{ stats.total }}</span>
          <span class="atl-metric-badge font-mono">PESERTA</span>
        </div>
        <span class="atl-metric-desc">Terdaftar dalam turnamen ini</span>
      </div>

      <div class="atl-metric-card atl-metric-card--male">
        <div class="atl-metric-corner-tag font-mono">SUDUT MERAH</div>
        <span class="atl-metric-label">ATLET PUTRA</span>
        <div class="atl-metric-value-row">
          <span class="atl-metric-val atl-val-male tabular-nums">{{ stats.male }}</span>
          <span class="atl-metric-badge atl-badge-male font-mono">PUTRA</span>
        </div>
        <span class="atl-metric-desc">Peserta tanding putra</span>
      </div>

      <div class="atl-metric-card atl-metric-card--female">
        <div class="atl-metric-corner-tag font-mono">SUDUT BIRU</div>
        <span class="atl-metric-label">ATLET PUTRI</span>
        <div class="atl-metric-value-row">
          <span class="atl-metric-val atl-val-female tabular-nums">{{ stats.female }}</span>
          <span class="atl-metric-badge atl-badge-female font-mono">PUTRI</span>
        </div>
        <span class="atl-metric-desc">Peserta tanding putri</span>
      </div>

      <div class="atl-metric-card">
        <span class="atl-metric-label">KONTINGEN / TIM</span>
        <div class="atl-metric-value-row">
          <span class="atl-metric-val-text font-mono tabular-nums">{{ stats.teams }}</span>
          <span class="atl-metric-badge font-mono">PERGURUAN</span>
        </div>
        <span class="atl-metric-desc">Kontingen resmi berpartisipasi</span>
      </div>
    </section>

    <!-- Controls Toolbar -->
    <section class="atl-toolbar">
      <div class="atl-toolbar__tabs font-mono" role="tablist">
        <button
          class="atl-tab-btn"
          :class="{ 'atl-tab-btn--active': activeGenderTab === 'all' }"
          role="tab"
          :aria-selected="activeGenderTab === 'all'"
          @click="activeGenderTab = 'all'"
        >
          Semua Atlet
          <span class="atl-tab-badge tabular-nums">{{ stats.total }}</span>
        </button>

        <button
          class="atl-tab-btn"
          :class="{ 'atl-tab-btn--active-male': activeGenderTab === 'male' }"
          role="tab"
          :aria-selected="activeGenderTab === 'male'"
          @click="activeGenderTab = 'male'"
        >
          <span class="atl-tab-dot atl-dot-male" />
          Putra
          <span class="atl-tab-badge tabular-nums">{{ stats.male }}</span>
        </button>

        <button
          class="atl-tab-btn"
          :class="{ 'atl-tab-btn--active-female': activeGenderTab === 'female' }"
          role="tab"
          :aria-selected="activeGenderTab === 'female'"
          @click="activeGenderTab = 'female'"
        >
          <span class="atl-tab-dot atl-dot-female" />
          Putri
          <span class="atl-tab-badge tabular-nums">{{ stats.female }}</span>
        </button>
      </div>

      <div class="atl-toolbar__search-group">
        <div class="atl-filter-select-wrapper">
          <select
            v-model="selectedContingentFilter"
            class="atl-filter-select font-mono"
            aria-label="Filter Kontingen"
          >
            <option value="all">Semua Kontingen / Tim</option>
            <option v-for="c in contingents" :key="c.id" :value="c.name">
              {{ c.name }}
            </option>
          </select>
        </div>

        <div class="atl-search-box">
          <svg
            class="atl-search-icon"
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
            class="atl-search-input font-mono"
            placeholder="Cari atlet / perguruan..."
            aria-label="Pencarian atlet"
          />
          <button
            v-if="searchQuery"
            class="atl-search-clear"
            aria-label="Hapus filter"
            @click="searchQuery = ''"
          >
            ✕
          </button>
        </div>

        <div class="atl-sort-select-wrapper">
          <select v-model="sortBy" class="atl-sort-select font-mono" aria-label="Urutkan Atlet">
            <option value="name">Urut: Nama Atlet (A-Z)</option>
            <option value="team">Urut: Perguruan / Tim</option>
          </select>
        </div>
      </div>
    </section>

    <!-- Loading Skeleton -->
    <div v-if="isLoading" class="atl-grid" aria-busy="true">
      <div v-for="i in 6" :key="i" class="atl-skeleton-card">
        <div class="skeleton-line skeleton-header" />
        <div class="skeleton-line skeleton-title" />
        <div class="skeleton-line skeleton-specs" />
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="atl-empty-state atl-empty-state--error">
      <div class="atl-empty-icon atl-empty-icon--error">
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
      <h3 class="atl-empty-title">Gagal Memuat Data Atlet</h3>
      <p class="atl-empty-desc">
        {{ getErrorMessage(error, 'Terjadi gangguan jaringan saat memuat data atlet.') }}
      </p>
      <AppButton variant="secondary" size="md" @click="() => refetch()"> Coba Lagi </AppButton>
    </div>

    <!-- Empty State -->
    <div v-else-if="filteredAthletes.length === 0" class="atl-empty-state">
      <div class="atl-empty-icon">
        <svg
          width="36"
          height="36"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.8"
        >
          <circle cx="12" cy="7" r="4" />
          <path d="M6 21v-2a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v2" />
        </svg>
      </div>
      <h3 class="atl-empty-title">
        {{ searchQuery ? 'Tidak Ada Atlet yang Cocok' : 'Belum Ada Atlet Terdaftar' }}
      </h3>
      <p class="atl-empty-desc">
        {{
          searchQuery
            ? `Pencarian "${searchQuery}" tidak menghasilkan data atlet.`
            : 'Belum ada atlet yang didaftarkan ke kompetisi ini. Daftarkan atlet sekarang.'
        }}
      </p>

      <div class="atl-empty-actions font-mono">
        <AppButton v-if="searchQuery" variant="secondary" size="md" @click="searchQuery = ''">
          Reset Pencarian
        </AppButton>
        <AppButton v-else variant="emerald" size="md" @click="openCreateModal">
          Daftarkan Atlet Pertama
        </AppButton>
      </div>
    </div>

    <!-- Responsive Cards Grid -->
    <div v-else class="atl-grid" role="list">
      <article
        v-for="athlete in filteredAthletes"
        :key="athlete.id"
        class="atl-card"
        :class="{
          'atl-card--male': isMaleGender(athlete.gender),
          'atl-card--female': !isMaleGender(athlete.gender),
        }"
        role="listitem"
      >
        <!-- Corner Accent Stripe -->
        <div
          class="atl-card__accent"
          :class="isMaleGender(athlete.gender) ? 'atl-accent--male' : 'atl-accent--female'"
        />

        <!-- Card Header -->
        <header class="atl-card__header">
          <span
            class="atl-badge-gender font-mono"
            :class="
              isMaleGender(athlete.gender) ? 'atl-badge-gender--male' : 'atl-badge-gender--female'
            "
          >
            <span class="atl-badge-dot" />
            {{ isMaleGender(athlete.gender) ? 'PUTRA (MERAH)' : 'PUTRI (BIRU)' }}
          </span>
          <span class="atl-chip-id font-mono">ID: {{ athlete.id.slice(0, 8) }}</span>
        </header>

        <!-- Card Content -->
        <div class="atl-card__content">
          <h3 class="atl-card__title">{{ athlete.name }}</h3>

          <div class="atl-card__team-box font-mono">
            <span class="atl-team-label">PERGURUAN / KONTINGEN:</span>
            <strong class="atl-team-value">{{ athlete.team || 'Independen' }}</strong>
          </div>
        </div>

        <!-- Card Footer -->
        <footer class="atl-card__footer">
          <div class="atl-card__status-row font-mono">
            <span class="atl-status-dot" />
            <span class="atl-status-text">TERDAFTAR RESMI</span>
          </div>
          <button
            v-if="authStore.isAdminOrDewan"
            type="button"
            class="atl-verify-link font-mono"
            @click="emit('navigateVerification')"
          >
            Verifikasi
            <svg
              width="13"
              height="13"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="M5 12h14" />
              <path d="m12 5 7 7-7 7" />
            </svg>
          </button>
        </footer>
      </article>
    </div>

    <!-- Modal Form (Daftarkan Atlet Baru) -->
    <div
      v-if="isModalOpen"
      class="atl-modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="atl-modal-title"
      @click.self="isModalOpen = false"
    >
      <div class="atl-modal">
        <header class="atl-modal__header">
          <div>
            <div class="atl-modal__tag font-mono">REGISTRASI PESERTA TURNAMEN</div>
            <h3 id="atl-modal-title" class="atl-modal__title">Daftarkan Atlet Baru</h3>
          </div>
          <button class="atl-modal__close" aria-label="Tutup dialog" @click="isModalOpen = false">
            ✕
          </button>
        </header>

        <form class="atl-modal__body" @submit.prevent="handleSubmitAthlete">
          <!-- Name -->
          <div class="atl-form-group">
            <label for="atl-name-input" class="atl-form-label font-mono">
              NAMA LENGKAP ATLET <span class="atl-req">*</span>
            </label>
            <input
              id="atl-name-input"
              v-model="formState.name"
              type="text"
              class="atl-form-input font-mono"
              :class="{ 'atl-form-input--error': formErrors.name }"
              placeholder="Contoh: Danang Sutawijaya"
              required
            />
            <span v-if="formErrors.name" class="atl-form-error">{{ formErrors.name }}</span>
          </div>

          <!-- Gender Toggle -->
          <div class="atl-form-group">
            <label class="atl-form-label font-mono">
              GOLONGAN & JENIS KELAMIN <span class="atl-req">*</span>
            </label>
            <div class="atl-gender-toggle-group">
              <label
                class="atl-gender-radio"
                :class="{
                  'atl-gender-radio--active-male':
                    formState.gender === 'male' || formState.gender === 'putra',
                }"
              >
                <input v-model="formState.gender" type="radio" value="male" class="sr-only" />
                <span class="atl-radio-indicator atl-radio-indicator--male" />
                <div class="atl-radio-text">
                  <strong>DEWASA PUTRA</strong>
                  <span class="atl-sub font-mono">Sudut Merah</span>
                </div>
              </label>

              <label
                class="atl-gender-radio"
                :class="{
                  'atl-gender-radio--active-female':
                    formState.gender === 'female' || formState.gender === 'putri',
                }"
              >
                <input v-model="formState.gender" type="radio" value="female" class="sr-only" />
                <span class="atl-radio-indicator atl-radio-indicator--female" />
                <div class="atl-radio-text">
                  <strong>DEWASA PUTRI</strong>
                  <span class="atl-sub font-mono">Sudut Biru</span>
                </div>
              </label>
            </div>
          </div>

          <!-- Perguruan / Kontingen -->
          <div class="atl-form-group">
            <label for="atl-team-input" class="atl-form-label font-mono">
              PERGURUAN / KONTINGEN TIM <span class="atl-req">*</span>
            </label>
            <input
              id="atl-team-input"
              v-model="formState.team"
              type="text"
              class="atl-form-input font-mono"
              :class="{ 'atl-form-input--error': formErrors.team }"
              placeholder="Contoh: Tapak Suci, PSHT, Merpati Putih, dll."
              list="contingent-datalist"
              required
            />
            <datalist id="contingent-datalist">
              <option v-for="c in contingents" :key="c.id" :value="c.name" />
            </datalist>
            <span v-if="formErrors.team" class="atl-form-error">{{ formErrors.team }}</span>
          </div>

          <!-- Weight Class -->
          <div class="atl-form-group">
            <label for="atl-weight-class-input" class="atl-form-label font-mono">
              KELAS TANDING (IPSI 2023) <span class="atl-req">*</span>
            </label>
            <select
              id="atl-weight-class-input"
              v-model="formState.weight_class"
              class="atl-form-select font-mono"
            >
              <option value="A">Kelas A (45–50 Kg)</option>
              <option value="B">Kelas B (50–55 Kg)</option>
              <option value="C">Kelas C (55–60 Kg)</option>
              <option value="D">Kelas D (60–65 Kg)</option>
              <option value="E">Kelas E (65–70 Kg)</option>
              <option value="F">Kelas F (70–75 Kg)</option>
              <option value="G">Kelas G (75–80 Kg)</option>
              <option value="H">Kelas H (80–85 Kg)</option>
              <option value="I">Kelas I (85–90 Kg)</option>
              <option value="J">Kelas J (90–95 Kg)</option>
              <option value="OPEN">Kelas Bebas</option>
            </select>
          </div>

          <footer class="atl-modal__footer font-mono">
            <AppButton type="button" variant="secondary" size="md" @click="isModalOpen = false">
              Batal
            </AppButton>
            <AppButton
              type="submit"
              variant="emerald"
              size="md"
              :loading="registerMutation.isPending.value"
            >
              Simpan & Daftarkan
            </AppButton>
          </footer>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.atl-ops {
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: 100%;
  color: var(--text-main, #f8fafc);
  box-sizing: border-box;
}

/* Toast */
.atl-toast {
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
.atl-toast--success {
  border-left: 4px solid var(--color-emerald-500, #10b981);
}
.atl-toast--danger {
  border-left: 4px solid var(--color-primary-600, #dc2626);
}
.atl-toast--info {
  border-left: 4px solid var(--color-secondary-600, #2563eb);
}
.atl-toast__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: currentColor;
}
.atl-toast__content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.atl-toast__title {
  font-size: 0.875rem;
  font-weight: 700;
}
.atl-toast__msg {
  font-size: 0.8125rem;
  color: var(--color-neutral-400, #94a3b8);
}
.atl-toast__close {
  background: none;
  border: none;
  color: var(--color-neutral-500, #64748b);
  cursor: pointer;
  font-size: 1rem;
}
.atl-fade-enter-active,
.atl-fade-leave-active {
  transition:
    opacity 0.2s ease,
    transform 0.2s ease;
}
.atl-fade-enter-from,
.atl-fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* Header */
.atl-header {
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
.atl-header__left {
  display: flex;
  align-items: center;
  gap: 16px;
}
.atl-header__icon-box {
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
.atl-header__sub {
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
.atl-bullet {
  color: var(--color-neutral-600, #475569);
}
.atl-tag {
  color: var(--color-emerald-400, #34d399);
}
.atl-pulse-wrapper {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--color-emerald-400, #34d399);
}
.atl-pulse-dot {
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
.atl-header__title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.5rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #ffffff;
  line-height: 1.2;
}
.atl-header__actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

/* Metrics */
.atl-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 12px;
}
.atl-metric-card {
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
.atl-metric-card--male {
  border-left: 3px solid var(--color-primary-600, #dc2626);
}
.atl-metric-card--female {
  border-left: 3px solid var(--color-secondary-600, #2563eb);
}
.atl-metric-corner-tag {
  position: absolute;
  top: 10px;
  right: 12px;
  font-size: 0.625rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--color-neutral-500, #64748b);
}
.atl-metric-label {
  font-family: var(--font-mono, monospace);
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--color-neutral-400, #94a3b8);
}
.atl-metric-value-row {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-top: 2px;
}
.atl-metric-val {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.875rem;
  font-weight: 800;
  color: #ffffff;
  line-height: 1;
}
.atl-val-male {
  color: #fca5a5;
}
.atl-val-female {
  color: #93c5fd;
}
.atl-metric-val-text {
  font-size: 1.875rem;
  font-weight: 800;
  color: var(--color-emerald-400, #34d399);
  line-height: 1;
}
.atl-metric-badge {
  font-size: 0.6875rem;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--color-neutral-400, #94a3b8);
}
.atl-badge-male {
  background: rgba(220, 38, 38, 0.15);
  color: #f87171;
}
.atl-badge-female {
  background: rgba(37, 99, 235, 0.15);
  color: #60a5fa;
}
.atl-metric-desc {
  font-size: 0.75rem;
  color: var(--color-neutral-500, #64748b);
  margin-top: 2px;
}

/* Toolbar */
.atl-toolbar {
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
.atl-toolbar__tabs {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.atl-tab-btn {
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
.atl-tab-btn:hover {
  background: rgba(255, 255, 255, 0.05);
  color: #ffffff;
}
.atl-tab-btn--active {
  background: rgba(255, 255, 255, 0.1);
  color: #ffffff;
  border-color: rgba(255, 255, 255, 0.15);
}
.atl-tab-btn--active-male {
  background: rgba(220, 38, 38, 0.15);
  color: #fca5a5;
  border-color: rgba(220, 38, 38, 0.4);
}
.atl-tab-btn--active-female {
  background: rgba(37, 99, 235, 0.15);
  color: #93c5fd;
  border-color: rgba(37, 99, 235, 0.4);
}
.atl-tab-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}
.atl-dot-male {
  background-color: var(--color-primary-500, #ef4444);
}
.atl-dot-female {
  background-color: var(--color-secondary-500, #3b82f6);
}
.atl-tab-badge {
  font-size: 0.6875rem;
  padding: 1px 6px;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.25);
}

.atl-toolbar__search-group {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  flex: 1;
  max-width: 680px;
  justify-content: flex-end;
}
.atl-search-box {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 200px;
}
.atl-search-icon {
  position: absolute;
  left: 10px;
  color: var(--color-neutral-500, #64748b);
  pointer-events: none;
}
.atl-search-input {
  width: 100%;
  padding: 8px 32px 8px 34px;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  color: #ffffff;
  font-size: 0.75rem;
  outline: none;
}
.atl-search-input:focus {
  border-color: var(--color-emerald-400, #34d399);
}
.atl-search-clear {
  position: absolute;
  right: 10px;
  background: none;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  cursor: pointer;
  font-size: 0.8125rem;
}
.atl-filter-select,
.atl-sort-select {
  padding: 8px 12px;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  color: var(--color-neutral-300, #cbd5e1);
  font-size: 0.75rem;
  outline: none;
  cursor: pointer;
}

/* Card Grid */
.atl-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(310px, 1fr));
  gap: 16px;
}
@media (max-width: 640px) {
  .atl-grid {
    grid-template-columns: 1fr;
  }
}

.atl-card {
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
.atl-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.35);
}
.atl-card__accent {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
}
.atl-accent--male {
  background: linear-gradient(90deg, #dc2626, #ef4444);
}
.atl-accent--female {
  background: linear-gradient(90deg, #2563eb, #3b82f6);
}

.atl-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 18px 10px 18px;
}
.atl-badge-gender {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.6875rem;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 4px;
}
.atl-badge-gender--male {
  background: rgba(220, 38, 38, 0.18);
  color: #fca5a5;
  border: 1px solid rgba(220, 38, 38, 0.35);
}
.atl-badge-gender--female {
  background: rgba(37, 99, 235, 0.18);
  color: #93c5fd;
  border: 1px solid rgba(37, 99, 235, 0.35);
}
.atl-badge-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}
.atl-chip-id {
  font-size: 0.625rem;
  color: var(--color-neutral-500, #64748b);
}

.atl-card__content {
  padding: 0 18px 14px 18px;
}
.atl-card__title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.125rem;
  font-weight: 700;
  color: #ffffff;
  line-height: 1.3;
  margin-bottom: 8px;
}
.atl-card__team-box {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 12px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 6px;
}
.atl-team-label {
  font-size: 0.5625rem;
  color: var(--color-neutral-500, #64748b);
}
.atl-team-value {
  font-size: 0.8125rem;
  color: #ffffff;
  font-weight: 600;
}

.atl-card__footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 18px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(0, 0, 0, 0.15);
}
.atl-card__status-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.atl-status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: var(--color-emerald-400, #34d399);
}
.atl-status-text {
  font-size: 0.625rem;
  color: var(--color-emerald-400, #34d399);
  font-weight: 700;
  letter-spacing: 0.05em;
}
.atl-verify-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: none;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  font-size: 0.6875rem;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.15s ease;
}
.atl-verify-link:hover {
  color: #ffffff;
}

/* Skeleton */
.atl-skeleton-card {
  height: 160px;
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
.skeleton-title {
  width: 75%;
  height: 24px;
}
.skeleton-specs {
  width: 100%;
  height: 40px;
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
.atl-empty-state {
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
.atl-empty-icon {
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
.atl-empty-icon--error {
  background: rgba(220, 38, 38, 0.15);
  color: var(--color-primary-400, #f87171);
}
.atl-empty-title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.25rem;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 8px;
}
.atl-empty-desc {
  max-width: 520px;
  font-size: 0.875rem;
  color: var(--color-neutral-400, #94a3b8);
  line-height: 1.5;
  margin-bottom: 20px;
}
.atl-empty-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* Modal */
.atl-modal-overlay {
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
.atl-modal {
  width: 100%;
  max-width: 540px;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
  overflow: hidden;
}
.atl-modal__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-card, #1e293b);
}
.atl-modal__tag {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--color-emerald-400, #34d399);
  margin-bottom: 4px;
}
.atl-modal__title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.25rem;
  font-weight: 800;
  color: #ffffff;
}
.atl-modal__close {
  background: none;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  cursor: pointer;
  font-size: 1.25rem;
}
.atl-modal__body {
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.atl-form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.atl-form-label {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--color-neutral-300, #cbd5e1);
}
.atl-req {
  color: var(--color-primary-500, #ef4444);
}
.atl-form-input,
.atl-form-select {
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
.atl-form-input:focus,
.atl-form-select:focus {
  border-color: var(--color-emerald-400, #34d399);
}
.atl-form-input--error {
  border-color: var(--color-primary-500, #ef4444) !important;
}
.atl-form-error {
  font-size: 0.6875rem;
  color: var(--color-primary-400, #f87171);
}

.atl-gender-toggle-group {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.atl-gender-radio {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.atl-radio-indicator {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 2px solid var(--color-neutral-600, #475569);
}
.atl-radio-indicator--male {
  border-color: #dc2626;
}
.atl-radio-indicator--female {
  border-color: #2563eb;
}
.atl-radio-text {
  display: flex;
  flex-direction: column;
}
.atl-radio-text strong {
  font-size: 0.75rem;
  color: #ffffff;
}
.atl-radio-text .atl-sub {
  font-size: 0.625rem;
  color: var(--color-neutral-400, #94a3b8);
}
.atl-gender-radio--active-male {
  background: rgba(220, 38, 38, 0.15);
  border-color: #dc2626;
}
.atl-gender-radio--active-male .atl-radio-indicator {
  background: #dc2626;
  box-shadow: 0 0 6px #dc2626;
}
.atl-gender-radio--active-female {
  background: rgba(37, 99, 235, 0.15);
  border-color: #2563eb;
}
.atl-gender-radio--active-female .atl-radio-indicator {
  background: #2563eb;
  box-shadow: 0 0 6px #2563eb;
}

.atl-modal__footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 10px;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
</style>
