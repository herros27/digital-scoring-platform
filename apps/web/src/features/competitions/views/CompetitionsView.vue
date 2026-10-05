<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { AppButton, AppBadge, AppInput } from '@/components/ui'
import { useAuthStore } from '@/stores/auth'
import {
  useCompetitionsQuery,
  useCreateCompetitionMutation,
  useUpdateCompetitionMutation,
  useActivateCompetitionMutation,
  usePauseCompetitionMutation,
  useFinishCompetitionMutation,
  useCancelCompetitionMutation,
} from '../api/competitionQueries'
import {
  createCompetitionSchema,
  updateCompetitionSchema,
  type Competition,
  type CompetitionStatus,
  type CompetitionFilterValues,
} from '../types/competition.schema'

const router = useRouter()
const authStore = useAuthStore()

// Filter State
const filters = ref<CompetitionFilterValues>({
  status: 'all',
  page: 1,
  limit: 20,
})
const searchQuery = ref('')

// TanStack Query for Server State (automatic caching, background refetch, and deduplication)
const { data: competitionsData, isLoading: loading, error: queryError, refetch } = useCompetitionsQuery(filters)

// TanStack Mutations for State Transitions
const createMutation = useCreateCompetitionMutation()
const updateMutation = useUpdateCompetitionMutation()
const activateMutation = useActivateCompetitionMutation()
const pauseMutation = usePauseCompetitionMutation()
const finishMutation = useFinishCompetitionMutation()
const cancelMutation = useCancelCompetitionMutation()

const competitions = computed(() => competitionsData.value?.data || [])
const totalCompetitions = computed(() => competitionsData.value?.pagination?.total || 0)

// Feedback toast
const toast = ref<{ title: string; message: string; type: 'success' | 'danger' | 'info' } | null>(null)
let toastTimer: any = null
function showToast(title: string, message: string, type: 'success' | 'danger' | 'info' = 'success') {
  if (toastTimer) clearTimeout(toastTimer)
  toast.value = { title, message, type }
  toastTimer = setTimeout(() => {
    toast.value = null
  }, 4000)
}

// Modal Form State (Create / Edit)
const isFormModalOpen = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const activeEditId = ref<string | null>(null)
const formErrors = ref<Record<string, string>>({})

const formState = ref({
  name: '',
  location: '',
  start_date: '',
  end_date: '',
  ruleset_name: 'PERSILAT 2024',
})

// Confirmation Dialog State
const confirmModal = ref<{
  open: boolean
  title: string
  message: string
  confirmText: string
  confirmVariant: 'emerald' | 'danger' | 'primary' | 'secondary'
  action: () => Promise<void>
}>({
  open: false,
  title: '',
  message: '',
  confirmText: 'Lanjutkan',
  confirmVariant: 'primary',
  action: async () => {},
})
const confirmLoading = ref(false)

// Detail Drawer State
const detailModal = ref<{
  open: boolean
  competition: Competition | null
}>({
  open: false,
  competition: null,
})

// Stats computation
const stats = computed(() => {
  const all = competitions.value
  return {
    total: totalCompetitions.value || all.length,
    active: all.filter((c) => c.status === 'active').length,
    paused: all.filter((c) => c.status === 'paused').length,
    draft: all.filter((c) => c.status === 'draft').length,
    completed: all.filter((c) => c.status === 'completed').length,
  }
})

// Filtered list
const filteredCompetitions = computed(() => {
  let list = competitions.value
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase().trim()
    list = list.filter(
      (c) =>
        c.name.toLowerCase().includes(q) ||
        (c.location && c.location.toLowerCase().includes(q)) ||
        (c.ruleset_name && c.ruleset_name.toLowerCase().includes(q)) ||
        c.id.toLowerCase().includes(q)
    )
  }
  return list
})

// Open Create Modal
function openCreateModal() {
  formMode.value = 'create'
  activeEditId.value = null
  const today = new Date().toISOString().split('T')[0]
  const nextWeek = new Date(Date.now() + 7 * 86400000).toISOString().split('T')[0]
  formState.value = {
    name: '',
    location: '',
    start_date: today,
    end_date: nextWeek,
    ruleset_name: 'PERSILAT 2024',
  }
  formErrors.value = {}
  isFormModalOpen.value = true
}

// Open Edit Modal
function openEditModal(c: Competition) {
  formMode.value = 'edit'
  activeEditId.value = c.id
  formErrors.value = {}
  formState.value = {
    name: c.name,
    location: c.location || '',
    start_date: c.start_date ? c.start_date.split('T')[0] : '',
    end_date: c.end_date ? c.end_date.split('T')[0] : '',
    ruleset_name: c.ruleset_name || 'PERSILAT 2024',
  }
  isFormModalOpen.value = true
}

