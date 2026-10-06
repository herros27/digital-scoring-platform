<script setup lang="ts">
import { getErrorMessage } from '@/api/error'
import { ref, computed } from 'vue'
import { AppButton } from '@/components/ui'
import {
  useArenasQuery,
  useCreateArenaMutation,
  useUpdateArenaMutation,
  useDeleteArenaMutation,
  useSeedArenasMutation,
} from '../api/arenaQueries'
import {
  useCompetitionQuery,
  useCompetitionsQuery,
} from '@/features/competitions/api/competitionQueries'
import type { CompetitionFilterValues } from '@/features/competitions/types/competition.schema'
import {
  createArenaSchema,
  type Arena,
} from '../types/arena.schema'

const props = withDefaults(
  defineProps<{
    competitionId?: string
  }>(),
  {
    competitionId: 'b26d7145-d341-4795-ac58-bff86d13cbf8',
  }
)

const emit = defineEmits<{
  (e: 'selectArena', arena: Arena): void
  (e: 'switchCompetition', compId: string): void
}>()

const activeCompId = computed(() => props.competitionId || 'b26d7145-d341-4795-ac58-bff86d13cbf8')
const { data: compDetail } = useCompetitionQuery(activeCompId)
const activeCompetitionName = computed(() => compDetail.value?.name || 'Kejuaraan Terpilih')

// All competitions for target selector
const compFilter = ref<CompetitionFilterValues>({ page: 1, limit: 50, status: 'all' })
const { data: competitionsData } = useCompetitionsQuery(compFilter)
const availableCompetitions = computed(() => competitionsData.value?.data || [])

// Queries & Mutations
const { data: rawArenas, isLoading, isFetching, error, refetch } = useArenasQuery(activeCompId)
const createMutation = useCreateArenaMutation(activeCompId)
const updateMutation = useUpdateArenaMutation(activeCompId)
const deleteMutation = useDeleteArenaMutation(activeCompId)
const seedMutation = useSeedArenasMutation(activeCompId)

const arenas = computed(() => rawArenas.value || [])

// UI Search
const searchQuery = ref('')

// Toast State
const toast = ref<{ title: string; message: string; type: 'success' | 'danger' | 'info' } | null>(null)
let toastTimer: ReturnType<typeof setTimeout> | null = null

function showToast(title: string, message: string, type: 'success' | 'danger' | 'info' = 'success') {
  if (toastTimer) clearTimeout(toastTimer)
  toast.value = { title, message, type }
  toastTimer = setTimeout(() => {
    toast.value = null
  }, 4000)
}

// Modal Form State (Create / Edit)
const isModalOpen = ref(false)
const modalMode = ref<'create' | 'edit'>('create')
const editingArenaId = ref<string | null>(null)
const selectedTargetCompId = ref(activeCompId.value)
const arenaName = ref('')
const formError = ref('')

function openCreateModal() {
  modalMode.value = 'create'
  editingArenaId.value = null
  selectedTargetCompId.value = activeCompId.value
  arenaName.value = ''
  formError.value = ''
  isModalOpen.value = true
}

function openEditModal(arena: Arena) {
  modalMode.value = 'edit'
  editingArenaId.value = arena.id
  selectedTargetCompId.value = activeCompId.value
  arenaName.value = arena.name
  formError.value = ''
  isModalOpen.value = true
}

async function handleSubmitArena() {
  formError.value = ''
  const parseResult = createArenaSchema.safeParse({ name: arenaName.value.trim() })

  if (!parseResult.success) {
    formError.value = parseResult.error.issues[0]?.message || 'Nama gelanggang tidak valid'
    return
  }

  try {
    if (modalMode.value === 'create') {
      const targetId = selectedTargetCompId.value || activeCompId.value
      const created = await createMutation.mutateAsync({
        name: arenaName.value.trim(),
        compId: targetId,
      })
      const targetCompName =
        availableCompetitions.value.find((c) => c.id === targetId)?.name || activeCompetitionName.value
      showToast('Gelanggang Dibuat', `${created.name} berhasil ditambahkan ke ${targetCompName}.`, 'success')

      if (targetId !== activeCompId.value) {
        emit('switchCompetition', targetId)
      }
    } else if (editingArenaId.value) {
      const updated = await updateMutation.mutateAsync({
        id: editingArenaId.value,
        name: arenaName.value.trim(),
      })
      showToast('Gelanggang Diperbarui', `Nama gelanggang diubah menjadi ${updated.name}.`, 'success')
    }
    isModalOpen.value = false
  } catch (err) {
    showToast('Terjadi Kesalahan', getErrorMessage(err, 'Gagal menyimpan data gelanggang.'), 'danger')
  }
}