// Submit Form with Zod Validation
async function handleFormSubmit() {
  formErrors.value = {}

  if (formMode.value === 'create') {
    const result = createCompetitionSchema.safeParse(formState.value)
    if (!result.success) {
      result.error.issues.forEach((err) => {
        const field = String(err.path[0])
        formErrors.value[field] = err.message
      })
      return
    }

    try {
      const created = await createMutation.mutateAsync(result.data)
      showToast('Kejuaraan Dibuat', `Kejuaraan "${created.name}" berhasil dibuat sebagai Draft.`, 'success')
      isFormModalOpen.value = false
    } catch (err: any) {
      showToast('Gagal Menyimpan', err.message || 'Terjadi kesalahan sistem saat membuat kejuaraan', 'danger')
    }
  } else if (formMode.value === 'edit' && activeEditId.value) {
    const result = updateCompetitionSchema.safeParse(formState.value)
    if (!result.success) {
      result.error.issues.forEach((err) => {
        const field = String(err.path[0])
        formErrors.value[field] = err.message
      })
      return
    }

    try {
      const updated = await updateMutation.mutateAsync({
        id: activeEditId.value,
        payload: result.data,
      })
      showToast('Perubahan Disimpan', `Data kejuaraan "${updated.name}" berhasil diperbarui.`, 'success')
      isFormModalOpen.value = false
    } catch (err: any) {
      showToast('Gagal Menyimpan', err.message || 'Terjadi kesalahan sistem saat memperbarui kejuaraan', 'danger')
    }
  }
}

// Detail View
function viewDetail(c: Competition) {
  detailModal.value = {
    open: true,
    competition: c,
  }
}

// Activate Action
function confirmActivate(c: Competition) {
  confirmModal.value = {
    open: true,
    title: 'Aktifkan Kejuaraan',
    message: `Apakah Anda yakin ingin mengaktifkan kejuaraan "${c.name}"? Seluruh gelanggang dan jadwal pertandingan akan dapat diproses.`,
    confirmText: 'Aktifkan Sekarang',
    confirmVariant: 'emerald',
    action: async () => {
      await activateMutation.mutateAsync(c.id)
      showToast('Status Diperbarui', `Kejuaraan "${c.name}" kini berstatus Aktif.`, 'success')
    },
  }
}

// Pause Action
function confirmPause(c: Competition) {
  confirmModal.value = {
    open: true,
    title: 'Jeda Sementara Kejuaraan',
    message: `Apakah Anda ingin menjeda kejuaraan "${c.name}"? Pertandingan yang sedang berlangsung dapat ditunda.`,
    confirmText: 'Jeda Kejuaraan',
    confirmVariant: 'secondary',
    action: async () => {
      await pauseMutation.mutateAsync(c.id)
      showToast('Kejuaraan Dijeda', `Kejuaraan "${c.name}" telah dijeda sementara.`, 'info')
    },
  }
}

// Finish Action
function confirmFinish(c: Competition) {
  confirmModal.value = {
    open: true,
    title: 'Selesaikan Kejuaraan',
    message: `Peringatan: Menyelesaikan kejuaraan "${c.name}" akan mengunci seluruh bagan dan perolehan medali akhir. Tindakan ini tidak dapat dibatalkan.`,
    confirmText: 'Selesaikan Pertandingan',
    confirmVariant: 'primary',
    action: async () => {
      await finishMutation.mutateAsync(c.id)
      showToast('Kejuaraan Selesai', `Kejuaraan "${c.name}" telah resmi ditutup dan selesai.`, 'success')
    },
  }
}

// Cancel Action
function confirmCancel(c: Competition) {
  confirmModal.value = {
    open: true,
    title: 'Batalkan Kejuaraan',
    message: `Peringatan: Seluruh aktivitas kejuaraan "${c.name}" akan dibatalkan secara permanen. Lanjutkan?`,
    confirmText: 'Batalkan Kejuaraan',
    confirmVariant: 'danger',
    action: async () => {
      await cancelMutation.mutateAsync(c.id)
      showToast('Kejuaraan Dibatalkan', `Kejuaraan "${c.name}" telah dibatalkan.`, 'danger')
    },
  }
}

// Execute confirmation action
async function executeConfirmAction() {
  confirmLoading.value = true
  try {
    await confirmModal.value.action()
    confirmModal.value.open = false
  } catch (err: any) {
    showToast('Operasi Gagal', err.message || 'Gagal mengubah status kompetisi', 'danger')
  } finally {
    confirmLoading.value = false
  }
}

// Date formatter
function formatDate(dateStr?: string): string {
  if (!dateStr) return '-'
  try {
    const d = new Date(dateStr)
    return d.toLocaleDateString('id-ID', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
    })
  } catch {
    return dateStr
  }
}

// Badge helpers
function getStatusBadgeVariant(status: CompetitionStatus): 'emerald' | 'warning' | 'neutral' | 'primary' | 'red' {
  switch (status) {
    case 'active':
      return 'emerald'
    case 'paused':
      return 'warning'
    case 'completed':
      return 'primary'
    case 'cancelled':
      return 'red'
    case 'draft':
    default:
      return 'neutral'
  }
}

function getStatusLabel(status: CompetitionStatus): string {
  switch (status) {
    case 'active':
      return 'Aktif'
    case 'paused':
      return 'Dijeda'
    case 'completed':
      return 'Selesai'
    case 'cancelled':
      return 'Dibatalkan'
    case 'draft':
    default:
      return 'Draft'
  }
}

function openBracketOps(c: Competition) {
  authStore.setSession({
    ...authStore.session!,
    token: authStore.token,
    role: authStore.role,
    competitionId: c.id,
  })
  router.push(`/ops?comp_id=${c.id}`)
}
</script>

<template>
  <div class="comp-mgmt">
    <!-- Feedback Toast -->
    <Transition name="fade">
      <div v-if="toast" class="comp-toast" :class="`comp-toast--${toast.type}`" role="status" aria-live="polite">
        <div class="comp-toast__content">
          <span class="comp-toast__title">{{ toast.title }}</span>
          <span class="comp-toast__msg">{{ toast.message }}</span>
        </div>
        <button class="comp-toast__close" aria-label="Tutup notifikasi" @click="toast = null">✕</button>
      </div>
    </Transition>

    <!-- Header Section (Standardized Ops Banner) -->
    <header class="comp-header">
      <div class="comp-header__left">
        <div class="comp-header__icon-box">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
            <path d="M6 9H4.5a2.5 2.5 0 0 1 0-5H6" />
            <path d="M18 9h1.5a2.5 2.5 0 0 0 0-5H18" />
            <path d="M4 22h16" />
            <path d="M10 14.66V17c0 .55-.47.98-.97 1.21C7.85 18.75 7 20.24 7 22" />
            <path d="M14 14.66V17c0 .55.47.98.97 1.21C16.15 18.75 17 20.24 17 22" />
            <path d="M18 2H6v7a6 6 0 0 0 12 0V2Z" />
          </svg>
        </div>
        <div>
          <div class="comp-header__sub font-mono">
            <span>DIREKTORAT PERTANDINGAN</span>
            <span class="comp-bullet">•</span>
            <span class="comp-tag">REGULASI PERSILAT 2024</span>
            <span class="comp-bullet">•</span>
            <span class="comp-pulse-wrapper">
              <span class="comp-pulse-dot" />
              <span>SERVER CACHE SYNC</span>
            </span>
          </div>
          <h2 class="comp-header__title">MANAJEMEN KEJUARAAN</h2>
        </div>
      </div>

      <div class="comp-header__actions font-mono">
        <AppButton variant="secondary" size="sm" :loading="loading" @click="() => refetch()">
          <template #icon>
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 2v6h-6" />
              <path d="M3 12a9 9 0 0 1 15-6.7L21 8" />
              <path d="M3 22v-6h6" />
              <path d="M21 12a9 9 0 0 1-15 6.7L3 16" />
            </svg>
          </template>
          Segarkan
        </AppButton>
        <AppButton variant="emerald" size="sm" @click="openCreateModal">
          <template #icon>
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <line x1="12" y1="5" x2="12" y2="19" />
              <line x1="5" y1="12" x2="19" y2="12" />
            </svg>
          </template>
          Tambah Kejuaraan
        </AppButton>
      </div>
    </header>

    <!-- Metrics Strip -->
    <section class="comp-metrics" aria-label="Ringkasan Statistik Kejuaraan">
      <div class="metric-card">
        <span class="metric-card__label">TOTAL KEJUARAAN</span>
        <span class="metric-card__value tabular-nums">{{ stats.total }}</span>
        <span class="metric-card__sub">Terdaftar di sistem</span>
      </div>
      <div class="metric-card metric-card--active">
        <span class="metric-card__label">KEJUARAAN AKTIF</span>
        <span class="metric-card__value tabular-nums text-emerald">{{ stats.active }}</span>
        <span class="metric-card__sub">Gelanggang live</span>
      </div>
      <div class="metric-card">
        <span class="metric-card__label">STATUS DRAFT</span>
        <span class="metric-card__value tabular-nums">{{ stats.draft }}</span>
        <span class="metric-card__sub">Dalam persiapan</span>
      </div>
      <div class="metric-card">
        <span class="metric-card__label">DIJEDA SEMENTARA</span>
        <span class="metric-card__value tabular-nums text-amber">{{ stats.paused }}</span>
        <span class="metric-card__sub">Break / Istirahat</span>
      </div>
      <div class="metric-card">
        <span class="metric-card__label">TELAH SELESAI</span>
        <span class="metric-card__value tabular-nums text-muted">{{ stats.completed }}</span>
        <span class="metric-card__sub">Arsip & Rekap Medali</span>
      </div>
    </section>

    <!-- Controls Toolbar (Search & Filter Tabs) -->
    <section class="comp-toolbar">
      <div class="comp-toolbar__tabs" role="tablist">
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': filters.status === 'all' }"
          @click="filters.status = 'all'"
        >
          Semua ({{ stats.total }})
        </button>
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': filters.status === 'active' }"
          @click="filters.status = 'active'"
        >
          <span class="dot-indicator dot-emerald" />
          Aktif ({{ stats.active }})
        </button>
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': filters.status === 'draft' }"
          @click="filters.status = 'draft'"
        >
          <span class="dot-indicator dot-neutral" />
          Draft ({{ stats.draft }})
        </button>
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': filters.status === 'paused' }"
          @click="filters.status = 'paused'"
        >
          <span class="dot-indicator dot-amber" />
          Dijeda ({{ stats.paused }})
        </button>
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': filters.status === 'completed' }"
          @click="filters.status = 'completed'"
        >
          Selesai ({{ stats.completed }})
        </button>
      </div>

      <div class="comp-toolbar__search">
        <AppInput
          v-model="searchQuery"
          placeholder="Cari kejuaraan, venue, atau ID..."
          icon
        />
      </div>
    </section>

    <!-- Content Area: Skeletons, Error State, Empty State, or Card Grid -->
    <main class="comp-main">
      <!-- Loading Skeletons -->
      <div v-if="loading && competitions.length === 0" class="comp-grid">
        <div v-for="i in 4" :key="i" class="comp-card-skeleton">
          <div class="skeleton-line skeleton-title" />
          <div class="skeleton-line skeleton-meta" />
          <div class="skeleton-line skeleton-date" />
          <div class="skeleton-actions" />
        </div>
      </div>

      <!-- Error State -->
      <div v-else-if="queryError && competitions.length === 0" class="comp-empty">
        <div class="comp-empty__icon comp-empty__icon--error">✕</div>
        <h3 class="comp-empty__title">Gagal Menghubungkan Backend</h3>
        <p class="comp-empty__desc">{{ (queryError as any)?.message || 'Terjadi gangguan jaringan' }}</p>
        <AppButton variant="secondary" size="md" @click="() => refetch()">Coba Lagi</AppButton>
      </div>

      <!-- Empty State -->
      <div v-else-if="filteredCompetitions.length === 0" class="comp-empty">
        <div class="comp-empty__icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M6 9H4.5a2.5 2.5 0 0 1 0-5H6" />
            <path d="M18 9h1.5a2.5 2.5 0 0 0 0-5H18" />
            <path d="M4 22h16" />
            <path d="M10 14.66V17c0 .55-.47.98-.97 1.21C7.85 18.75 7 20.24 7 22" />
            <path d="M14 14.66V17c0 .55.47.98.97 1.21C16.15 18.75 17 20.24 17 22" />
            <path d="M18 2H6v7a6 6 0 0 0 12 0V2Z" />
          </svg>
        </div>
        <h3 class="comp-empty__title">Tidak Ada Kejuaraan</h3>
        <p class="comp-empty__desc">
          {{ searchQuery ? 'Tidak ada kejuaraan yang sesuai dengan pencarian Anda.' : 'Belum ada kejuaraan dengan status ini. Mulai dengan membuat kejuaraan baru.' }}
        </p>
        <AppButton v-if="!searchQuery" variant="primary" size="md" @click="openCreateModal">
          Buat Kejuaraan Pertama
        </AppButton>
      </div>

      <!-- Competition Cards Grid -->
      <div v-else class="comp-grid">
        <article
          v-for="c in filteredCompetitions"
          :key="c.id"
          class="comp-card"
          :class="`comp-card--${c.status}`"
        >
          <!-- Card Header -->
          <div class="comp-card__head">
            <div class="comp-card__badges">
              <AppBadge
                :variant="getStatusBadgeVariant(c.status)"
                size="sm"
                :dot="c.status === 'active' || c.status === 'paused' || c.status === 'draft'"
                :pulse="c.status === 'active'"
              >
                {{ getStatusLabel(c.status) }}
              </AppBadge>
              <AppBadge variant="neutral" size="sm" outline>
                {{ c.ruleset_name || 'PERSILAT 2024' }}
              </AppBadge>
            </div>
            <span class="comp-card__id font-mono tabular-nums" title="ID Kejuaraan">#{{ c.id.slice(0, 8) }}</span>
          </div>

          <!-- Card Body -->
          <div class="comp-card__body">
            <h3 class="comp-card__title" @click="viewDetail(c)">{{ c.name }}</h3>

            <div class="comp-card__meta">
              <div class="meta-row">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="meta-icon">
                  <path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z" />
                  <circle cx="12" cy="10" r="3" />
                </svg>
                <span class="meta-text">{{ c.location || 'Lokasi belum ditentukan' }}</span>
              </div>
              <div class="meta-row">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="meta-icon">
                  <rect width="18" height="18" x="3" y="4" rx="2" ry="2" />
                  <line x1="16" y1="2" x2="16" y2="6" />
                  <line x1="8" y1="2" x2="8" y2="6" />
                  <line x1="3" y1="10" x2="21" y2="10" />
                </svg>
                <span class="meta-text tabular-nums">{{ formatDate(c.start_date) }} - {{ formatDate(c.end_date) }}</span>
              </div>
            </div>
          </div>

          <!-- Card Actions (Endpoints: Activate, Pause, Finish, Cancel, Update, Detail) -->
          <div class="comp-card__actions">
            <div class="primary-actions">
              <AppButton variant="secondary" size="sm" @click="viewDetail(c)">
                Detail
              </AppButton>
              <AppButton variant="outlined" size="sm" @click="openEditModal(c)">
                Edit
              </AppButton>
              <AppButton variant="primary" size="sm" @click="openBracketOps(c)">
                Bagan & Arena
              </AppButton>
            </div>

            <!-- Lifecycle Controls -->
            <div class="lifecycle-actions">
              <!-- Activate Button -->
              <button
                v-if="c.status === 'draft' || c.status === 'paused'"
                class="lc-btn lc-btn--activate"
                title="Aktifkan Kejuaraan (PUT /activate)"
                :disabled="activateMutation.isPending.value"
                @click="confirmActivate(c)"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <polygon points="5 3 19 12 5 21 5 3" />
                </svg>
                Aktifkan
              </button>

              <!-- Pause Button -->
              <button
                v-if="c.status === 'active'"
                class="lc-btn lc-btn--pause"
                title="Jeda Kejuaraan (POST /pause)"
                :disabled="pauseMutation.isPending.value"
                @click="confirmPause(c)"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <rect x="6" y="4" width="4" height="16" />
                  <rect x="14" y="4" width="4" height="16" />
                </svg>
                Jeda
              </button>

              <!-- Finish Button -->
              <button
                v-if="c.status === 'active' || c.status === 'paused'"
                class="lc-btn lc-btn--finish"
                title="Selesaikan Kejuaraan (POST /finish)"
                :disabled="finishMutation.isPending.value"
                @click="confirmFinish(c)"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                  <polyline points="22 4 12 14.01 9 11.01" />
                </svg>
                Selesai
              </button>

              <!-- Cancel Button -->
              <button
                v-if="c.status !== 'completed' && c.status !== 'cancelled'"
                class="lc-btn lc-btn--cancel"
                title="Batalkan Kejuaraan (POST /cancel)"
                :disabled="cancelMutation.isPending.value"
                @click="confirmCancel(c)"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <circle cx="12" cy="12" r="10" />
                  <line x1="15" y1="9" x2="9" y2="15" />
                  <line x1="9" y1="9" x2="15" y2="15" />
                </svg>
                Batalkan
              </button>
            </div>
          </div>
        </article>
      </div>
    </main>

    <!-- Create / Edit Competition Modal -->
    <div v-if="isFormModalOpen" class="comp-modal-overlay" @click.self="isFormModalOpen = false">
      <div class="comp-modal" role="dialog" aria-modal="true">
        <header class="comp-modal__header">
          <div class="comp-modal__head-text">
            <h2 class="comp-modal__title">
              {{ formMode === 'create' ? 'Buat Kejuaraan Baru' : 'Edit Data Kejuaraan' }}
            </h2>
            <p class="comp-modal__subtitle text-pretty">
              {{ formMode === 'create' ? 'Inisialisasi turnamen silat baru ke dalam sistem scoring.' : 'Perbarui informasi venue, jadwal, atau regulasi kejuaraan.' }}
            </p>
          </div>
          <button class="comp-modal__close" aria-label="Tutup modal" @click="isFormModalOpen = false">✕</button>
        </header>

        <form class="comp-modal__form" @submit.prevent="handleFormSubmit">
          <!-- Field: Nama Kejuaraan -->
          <div class="form-field">
            <label class="form-label" for="comp-name">
              Nama Kejuaraan <span class="required-star">*</span>
            </label>
            <AppInput
              id="comp-name"
              v-model="formState.name"
              placeholder="Contoh: KEJURNAS PENCAK SILAT DEWASA PIALA PRESIDEN 2026"
              :error="formErrors.name"
            />
            <span v-if="formErrors.name" class="field-error-msg">{{ formErrors.name }}</span>
          </div>

          <!-- Field: Lokasi / Venue -->
          <div class="form-field">
            <label class="form-label" for="comp-location">
              Lokasi / Venue Gelanggang <span class="required-star">*</span>
            </label>
            <AppInput
              id="comp-location"
              v-model="formState.location"
              placeholder="Contoh: Padepokan Pencak Silat TMII, Jakarta Timur"
              :error="formErrors.location"
            />
            <span v-if="formErrors.location" class="field-error-msg">{{ formErrors.location }}</span>
          </div>

          <!-- Fields: Tanggal Mulai & Tanggal Selesai -->
          <div class="form-row">
            <div class="form-field">
              <label class="form-label" for="comp-start">
                Tanggal Mulai <span class="required-star">*</span>
              </label>
              <input
                id="comp-start"
                v-model="formState.start_date"
                type="date"
                class="native-date-input"
                :class="{ 'input--error': formErrors.start_date }"
              />
              <span v-if="formErrors.start_date" class="field-error-msg">{{ formErrors.start_date }}</span>
            </div>

            <div class="form-field">
              <label class="form-label" for="comp-end">
                Tanggal Selesai <span class="required-star">*</span>
              </label>
              <input
                id="comp-end"
                v-model="formState.end_date"
                type="date"
                class="native-date-input"
                :class="{ 'input--error': formErrors.end_date }"
              />
              <span v-if="formErrors.end_date" class="field-error-msg">{{ formErrors.end_date }}</span>
            </div>
          </div>

          <!-- Field: Ruleset Pertandingan -->
          <div class="form-field">
            <label class="form-label">
              Regulasi & Ruleset Scoring <span class="required-star">*</span>
            </label>
            <div class="ruleset-options">
              <label class="ruleset-option" :class="{ 'ruleset-option--active': formState.ruleset_name === 'PERSILAT 2024' }">
                <input
                  v-model="formState.ruleset_name"
                  type="radio"
                  value="PERSILAT 2024"
                  name="ruleset"
                />
                <div class="ruleset-text">
                  <strong class="ruleset-title">PERSILAT Rules 2024</strong>
                  <span class="ruleset-desc">Regulasi Internasional: 3 babak @2 menit, sliding window consensus 2/3 juri, teknik jatuhan sah.</span>
                </div>
              </label>

              <label class="ruleset-option" :class="{ 'ruleset-option--active': formState.ruleset_name === 'IPSI 2024' }">
                <input
                  v-model="formState.ruleset_name"
                  type="radio"
                  value="IPSI 2024"
                  name="ruleset"
                />
                <div class="ruleset-text">
                  <strong class="ruleset-title">IPSI Nasional 2024</strong>
                  <span class="ruleset-desc">Regulasi Kejuaraan Nasional & Daerah: Verifikasi timbang badan otomatis, pembagian Pool 16/32.</span>
                </div>
              </label>
            </div>
          </div>

          <!-- Modal Actions -->
          <footer class="comp-modal__footer">
            <AppButton variant="secondary" size="md" type="button" @click="isFormModalOpen = false">
              Batal
            </AppButton>
            <AppButton
              variant="primary"
              size="md"
              type="submit"
              :loading="createMutation.isPending.value || updateMutation.isPending.value"
            >
              {{ formMode === 'create' ? 'Buat Kejuaraan' : 'Simpan Perubahan' }}
            </AppButton>
          </footer>
        </form>
      </div>
    </div>

    <!-- Confirmation Dialog (AlertDialog standard from baseline-ui) -->
    <div v-if="confirmModal.open" class="comp-modal-overlay" @click.self="confirmModal.open = false">
      <div class="comp-modal comp-modal--confirm" role="alertdialog" aria-modal="true">
        <header class="comp-modal__header">
          <h2 class="comp-modal__title">{{ confirmModal.title }}</h2>
          <button class="comp-modal__close" aria-label="Tutup" @click="confirmModal.open = false">✕</button>
        </header>
        <div class="comp-modal__body">
          <p class="confirm-message text-pretty">{{ confirmModal.message }}</p>
        </div>
        <footer class="comp-modal__footer">
          <AppButton variant="secondary" size="md" @click="confirmModal.open = false">
            Batal
          </AppButton>
          <AppButton
            :variant="confirmModal.confirmVariant"
            size="md"
            :loading="confirmLoading"
            @click="executeConfirmAction"
          >
            {{ confirmModal.confirmText }}
          </AppButton>
        </footer>
      </div>
    </div>

    <!-- Competition Detail Drawer / Modal -->
    <div v-if="detailModal.open && detailModal.competition" class="comp-modal-overlay" @click.self="detailModal.open = false">
      <div class="comp-modal comp-modal--detail" role="dialog" aria-modal="true">
        <header class="comp-modal__header">
          <div>
            <div class="detail-badge-row">
              <AppBadge
                :variant="getStatusBadgeVariant(detailModal.competition.status)"
                size="sm"
                :dot="detailModal.competition.status === 'active'"
              >
                {{ getStatusLabel(detailModal.competition.status) }}
              </AppBadge>
              <span class="font-mono tabular-nums text-muted">ID: {{ detailModal.competition.id }}</span>
            </div>
            <h2 class="comp-modal__title">{{ detailModal.competition.name }}</h2>
          </div>
          <button class="comp-modal__close" aria-label="Tutup detail" @click="detailModal.open = false">✕</button>
        </header>

        <div class="comp-modal__body detail-specs">
          <div class="spec-group">
            <span class="spec-label">LOKASI & VENUE</span>
            <span class="spec-value">{{ detailModal.competition.location || 'Belum diatur' }}</span>
          </div>

          <div class="spec-group">
            <span class="spec-label">PERIODE PERTANDINGAN</span>
            <span class="spec-value tabular-nums">
              {{ formatDate(detailModal.competition.start_date) }} s/d {{ formatDate(detailModal.competition.end_date) }}
            </span>
          </div>

          <div class="spec-group">
            <span class="spec-label">REGULASI & RULESET</span>
            <span class="spec-value font-mono">{{ detailModal.competition.ruleset_name || 'PERSILAT 2024' }}</span>
          </div>

          <div class="spec-group">
            <span class="spec-label">TANGGAL DIBUAT SISTEM</span>
            <span class="spec-value tabular-nums">{{ formatDate(detailModal.competition.created_at) }}</span>
          </div>
        </div>

        <footer class="comp-modal__footer">
          <AppButton variant="secondary" size="md" @click="detailModal.open = false">
            Tutup
          </AppButton>
          <AppButton variant="primary" size="md" @click="openBracketOps(detailModal.competition); detailModal.open = false">
            Buka Bagan Pertandingan
          </AppButton>
        </footer>
      </div>
    </div>
  </div>