// Delete Confirmation
const deletingId = ref<string | null>(null)

async function handleDeleteArena(arena: Arena) {
  if (!confirm(`Apakah Anda yakin ingin menghapus ${arena.name}?`)) return
  deletingId.value = arena.id
  try {
    await deleteMutation.mutateAsync(arena.id)
    showToast('Gelanggang Dihapus', `${arena.name} telah dihapus dari sistem.`, 'info')
  } catch (err) {
    showToast('Gagal Menghapus', getErrorMessage(err, 'Gagal menghapus gelanggang.'), 'danger')
  } finally {
    deletingId.value = null
  }
}

// Batch Seed Standard Mats
async function handleSeedStandard() {
  try {
    const result = await seedMutation.mutateAsync()
    if (result.created > 0) {
      showToast(
        'Template Gelanggang Diterapkan',
        `${result.created} gelanggang standar berhasil dibuat (${result.skipped} gelanggang dilewati).`,
        'success'
      )
    } else {
      showToast(
        'Gelanggang Sudah Siap',
        'Gelanggang standar kompetisi sudah terdaftar.',
        'info'
      )
    }
  } catch (err) {
    showToast('Gagal Menerapkan Template', getErrorMessage(err, 'Terjadi gangguan.'), 'danger')
  }
}

// Open External Links
function openScoreboard(arena: Arena) {
  window.open(`/scoreboard/${arena.id}`, '_blank')
}

function openRefereeConsole() {
  window.open('/referee', '_blank')
}

// Filtered Arenas
const filteredArenas = computed(() => {
  let list = [...arenas.value]
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase().trim()
    list = list.filter((a) => a.name.toLowerCase().includes(q))
  }
  return list
})
</script>

<template>
  <div class="arena-ops @container" role="region" aria-label="Manajemen dan Monitoring Gelanggang">
    <!-- Toast Notification -->
    <transition name="arena-fade">
      <div v-if="toast" class="arena-toast" :class="`arena-toast--${toast.type}`" role="status" aria-live="polite">
        <div class="arena-toast__dot" />
        <div class="arena-toast__content">
          <strong class="arena-toast__title">{{ toast.title }}</strong>
          <span class="arena-toast__msg">{{ toast.message }}</span>
        </div>
        <button class="arena-toast__close" aria-label="Tutup notifikasi" @click="toast = null">✕</button>
      </div>
    </transition>

    <!-- Header Banner -->
    <header class="arena-header">
      <div class="arena-header__left">
        <div class="arena-header__icon-box">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
            <rect width="18" height="18" x="3" y="3" rx="2" />
            <circle cx="12" cy="12" r="4" />
          </svg>
        </div>
        <div>
          <div class="arena-header__sub font-mono">
            <span>OPERASIONAL & MONITORING GELANGGANG</span>
            <span class="arena-bullet">•</span>
            <span class="arena-comp-badge">EVENT: {{ activeCompetitionName.toUpperCase() }}</span>
            <span class="arena-bullet">•</span>
            <span class="arena-pulse-wrapper">
              <span class="arena-pulse-dot" />
              <span>SISTEM DIGITAL SCORING</span>
            </span>
          </div>
          <h2 class="arena-header__title">GELANGGANG PERTANDINGAN</h2>
        </div>
      </div>

      <div class="arena-header__actions font-mono">
        <AppButton
          variant="secondary"
          size="sm"
          :loading="isFetching"
          @click="() => refetch()"
        >
          <template #icon>
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" />
              <path d="M21 3v5h-5" />
              <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" />
              <path d="M3 21v-5h5" />
            </svg>
          </template>
          Segarkan
        </AppButton>

        <AppButton
          variant="secondary"
          size="sm"
          :loading="seedMutation.isPending.value"
          @click="handleSeedStandard"
        >
          <template #icon>
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect width="7" height="7" x="3" y="3" rx="1" />
              <rect width="7" height="7" x="14" y="3" rx="1" />
              <rect width="7" height="7" x="14" y="14" rx="1" />
              <rect width="7" height="7" x="3" y="14" rx="1" />
            </svg>
          </template>
          Template Standar (Gelanggang A-D)
        </AppButton>

        <AppButton
          variant="emerald"
          size="sm"
          @click="openCreateModal"
        >
          <template #icon>
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <line x1="12" y1="5" x2="12" y2="19" />
              <line x1="5" y1="12" x2="19" y2="12" />
            </svg>
          </template>
          Tambah Gelanggang
        </AppButton>
      </div>
    </header>

    <!-- Metrics Strip -->
    <section class="arena-metrics" aria-label="Statistik Gelanggang">
      <div class="arena-metric-card">
        <span class="arena-metric-label">TOTAL GELANGGANG</span>
        <div class="arena-metric-value-row">
          <span class="arena-metric-val tabular-nums">{{ arenas.length }}</span>
          <span class="arena-metric-badge font-mono">ARENA</span>
        </div>
        <span class="arena-metric-desc">Tersinkronisasi turnamen</span>
      </div>

      <div class="arena-metric-card arena-metric-card--active">
        <div class="arena-metric-corner-tag font-mono">LIVE</div>
        <span class="arena-metric-label">STATUS OPERASIONAL</span>
        <div class="arena-metric-value-row">
          <span class="arena-metric-val text-emerald">SIAP TANDING</span>
        </div>
        <span class="arena-metric-desc">Sinkronisasi WebSocket Aktif</span>
      </div>

      <div class="arena-metric-card">
        <span class="arena-metric-label">STANDAR LAPANGAN</span>
        <div class="arena-metric-value-row">
          <span class="arena-metric-val-text font-mono">8 x 8 METER</span>
        </div>
        <span class="arena-metric-desc">Matras Standar PERSILAT</span>
      </div>

      <div class="arena-metric-card">
        <span class="arena-metric-label">PERANGKAT & JURI</span>
        <div class="arena-metric-value-row">
          <span class="arena-metric-val-text font-mono">3 - 5 JURI</span>
        </div>
        <span class="arena-metric-desc">Sudut Merah & Biru Terkoneksi</span>
      </div>
    </section>

    <!-- Controls Toolbar -->
    <section class="arena-toolbar">
      <div class="arena-toolbar__info font-mono">
        <span class="arena-indicator-dot" />
        <span>DAFTAR GELANGGANG AKTIF ({{ arenas.length }})</span>
      </div>

      <div class="arena-toolbar__search-group">
        <div class="arena-search-box">
          <svg class="arena-search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            v-model="searchQuery"
            type="search"
            class="arena-search-input font-mono"
            placeholder="Cari nama gelanggang..."
            aria-label="Cari gelanggang"
          />
          <button
            v-if="searchQuery"
            class="arena-search-clear"
            aria-label="Hapus pencarian"
            @click="searchQuery = ''"
          >
            ✕
          </button>
        </div>
      </div>
    </section>

    <!-- Skeleton Loading -->
    <div v-if="isLoading" class="arena-grid" aria-busy="true">
      <div v-for="i in 3" :key="i" class="arena-skeleton-card">
        <div class="skeleton-line skeleton-header" />
        <div class="skeleton-line skeleton-title" />
        <div class="skeleton-line skeleton-specs" />
        <div class="skeleton-line skeleton-btn" />
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="arena-empty-state arena-empty-state--error">
      <div class="arena-empty-icon arena-empty-icon--error">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
      </div>
      <h3 class="arena-empty-title">Gagal Memuat Gelanggang</h3>
      <p class="arena-empty-desc">{{ getErrorMessage(error, 'Terjadi gangguan jaringan saat memuat gelanggang.') }}</p>
      <AppButton variant="secondary" size="md" @click="() => refetch()">
        Coba Lagi
      </AppButton>
    </div>

    <!-- Empty State -->
    <div v-else-if="filteredArenas.length === 0" class="arena-empty-state">
      <div class="arena-empty-icon">
        <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
          <rect width="18" height="18" x="3" y="3" rx="2" />
          <circle cx="12" cy="12" r="4" />
        </svg>
      </div>
      <h3 class="arena-empty-title">
        {{ searchQuery ? 'Tidak Ada Gelanggang yang Cocok' : 'Belum Ada Gelanggang' }}
      </h3>
      <p class="arena-empty-desc">
        {{
          searchQuery
            ? `Pencarian "${searchQuery}" tidak menghasilkan data gelanggang.`
            : 'Kejuaraan ini belum memiliki gelanggang pertandingan. Terapkan template standar atau tambahkan manual.'
        }}
      </p>

      <div class="arena-empty-actions font-mono">
        <AppButton
          v-if="searchQuery"
          variant="secondary"
          size="md"
          @click="searchQuery = ''"
        >
          Reset Pencarian
        </AppButton>
        <template v-else>
          <AppButton
            variant="emerald"
            size="md"
            :loading="seedMutation.isPending.value"
            @click="handleSeedStandard"
          >
            Terapkan Gelanggang Standar (A s/d D)
          </AppButton>
          <AppButton
            variant="secondary"
            size="md"
            @click="openCreateModal"
          >
            Tambah Gelanggang Kustom
          </AppButton>
        </template>
      </div>
    </div>

    <!-- Responsive Cards Grid -->
    <div v-else class="arena-grid" role="list">
      <article
        v-for="arena in filteredArenas"
        :key="arena.id"
        class="arena-card"
        role="listitem"
      >
        <!-- Corner Accent Stripe -->
        <div class="arena-card__accent" />

        <!-- Header -->
        <header class="arena-card__header">
          <div class="arena-card__badge-row">
            <span class="arena-badge-mat font-mono">
              <span class="arena-badge-dot" />
              ARENA RESMI
            </span>
            <span class="arena-badge-status font-mono">STANDBY / READY</span>
          </div>

          <span class="arena-chip-id font-mono">ID: {{ arena.id.slice(0, 8) }}</span>
        </header>

        <!-- Body -->
        <div class="arena-card__content">
          <h3 class="arena-card__title">{{ arena.name }}</h3>

          <div class="arena-card__spec-box font-mono">
            <div class="arena-spec-row">
              <span class="arena-spec-label">DIMENSI GELANGGANG</span>
              <span class="arena-spec-val">8 x 8 METER (SUDUT MERAH & BIRU)</span>
            </div>
            <div class="arena-spec-row">
              <span class="arena-spec-label">SINKRONISASI SERVER</span>
              <span class="arena-spec-val text-emerald">WEBSOCKET AKTIF</span>
            </div>
          </div>

          <!-- Quick Console Links -->
          <div class="arena-links-row font-mono">
            <button
              type="button"
              class="arena-link-btn arena-link--score"
              title="Buka layar scoreboard arena di tab baru"
              @click="openScoreboard(arena)"
            >
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect width="20" height="14" x="2" y="3" rx="2" />
                <line x1="8" y1="21" x2="16" y2="21" />
                <line x1="12" y1="17" x2="12" y2="21" />
              </svg>
              Layar Scoreboard
            </button>

            <button
              type="button"
              class="arena-link-btn arena-link--referee"
              title="Buka konsol wasit / dewan juri"
              @click="openRefereeConsole"
            >
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 2v20" />
                <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
              </svg>
              Konsol Wasit
            </button>
          </div>
        </div>

        <!-- Footer -->
        <footer class="arena-card__footer font-mono">
          <button
            type="button"
            class="arena-select-btn"
            @click="emit('selectArena', arena)"
          >
            <span>Pilih untuk Bagan</span>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6" />
            </svg>
          </button>

          <div class="arena-manage-btns">
            <button
              type="button"
              class="arena-btn-icon"
              title="Ubah nama gelanggang"
              @click="openEditModal(arena)"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" />
              </svg>
            </button>

            <button
              type="button"
              class="arena-btn-icon arena-btn-icon--danger"
              title="Hapus gelanggang"
              :disabled="deletingId === arena.id"
              @click="handleDeleteArena(arena)"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M3 6h18" />
                <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" />
                <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
              </svg>
            </button>
          </div>
        </footer>
      </article>
    </div>

    <!-- Modal Form (Tambah / Edit Gelanggang) -->
    <div
      v-if="isModalOpen"
      class="arena-modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="arena-modal-title"
      @click.self="isModalOpen = false"
    >
      <div class="arena-modal">
        <header class="arena-modal__header">
          <div>
            <div class="arena-modal__tag font-mono">OPERASIONAL LAPANGAN</div>
            <h3 id="arena-modal-title" class="arena-modal__title">
              {{ modalMode === 'create' ? 'Tambah Gelanggang Baru' : 'Ubah Data Gelanggang' }}
            </h3>
          </div>
          <button
            class="arena-modal__close"
            aria-label="Tutup dialog"
            @click="isModalOpen = false"
          >
            ✕
          </button>
        </header>
 
        <form class="arena-modal__body" @submit.prevent="handleSubmitArena">
          <!-- Interactive Competition Selector in Modal -->
          <div class="arena-form-group">
            <label for="arena-comp-select" class="arena-form-label font-mono">
              TARGET KEJUARAAN / TURNAMEN <span class="arena-req">*</span>
            </label>
            <div class="arena-select-wrap">
              <select
                id="arena-comp-select"
                v-model="selectedTargetCompId"
                class="arena-form-select font-mono"
                :disabled="modalMode === 'edit'"
              >
                <template v-if="availableCompetitions.length > 0">
                  <option
                    v-for="c in availableCompetitions"
                    :key="c.id"
                    :value="c.id"
                  >
                    {{ c.name }} ({{ c.status.toUpperCase() }})
                  </option>
                </template>
                <template v-else>
                  <option :value="activeCompId">{{ activeCompetitionName }}</option>
                </template>
              </select>
            </div>
            <span class="arena-form-help font-mono">
              {{ modalMode === 'create' ? 'Pilih kejuaraan tempat gelanggang ini akan didaftarkan.' : 'Kejuaraan tempat gelanggang ini terdaftar.' }}
            </span>
          </div>

          <div class="arena-form-group">
            <label for="arena-name-input" class="arena-form-label font-mono">
              NAMA GELANGGANG <span class="arena-req">*</span>
            </label>
            <input
              id="arena-name-input"
              v-model="arenaName"
              type="text"
              class="arena-form-input font-mono"
              :class="{ 'arena-form-input--error': formError }"
              placeholder="Contoh: Gelanggang A (Utama), Gelanggang 1, dll."
              required
            />
            <span v-if="formError" class="arena-form-error">{{ formError }}</span>
            <span class="arena-form-help font-mono">Nama gelanggang resmi yang akan ditampilkan pada layar pertandingan.</span>
          </div>

          <footer class="arena-modal__footer font-mono">
            <AppButton
              type="button"
              variant="secondary"
              size="md"
              @click="isModalOpen = false"
            >
              Batal
            </AppButton>
            <AppButton
              type="submit"
              variant="emerald"
              size="md"
              :loading="createMutation.isPending.value || updateMutation.isPending.value"
            >
              Simpan Gelanggang
            </AppButton>
          </footer>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.arena-ops {
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: 100%;
  color: var(--text-main, #f8fafc);
  box-sizing: border-box;
}

/* Toast */
.arena-toast {
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
.arena-toast--success { border-left: 4px solid var(--color-emerald-500, #10b981); }
.arena-toast--danger { border-left: 4px solid var(--color-primary-600, #dc2626); }
.arena-toast--info { border-left: 4px solid var(--color-secondary-600, #2563eb); }
.arena-toast__dot { width: 8px; height: 8px; border-radius: 50%; background-color: currentColor; }
.arena-toast__content { display: flex; flex-direction: column; gap: 2px; }
.arena-toast__title { font-size: 0.875rem; font-weight: 700; }
.arena-toast__msg { font-size: 0.8125rem; color: var(--color-neutral-400, #94a3b8); }
.arena-toast__close { background: none; border: none; color: var(--color-neutral-500, #64748b); cursor: pointer; font-size: 1rem; }
.arena-fade-enter-active, .arena-fade-leave-active { transition: opacity 0.2s ease, transform 0.2s ease; }
.arena-fade-enter-from, .arena-fade-leave-to { opacity: 0; transform: translateY(-8px); }

/* Header */
.arena-header {
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
.arena-header__left { display: flex; align-items: center; gap: 16px; }
.arena-header__icon-box {
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
.arena-header__sub {
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
.arena-bullet { color: var(--color-neutral-600, #475569); }
.arena-tag { color: var(--color-primary-400, #f87171); }
.arena-pulse-wrapper { display: inline-flex; align-items: center; gap: 5px; color: var(--color-emerald-400, #34d399); }
.arena-pulse-dot {
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
.arena-header__title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.5rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #ffffff;
  line-height: 1.2;
}
.arena-header__actions { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }

/* Metrics */
.arena-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 12px;
}
.arena-metric-card {
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
.arena-metric-card--active { border-left: 3px solid var(--color-emerald-500, #10b981); }
.arena-metric-corner-tag {
  position: absolute;
  top: 10px;
  right: 12px;
  font-size: 0.625rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--color-emerald-400, #34d399);
}
.arena-metric-label {
  font-family: var(--font-mono, monospace);
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--color-neutral-400, #94a3b8);
}
.arena-metric-value-row { display: flex; align-items: baseline; gap: 8px; margin-top: 2px; }
.arena-metric-val {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.875rem;
  font-weight: 800;
  color: #ffffff;
  line-height: 1;
}
.arena-metric-val-text {
  font-size: 1.25rem;
  font-weight: 800;
  color: #ffffff;
  line-height: 1;
}
.text-emerald { color: var(--color-emerald-400, #34d399) !important; font-size: 1.25rem; }
.arena-metric-badge {
  font-size: 0.6875rem;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--color-neutral-400, #94a3b8);
}
.arena-metric-desc { font-size: 0.75rem; color: var(--color-neutral-500, #64748b); margin-top: 2px; }

/* Toolbar */
.arena-toolbar {
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
.arena-toolbar__info { display: flex; align-items: center; gap: 8px; font-size: 0.75rem; font-weight: 700; color: #ffffff; }
.arena-indicator-dot { width: 6px; height: 6px; border-radius: 50%; background-color: var(--color-emerald-400, #34d399); }
.arena-toolbar__search-group { display: flex; align-items: center; gap: 10px; flex: 1; max-width: 380px; justify-content: flex-end; }
.arena-search-box { position: relative; display: flex; align-items: center; width: 100%; }
.arena-search-icon { position: absolute; left: 10px; color: var(--color-neutral-500, #64748b); pointer-events: none; }
.arena-search-input {
  width: 100%;
  padding: 8px 32px 8px 34px;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  color: #ffffff;
  font-size: 0.75rem;
  outline: none;
}
.arena-search-input:focus { border-color: var(--color-emerald-400, #34d399); }
.arena-search-clear { position: absolute; right: 10px; background: none; border: none; color: var(--color-neutral-400, #94a3b8); cursor: pointer; font-size: 0.8125rem; }

/* Grid */
.arena-grid {
  display: grid;
  grid-template-columns: repeat(1, minmax(0, 1fr));
  gap: 16px;
}
@container (min-width: 640px) {
  .arena-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@container (min-width: 1024px) {
  .arena-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}

/* Card */
.arena-card {
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #1e293b);
  border-radius: var(--radius-md, 10px);
  overflow: hidden;
  transition: transform 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease;
}
.arena-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.35);
}
.arena-card__accent {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, #059669, #10b981);
}

.arena-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 18px 10px 18px;
}
.arena-card__badge-row { display: flex; align-items: center; gap: 6px; }
.arena-badge-mat {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.625rem;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 4px;
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
  border: 1px solid rgba(16, 185, 129, 0.3);
}
.arena-badge-dot { width: 5px; height: 5px; border-radius: 50%; background: #34d399; }
.arena-badge-status { font-size: 0.625rem; color: var(--color-neutral-400, #94a3b8); }
.arena-chip-id { font-size: 0.625rem; color: var(--color-neutral-500, #64748b); }

.arena-card__content { padding: 0 18px 14px 18px; }
.arena-card__title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.25rem;
  font-weight: 800;
  color: #ffffff;
  line-height: 1.3;
  margin-bottom: 12px;
}
.arena-card__spec-box {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 6px;
  margin-bottom: 12px;
}
.arena-spec-row { display: flex; justify-content: space-between; align-items: center; }
.arena-spec-label { font-size: 0.5625rem; color: var(--color-neutral-500, #64748b); font-weight: 600; }
.arena-spec-val { font-size: 0.6875rem; color: #ffffff; font-weight: 700; }

.arena-links-row { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
.arena-link-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 6px 10px;
  border-radius: 6px;
  font-size: 0.6875rem;
  font-weight: 700;
  cursor: pointer;
  border: 1px solid transparent;
  transition: all 0.15s ease;
}
.arena-link--score {
  background: rgba(37, 99, 235, 0.12);
  color: #93c5fd;
  border-color: rgba(37, 99, 235, 0.25);
}
.arena-link--score:hover {
  background: rgba(37, 99, 235, 0.25);
  color: #ffffff;
}
.arena-link--referee {
  background: rgba(220, 38, 38, 0.12);
  color: #fca5a5;
  border-color: rgba(220, 38, 38, 0.25);
}
.arena-link--referee:hover {
  background: rgba(220, 38, 38, 0.25);
  color: #ffffff;
}

.arena-card__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 18px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(0, 0, 0, 0.15);
}
.arena-select-btn {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #ffffff;
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.15s ease;
}
.arena-select-btn:hover {
  background: rgba(16, 185, 129, 0.2);
  border-color: rgba(16, 185, 129, 0.4);
  color: #34d399;
}
.arena-manage-btns { display: flex; align-items: center; gap: 4px; }
.arena-btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 4px;
  background: transparent;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  cursor: pointer;
  transition: all 0.15s ease;
}
.arena-btn-icon:hover { background: rgba(255, 255, 255, 0.08); color: #ffffff; }
.arena-btn-icon--danger:hover { background: rgba(220, 38, 38, 0.2); color: #f87171; }

/* Skeleton */
.arena-skeleton-card {
  height: 200px;
  background: rgba(19, 29, 51, 0.6);
  border: 1px solid var(--border-card, #1e293b);
  border-radius: 10px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.skeleton-line {
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.04) 25%, rgba(255, 255, 255, 0.08) 50%, rgba(255, 255, 255, 0.04) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 4px;
}
.skeleton-header { width: 45%; height: 20px; }
.skeleton-title { width: 70%; height: 26px; }
.skeleton-specs { width: 100%; height: 48px; }
.skeleton-btn { width: 100%; height: 32px; margin-top: auto; }
@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* Empty State */
.arena-empty-state {
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
.arena-empty-icon {
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
.arena-empty-icon--error { background: rgba(220, 38, 38, 0.15); color: var(--color-primary-400, #f87171); }
.arena-empty-title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.25rem;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 8px;
}
.arena-empty-desc {
  max-width: 520px;
  font-size: 0.875rem;
  color: var(--color-neutral-400, #94a3b8);
  line-height: 1.5;
  margin-bottom: 20px;
}
.arena-empty-actions { display: flex; align-items: center; gap: 12px; }

/* Modal */
.arena-modal-overlay {
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
.arena-modal {
  width: 100%;
  max-width: 500px;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
  overflow: hidden;
}
.arena-modal__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-card, #1e293b);
}
.arena-modal__tag {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--color-emerald-400, #34d399);
  margin-bottom: 4px;
}
.arena-modal__title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.25rem;
  font-weight: 800;
  color: #ffffff;
}
.arena-modal__close { background: none; border: none; color: var(--color-neutral-400, #94a3b8); cursor: pointer; font-size: 1.25rem; }
.arena-modal__body { padding: 20px 24px; display: flex; flex-direction: column; gap: 16px; }
.arena-form-group { display: flex; flex-direction: column; gap: 6px; }
.arena-form-label { font-size: 0.6875rem; font-weight: 700; letter-spacing: 0.06em; color: var(--color-neutral-300, #cbd5e1); }
.arena-req { color: var(--color-primary-500, #ef4444); }
.arena-form-input {
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
.arena-target-comp {
  display: flex;
  align-items: center;
  gap: 10px;
  background: rgba(245, 158, 11, 0.08);
  border: 1px solid rgba(245, 158, 11, 0.25);
  border-radius: 8px;
  padding: 10px 14px;
}
.target-comp-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.target-comp-label {
  font-size: 0.625rem;
  font-weight: 700;
  color: #F59E0B;
  letter-spacing: 0.06em;
}
.arena-comp-badge {
  color: #F59E0B;
  font-weight: 700;
}
.arena-form-select {
  width: 100%;
  padding: 10px 14px;
  background: rgba(15, 23, 42, 0.9);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  color: #ffffff;
  font-size: 0.8125rem;
  outline: none;
  box-sizing: border-box;
  cursor: pointer;
}
.arena-form-select:focus {
  border-color: var(--color-emerald-400, #34d399);
}
.arena-form-select:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  background: rgba(15, 23, 42, 0.5);
}
.arena-form-help {
  font-size: 0.6875rem;
  color: var(--color-neutral-400, #94a3b8);
  margin-top: 2px;
}
.arena-modal__footer { display: flex; justify-content: flex-end; gap: 10px; margin-top: 10px; }
</style>