</template>

<style scoped>
.comp-mgmt {
  display: flex;
  flex-direction: column;
  gap: 24px;
  width: 100%;
  max-width: 1440px;
  margin: 0 auto;
  padding: 24px;
  color: var(--text-primary);
  box-sizing: border-box;
}

/* Toast */
.comp-toast {
  position: fixed;
  top: 24px;
  right: 24px;
  z-index: 1000;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px 20px;
  border-radius: var(--radius-md);
  background-color: var(--bg-card);
  border: 1px solid var(--border-card);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
}
.comp-toast--success {
  border-left: 4px solid var(--color-emerald-500);
}
.comp-toast--danger {
  border-left: 4px solid var(--color-red-500);
}
.comp-toast--info {
  border-left: 4px solid var(--color-blue-500);
}
.comp-toast__content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.comp-toast__title {
  font-weight: 700;
  font-size: 0.875rem;
}
.comp-toast__msg {
  font-size: 0.8125rem;
  color: var(--text-secondary);
}
.comp-toast__close {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 1rem;
}

/* Header */
.comp-header {
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
.comp-header__left {
  display: flex;
  align-items: center;
  gap: 16px;
}
.comp-header__icon-box {
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
.comp-header__sub {
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
.comp-bullet { color: var(--color-neutral-600, #475569); }
.comp-tag { color: var(--color-primary-400, #f87171); }
.comp-pulse-wrapper { display: inline-flex; align-items: center; gap: 5px; color: var(--color-emerald-400, #34d399); }
.comp-pulse-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-emerald-400, #34d399);
  box-shadow: 0 0 8px var(--color-emerald-400, #34d399);
  animation: pulse-dot 1.8s infinite;
}
@keyframes pulse-dot {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(0.85); }
}
.comp-header__title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.5rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #ffffff;
  line-height: 1.2;
}
.comp-header__actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.btn-icon {
  margin-right: 6px;
}

/* Metrics Strip */
.comp-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 14px;
}
.metric-card {
  display: flex;
  flex-direction: column;
  padding: 16px 20px;
  background-color: var(--bg-card);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-lg);
  transition: transform var(--transition-fast), border-color var(--transition-fast);
}
.metric-card:hover {
  border-color: var(--border-active);
}
.metric-card--active {
  background: linear-gradient(180deg, rgba(16, 185, 129, 0.08) 0%, rgba(16, 185, 129, 0.01) 100%);
  border-color: rgba(16, 185, 129, 0.3);
}
.metric-card__label {
  font-family: var(--font-mono);
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--text-muted);
  letter-spacing: 0.05em;
  margin-bottom: 6px;
}
.metric-card__value {
  font-family: var(--font-mono);
  font-size: 1.875rem;
  font-weight: 800;
  line-height: 1.1;
  margin-bottom: 4px;
}
.metric-card__sub {
  font-size: 0.75rem;
  color: var(--text-secondary);
}
.text-emerald { color: var(--color-emerald-400); }
.text-amber { color: var(--color-amber-400); }
.text-muted { color: var(--text-muted); }

/* Controls Toolbar */
.comp-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
  background-color: var(--bg-card);
  padding: 12px 16px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-card);
}
.comp-toolbar__tabs {
  display: flex;
  align-items: center;
  gap: 6px;
  overflow-x: auto;
}
.tab-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border-radius: var(--radius-full);
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-secondary);
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}
.tab-btn:hover {
  background-color: var(--bg-elevated);
  color: var(--text-primary);
}
.tab-btn--active {
  background-color: var(--bg-elevated);
  border-color: var(--border-active);
  color: var(--text-primary);
}
.dot-indicator {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}
.dot-emerald { background-color: var(--color-emerald-400); }
.dot-neutral { background-color: var(--color-neutral-400); }
.dot-amber { background-color: var(--color-amber-400); }
.comp-toolbar__search {
  min-width: 280px;
}

/* Grid & Cards */
.comp-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 20px;
}
.comp-card {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  background-color: var(--bg-card);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-lg);
  padding: 20px;
  transition: transform var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast);
}
.comp-card:hover {
  border-color: var(--border-active);
  transform: translateY(-2px);
  box-shadow: 0 12px 30px rgba(0, 0, 0, 0.25);
}
.comp-card--active {
  border-top: 3px solid var(--color-emerald-500);
}
.comp-card--paused {
  border-top: 3px solid var(--color-amber-500);
}
.comp-card--draft {
  border-top: 3px solid var(--color-neutral-500);
}
.comp-card--completed {
  border-top: 3px solid var(--color-primary-500);
}
.comp-card--cancelled {
  border-top: 3px solid var(--color-red-500);
  opacity: 0.75;
}

.comp-card__head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}
.comp-card__badges {
  display: flex;
  align-items: center;
  gap: 8px;
}
.comp-card__id {
  font-size: 0.75rem;
  color: var(--text-muted);
}
.comp-card__body {
  margin-bottom: 20px;
}
.comp-card__title {
  font-size: 1.125rem;
  font-weight: 700;
  line-height: 1.35;
  margin: 0 0 12px 0;
  cursor: pointer;
  transition: color var(--transition-fast);
}
.comp-card__title:hover {
  color: var(--color-primary-400);
}
.comp-card__meta {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.meta-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  color: var(--text-secondary);
}
.meta-icon {
  color: var(--text-muted);
  flex-shrink: 0;
}
.meta-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.comp-card__actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-top: 14px;
  border-top: 1px solid var(--border-card);
}
.primary-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.lifecycle-actions {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

/* Lifecycle Buttons */
.lc-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  background: var(--bg-elevated);
  border: 1px solid var(--border-card);
  transition: all var(--transition-fast);
}
.lc-btn--activate {
  color: var(--color-emerald-400);
  border-color: rgba(16, 185, 129, 0.3);
}
.lc-btn--activate:hover:not(:disabled) {
  background-color: var(--color-emerald-500);
  color: #fff;
}
.lc-btn--pause {
  color: var(--color-amber-400);
  border-color: rgba(245, 158, 11, 0.3);
}
.lc-btn--pause:hover:not(:disabled) {
  background-color: var(--color-amber-500);
  color: #fff;
}
.lc-btn--finish {
  color: var(--color-primary-400);
  border-color: rgba(59, 130, 246, 0.3);
}
.lc-btn--finish:hover:not(:disabled) {
  background-color: var(--color-primary-500);
  color: #fff;
}
.lc-btn--cancel {
  color: var(--color-red-400);
  border-color: rgba(239, 68, 68, 0.3);
}
.lc-btn--cancel:hover:not(:disabled) {
  background-color: var(--color-red-500);
  color: #fff;
}
.lc-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Skeleton Loading */
.comp-card-skeleton {
  background-color: var(--bg-card);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-lg);
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.skeleton-line {
  background: linear-gradient(90deg, rgba(255,255,255,0.04) 25%, rgba(255,255,255,0.08) 50%, rgba(255,255,255,0.04) 75%);
  background-size: 200% 100%;
  animation: skeleton-shimmer 1.5s infinite;
  border-radius: 4px;
}
.skeleton-title { height: 24px; width: 80%; }
.skeleton-meta { height: 16px; width: 60%; }
.skeleton-date { height: 16px; width: 50%; }
.skeleton-actions { height: 32px; width: 100%; background-color: rgba(255,255,255,0.03); border-radius: 4px; }

@keyframes skeleton-shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* Empty State */
.comp-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 24px;
  background-color: var(--bg-card);
  border: 1px dashed var(--border-card);
  border-radius: var(--radius-lg);
  text-align: center;
  gap: 12px;
}
.comp-empty__icon {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background-color: var(--bg-elevated);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
}
.comp-empty__icon--error {
  color: var(--color-red-400);
  background-color: rgba(239, 68, 68, 0.1);
}
.comp-empty__title {
  font-size: 1.125rem;
  font-weight: 700;
  margin: 0;
}
.comp-empty__desc {
  font-size: 0.875rem;
  color: var(--text-secondary);
  max-width: 420px;
  margin: 0 0 12px 0;
}

/* Modal Styling */
.comp-modal-overlay {
  position: fixed;
  inset: 0;
  background-color: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
  z-index: 999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}
.comp-modal {
  background-color: var(--bg-card);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-lg);
  width: 100%;
  max-width: 560px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
}
.comp-modal--confirm {
  max-width: 440px;
}
.comp-modal--detail {
  max-width: 500px;
}
.comp-modal__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-card);
}
.comp-modal__title {
  font-size: 1.25rem;
  font-weight: 700;
  margin: 0;
}
.comp-modal__subtitle {
  font-size: 0.8125rem;
  color: var(--text-secondary);
  margin: 4px 0 0 0;
}
.comp-modal__close {
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: 1.25rem;
  cursor: pointer;
  padding: 4px;
}
.comp-modal__close:hover {
  color: var(--text-primary);
}
.comp-modal__form,
.comp-modal__body {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.form-label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-secondary);
}
.required-star {
  color: var(--color-red-400);
}
.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}
.native-date-input {
  width: 100%;
  padding: 10px 14px;
  background-color: var(--bg-elevated);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-family: var(--font-mono);
  font-size: 0.875rem;
  outline: none;
  box-sizing: border-box;
}
.native-date-input:focus {
  border-color: var(--border-active);
}
.input--error {
  border-color: var(--color-red-500) !important;
}
.field-error-msg {
  font-size: 0.75rem;
  color: var(--color-red-400);
}

/* Ruleset selector */
.ruleset-options {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.ruleset-option {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 14px;
  border-radius: var(--radius-md);
  background-color: var(--bg-elevated);
  border: 1px solid var(--border-card);
  cursor: pointer;
  transition: all var(--transition-fast);
}
.ruleset-option input {
  margin-top: 3px;
}
.ruleset-option--active {
  border-color: var(--border-active);
  background-color: rgba(59, 130, 246, 0.05);
}
.ruleset-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ruleset-title {
  font-size: 0.875rem;
  color: var(--text-primary);
}
.ruleset-desc {
  font-size: 0.75rem;
  color: var(--text-secondary);
  line-height: 1.4;
}

.comp-modal__footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  border-top: 1px solid var(--border-card);
  background-color: rgba(0, 0, 0, 0.15);
}

/* Detail modal specs */
.detail-badge-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
}
.detail-specs {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.spec-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--border-card);
}
.spec-group:last-child {
  border-bottom: none;
}
.spec-label {
  font-family: var(--font-mono);
  font-size: 0.6875rem;
  color: var(--text-muted);
  letter-spacing: 0.05em;
}
.spec-value {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--text-primary);
}
.confirm-message {
  font-size: 0.9375rem;
  color: var(--text-secondary);
  line-height: 1.5;
  margin: 0;
}

/* Transitions */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
