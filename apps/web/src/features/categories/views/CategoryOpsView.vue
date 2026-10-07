<script setup lang="ts">
import { getErrorMessage } from '@/api/error'
import { ref, computed } from 'vue'
import { AppButton } from '@/components/ui'
import {
  useCategoriesQuery,
  useCreateCategoryMutation,
  useSeedPresetsMutation,
} from '../api/categoryQueries'
import {
  createCategorySchema,
  IPSI_STANDARD_PRESETS,
  type Category,
  type CreateCategoryFormValues,
} from '../types/category.schema'

const props = withDefaults(
  defineProps<{
    competitionId?: string
    competitionName?: string
  }>(),
  {
    competitionId: 'b26d7145-d341-4795-ac58-bff86d13cbf8',
    competitionName: 'Kejuaraan Nasional Pencak Silat',
  },
)

const emit = defineEmits<{
  (e: 'selectCategory', category: Category): void
}>()

// Reactive Competition ID
const activeCompId = computed(() => props.competitionId || 'b26d7145-d341-4795-ac58-bff86d13cbf8')

// Queries & Mutations
const {
  data: rawCategories,
  isLoading,
  isFetching,
  error,
  refetch,
} = useCategoriesQuery(activeCompId)
const createMutation = useCreateCategoryMutation(activeCompId)
const seedMutation = useSeedPresetsMutation(activeCompId)

const categories = computed(() => rawCategories.value || [])

// UI Filter & Search State
const activeGenderTab = ref<'all' | 'male' | 'female'>('all')
const searchQuery = ref('')
const sortBy = ref<'class' | 'name'>('class')

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

// Modal Form State (Create Category)
const isModalOpen = ref(false)
const selectedPresetIndex = ref<number | ''>('')
const formErrors = ref<Record<string, string>>({})
const formState = ref<{
  name: string
  gender: 'male' | 'female' | 'putra' | 'putri'
  weight_class: string
}>({
  name: '',
  gender: 'male',
  weight_class: 'A',
})

function openCreateModal() {
  selectedPresetIndex.value = ''
  formState.value = {
    name: '',
    gender: 'male',
    weight_class: 'A',
  }
  formErrors.value = {}
  isModalOpen.value = true
}

function handlePresetSelect(event: Event) {
  const target = event.target as HTMLSelectElement
  const val = target.value
  if (val === '') return

  const idx = Number(val)
  selectedPresetIndex.value = idx
  const preset = IPSI_STANDARD_PRESETS[idx]
  if (preset) {
    formState.value.name = preset.categoryName
    formState.value.gender = preset.gender
    formState.value.weight_class = preset.code
    formErrors.value = {}
  }
}

async function handleSubmitCategory() {
  formErrors.value = {}
  const parseResult = createCategorySchema.safeParse(formState.value)

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
    const payload: CreateCategoryFormValues = {
      name: formState.value.name.trim(),
      gender: formState.value.gender,
      weight_class: formState.value.weight_class.trim().toUpperCase(),
    }

    const created = await createMutation.mutateAsync(payload)
    isModalOpen.value = false
    showToast(
      'Kategori Dibuat',
      `Kategori ${created.name} berhasil ditambahkan ke kompetisi.`,
      'success',
    )
  } catch (err) {
    const message = getErrorMessage(err, 'Gagal menambahkan kategori ke server.')
    showToast('Terjadi Kesalahan', message, 'danger')
  }
}

// Batch Seed Standard IPSI
async function handleSeedPresets() {
  try {
    const result = await seedMutation.mutateAsync(IPSI_STANDARD_PRESETS)
    if (result.created > 0) {
      showToast(
        'Sinkronisasi Kategori Berhasil',
        `${result.created} kelas kategori standar IPSI berhasil dibuat (${result.skipped} kelas dilewati karena sudah ada).`,
        'success',
      )
    } else {
      showToast(
        'Kategori Sudah Lengkap',
        `Semua ${result.skipped} kelas tanding standar IPSI sudah terdaftar di kompetisi ini.`,
        'info',
      )
    }
  } catch (err) {
    showToast('Gagal Sinkronisasi', getErrorMessage(err, 'Gagal memuat standar IPSI.'), 'danger')
  }
}

// Stats & Metrics
const stats = computed(() => {
  const all = categories.value
  const male = all.filter((c) => c.gender === 'male' || c.gender === 'putra').length
  const female = all.filter((c) => c.gender === 'female' || c.gender === 'putri').length
  return {
    total: all.length,
    male,
    female,
  }
})

// Filtered & Sorted Categories
const filteredCategories = computed(() => {
  let list = [...categories.value]

  // Filter gender
  if (activeGenderTab.value === 'male') {
    list = list.filter((c) => c.gender === 'male' || c.gender === 'putra')
  } else if (activeGenderTab.value === 'female') {
    list = list.filter((c) => c.gender === 'female' || c.gender === 'putri')
  }

  // Filter search
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase().trim()
    list = list.filter(
      (c) =>
        c.name.toLowerCase().includes(q) ||
        c.weight_class.toLowerCase().includes(q) ||
        c.gender.toLowerCase().includes(q),
    )
  }

  // Sort
  list.sort((a, b) => {
    if (sortBy.value === 'class') {
      // Sort by gender first (putra then putri), then weight_class
      const gOrderA = a.gender === 'male' || a.gender === 'putra' ? 0 : 1
      const gOrderB = b.gender === 'male' || b.gender === 'putri' ? 0 : 1
      if (gOrderA !== gOrderB) return gOrderA - gOrderB
      return a.weight_class.localeCompare(b.weight_class, undefined, { numeric: true })
    }
    return a.name.localeCompare(b.name)
  })

  return list
})

function isMaleGender(gender: string): boolean {
  return gender === 'male' || gender === 'putra'
}

function handleCategoryClick(cat: Category) {
  emit('selectCategory', cat)
}
</script>

<template>
  <div class="cat-ops @container" role="region" aria-label="Manajemen Kategori Tanding">
    <!-- Toast Notification -->
    <transition name="cat-fade">
      <div
        v-if="toast"
        class="cat-toast"
        :class="`cat-toast--${toast.type}`"
        role="status"
        aria-live="polite"
      >
        <div class="cat-toast__dot" />
        <div class="cat-toast__content">
          <strong class="cat-toast__title">{{ toast.title }}</strong>
          <span class="cat-toast__msg">{{ toast.message }}</span>
        </div>
        <button class="cat-toast__close" aria-label="Tutup notifikasi" @click="toast = null">
          ✕
        </button>
      </div>
    </transition>

    <!-- Header Banner & Action Buttons -->
    <header class="cat-header">
      <div class="cat-header__left">
        <div class="cat-header__icon-box">
          <svg
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.2"
          >
            <path d="M6 3h12l4 6-10 12L2 9l4-6z" />
            <path d="M11 3 8 9l4 12 4-12-3-6" />
            <path d="M2 9h20" />
          </svg>
        </div>
        <div>
          <div class="cat-header__sub font-mono">
            <span>SISTEM ELIMINASI IPSI RULES 2023</span>
            <span class="cat-bullet">•</span>
            <span class="cat-comp-badge"
              >EVENT: {{ (competitionName || 'Kejuaraan').toUpperCase() }}</span
            >
            <span class="cat-bullet">•</span>
            <span class="cat-pulse-wrapper">
              <span class="cat-pulse-dot" />
              <span>LIVE SERVER</span>
            </span>
          </div>
          <h2 class="cat-header__title">KATEGORI TANDING</h2>
        </div>
      </div>

      <div class="cat-header__actions font-mono">
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
          variant="secondary"
          size="sm"
          :loading="seedMutation.isPending.value"
          @click="handleSeedPresets"
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
              <path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1-2.5-2.5Z" />
              <path d="M6 6h10" />
              <path d="M6 10h10" />
            </svg>
          </template>
          Template Standar IPSI
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
          Tambah Kategori
        </AppButton>
      </div>
    </header>

    <!-- Metrics Strip (Better Typography & Tabular Nums) -->
    <section class="cat-metrics" aria-label="Statistik Kategori">
      <div class="cat-metric-card">
        <span class="cat-metric-label">TOTAL KELAS TANDING</span>
        <div class="cat-metric-value-row">
          <span class="cat-metric-val tabular-nums">{{ stats.total }}</span>
          <span class="cat-metric-badge font-mono">KELAS</span>
        </div>
        <span class="cat-metric-desc">Tersinkronisasi database pusat</span>
      </div>

      <div class="cat-metric-card cat-metric-card--male">
        <div class="cat-metric-corner-tag font-mono">SUDUT MERAH</div>
        <span class="cat-metric-label">DEWASA PUTRA</span>
        <div class="cat-metric-value-row">
          <span class="cat-metric-val cat-val-male tabular-nums">{{ stats.male }}</span>
          <span class="cat-metric-badge cat-badge-male font-mono">KELAS A - J</span>
        </div>
        <span class="cat-metric-desc">Golongan Putra Remaja / Dewasa</span>
      </div>

      <div class="cat-metric-card cat-metric-card--female">
        <div class="cat-metric-corner-tag font-mono">SUDUT BIRU</div>
        <span class="cat-metric-label">DEWASA PUTRI</span>
        <div class="cat-metric-value-row">
          <span class="cat-metric-val cat-val-female tabular-nums">{{ stats.female }}</span>
          <span class="cat-metric-badge cat-badge-female font-mono">KELAS A - F</span>
        </div>
        <span class="cat-metric-desc">Golongan Putri Remaja / Dewasa</span>
      </div>

      <div class="cat-metric-card">
        <span class="cat-metric-label">STANDAR ATURAN</span>
        <div class="cat-metric-value-row">
          <span class="cat-metric-val-text font-mono">IPSI 2023</span>
        </div>
        <span class="cat-metric-desc">Sistem Gugur Tunggal Digital</span>
      </div>
    </section>

    <!-- Controls Toolbar (Search & Filter Tabs) -->
    <section class="cat-toolbar">
      <div class="cat-toolbar__tabs font-mono" role="tablist">
        <button
          class="cat-tab-btn"
          :class="{ 'cat-tab-btn--active': activeGenderTab === 'all' }"
          role="tab"
          :aria-selected="activeGenderTab === 'all'"
          @click="activeGenderTab = 'all'"
        >
          Semua Kelas
          <span class="cat-tab-badge tabular-nums">{{ stats.total }}</span>
        </button>

        <button
          class="cat-tab-btn"
          :class="{ 'cat-tab-btn--active-male': activeGenderTab === 'male' }"
          role="tab"
          :aria-selected="activeGenderTab === 'male'"
          @click="activeGenderTab = 'male'"
        >
          <span class="cat-tab-dot cat-dot-male" />
          Dewasa Putra
          <span class="cat-tab-badge tabular-nums">{{ stats.male }}</span>
        </button>

        <button
          class="cat-tab-btn"
          :class="{ 'cat-tab-btn--active-female': activeGenderTab === 'female' }"
          role="tab"
          :aria-selected="activeGenderTab === 'female'"
          @click="activeGenderTab = 'female'"
        >
          <span class="cat-tab-dot cat-dot-female" />
          Dewasa Putri
          <span class="cat-tab-badge tabular-nums">{{ stats.female }}</span>
        </button>
      </div>

      <div class="cat-toolbar__search-group">
        <div class="cat-search-box">
          <svg
            class="cat-search-icon"
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
            class="cat-search-input font-mono"
            placeholder="Cari kelas (contoh: Kelas A, Putra, 50 Kg)..."
            aria-label="Pencarian kategori tanding"
          />
          <button
            v-if="searchQuery"
            class="cat-search-clear"
            aria-label="Hapus filter"
            @click="searchQuery = ''"
          >
            ✕
          </button>
        </div>

        <div class="cat-sort-select-wrapper">
          <select v-model="sortBy" class="cat-sort-select font-mono" aria-label="Urutkan Kategori">
            <option value="class">Urut: Golongan & Kelas</option>
            <option value="name">Urut: Nama Kategori (A-Z)</option>
          </select>
        </div>
      </div>
    </section>

    <!-- Loading State Skeleton -->
    <div v-if="isLoading" class="cat-grid" aria-busy="true">
      <div v-for="i in 6" :key="i" class="cat-skeleton-card">
        <div class="skeleton-line skeleton-header" />
        <div class="skeleton-line skeleton-title" />
        <div class="skeleton-line skeleton-specs" />
        <div class="skeleton-line skeleton-btn" />
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="cat-empty-state cat-empty-state--error">
      <div class="cat-empty-icon cat-empty-icon--error">
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
      <h3 class="cat-empty-title">Gagal Memuat Kategori</h3>
      <p class="cat-empty-desc">
        {{ getErrorMessage(error, 'Terjadi gangguan jaringan saat menghubungi server.') }}
      </p>
      <AppButton variant="secondary" size="md" @click="() => refetch()"> Coba Lagi </AppButton>
    </div>

    <!-- Empty State (No Categories) -->
    <div v-else-if="filteredCategories.length === 0" class="cat-empty-state">
      <div class="cat-empty-icon">
        <svg
          width="36"
          height="36"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.8"
        >
          <rect width="18" height="18" x="3" y="3" rx="2" />
          <path d="M9 3v18" />
          <path d="m14 9 3 3-3 3" />
        </svg>
      </div>
      <h3 class="cat-empty-title">
        {{ searchQuery ? 'Tidak Ada Kategori yang Cocok' : 'Belum Ada Kategori Tanding' }}
      </h3>
      <p class="cat-empty-desc">
        {{
          searchQuery
            ? `Pencarian "${searchQuery}" tidak menghasilkan kategori apapun. Silakan periksa kembali kata kunci.`
            : 'Kejuaraan ini belum memiliki kelas kategori tanding. Anda dapat menambahkan manual atau memuat template standar IPSI (Kelas A s/d J).'
        }}
      </p>

      <div class="cat-empty-actions font-mono">
        <AppButton v-if="searchQuery" variant="secondary" size="md" @click="searchQuery = ''">
          Reset Pencarian
        </AppButton>
        <template v-else>
          <AppButton
            variant="emerald"
            size="md"
            :loading="seedMutation.isPending.value"
            @click="handleSeedPresets"
          >
            Muat Template Standar IPSI (Kelas A-J)
          </AppButton>
          <AppButton variant="secondary" size="md" @click="openCreateModal">
            Buat Kategori Kustom
          </AppButton>
        </template>
      </div>
    </div>

    <!-- Responsive Cards Grid (Container Query Enabled) -->
    <div v-else class="cat-grid" role="list">
      <article
        v-for="cat in filteredCategories"
        :key="cat.id"
        class="cat-card"
        :class="{
          'cat-card--male': isMaleGender(cat.gender),
          'cat-card--female': !isMaleGender(cat.gender),
        }"
        role="listitem"
      >
        <!-- Corner Accent Stripe -->
        <div
          class="cat-card__accent"
          :class="isMaleGender(cat.gender) ? 'cat-accent--male' : 'cat-accent--female'"
        />

        <!-- Card Header: Badges & Class Code -->
        <header class="cat-card__header">
          <div class="cat-card__badges">
            <span
              class="cat-badge-gender font-mono"
              :class="
                isMaleGender(cat.gender) ? 'cat-badge-gender--male' : 'cat-badge-gender--female'
              "
            >
              <span class="cat-badge-dot" />
              {{ isMaleGender(cat.gender) ? 'PUTRA (MERAH)' : 'PUTRI (BIRU)' }}
            </span>
            <span class="cat-badge-rule font-mono">IPSI 2023</span>
          </div>

          <div class="cat-card__code font-mono">
            <span>KELAS</span>
            <strong>{{ cat.weight_class }}</strong>
          </div>
        </header>

        <!-- Card Content: Category Name & Weight Specs -->
        <div class="cat-card__content">
          <h3 class="cat-card__title">{{ cat.name }}</h3>

          <div class="cat-card__spec-box font-mono">
            <div class="cat-spec-item">
              <span class="cat-spec-label">KELAS BOBOT</span>
              <span class="cat-spec-value tabular-nums">KELAS {{ cat.weight_class }}</span>
            </div>
            <div class="cat-spec-divider" />
            <div class="cat-spec-item">
              <span class="cat-spec-label">GOLONGAN</span>
              <span class="cat-spec-value">{{
                isMaleGender(cat.gender) ? 'DEWASA PUTRA' : 'DEWASA PUTRI'
              }}</span>
            </div>
          </div>
        </div>

        <!-- Card Footer: Direct Bracket Action -->
        <footer class="cat-card__footer">
          <button
            type="button"
            class="cat-view-bracket-btn font-mono"
            :class="
              isMaleGender(cat.gender)
                ? 'cat-view-bracket-btn--male'
                : 'cat-view-bracket-btn--female'
            "
            @click="handleCategoryClick(cat)"
          >
            <span>Lihat Bagan Partai</span>
            <svg
              width="15"
              height="15"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2.2"
            >
              <line x1="5" y1="12" x2="19" y2="12" />
              <polyline points="12 5 19 12 12 19" />
            </svg>
          </button>
        </footer>
      </article>
    </div>

    <!-- Create Category Modal Dialog -->
    <div
      v-if="isModalOpen"
      class="cat-modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
      @click.self="isModalOpen = false"
    >
      <div class="cat-modal">
        <header class="cat-modal__header">
          <div>
            <div class="cat-modal__tag font-mono">FORM KATEGORI BARU</div>
            <h3 id="modal-title" class="cat-modal__title">Tambah Kategori Tanding</h3>
          </div>
          <button class="cat-modal__close" aria-label="Tutup dialog" @click="isModalOpen = false">
            ✕
          </button>
        </header>

        <form class="cat-modal__body" @submit.prevent="handleSubmitCategory">
          <!-- Active Competition Target Banner -->
          <div class="cat-target-comp font-mono">
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
            <div class="target-comp-text">
              <span class="target-comp-label">KEJUARAAN TUJUAN:</span>
              <span class="target-comp-name">{{ competitionName || 'Kejuaraan Terpilih' }}</span>
            </div>
          </div>

          <!-- Preset Selector -->
          <div class="cat-form-group">
            <label for="cat-preset-select" class="cat-form-label font-mono">
              PILIH TEMPLATE KELAS STANDAR IPSI (OPSIONAL)
            </label>
            <select
              id="cat-preset-select"
              v-model="selectedPresetIndex"
              class="cat-form-select font-mono"
              @change="handlePresetSelect"
            >
              <option value="">— Input Kustom / Manual —</option>
              <optgroup label="Dewasa Putra (Sudut Merah)">
                <option
                  v-for="(preset, idx) in IPSI_STANDARD_PRESETS.filter((p) => p.gender === 'male')"
                  :key="idx"
                  :value="IPSI_STANDARD_PRESETS.indexOf(preset)"
                >
                  {{ preset.label }} - PUTRA
                </option>
              </optgroup>
              <optgroup label="Dewasa Putri (Sudut Biru)">
                <option
                  v-for="(preset, idx) in IPSI_STANDARD_PRESETS.filter(
                    (p) => p.gender === 'female',
                  )"
                  :key="idx"
                  :value="IPSI_STANDARD_PRESETS.indexOf(preset)"
                >
                  {{ preset.label }} - PUTRI
                </option>
              </optgroup>
            </select>
            <span class="cat-form-help"
              >Memilih template akan otomatis mengisi nama kategori dan rentang bobot.</span
            >
          </div>

          <!-- Category Name -->
          <div class="cat-form-group">
            <label for="cat-name-input" class="cat-form-label font-mono">
              NAMA KATEGORI TANDING <span class="cat-req">*</span>
            </label>
            <input
              id="cat-name-input"
              v-model="formState.name"
              type="text"
              class="cat-form-input font-mono"
              :class="{ 'cat-form-input--error': formErrors.name }"
              placeholder="Contoh: Tanding Dewasa Putra - Kelas A (45-50 Kg)"
              required
            />
            <span v-if="formErrors.name" class="cat-form-error">{{ formErrors.name }}</span>
          </div>

          <!-- Gender Radio Options -->
          <div class="cat-form-group">
            <label class="cat-form-label font-mono">
              GOLONGAN & JENIS KELAMIN <span class="cat-req">*</span>
            </label>
            <div class="cat-gender-toggle-group">
              <label
                class="cat-gender-radio"
                :class="{
                  'cat-gender-radio--active-male':
                    formState.gender === 'male' || formState.gender === 'putra',
                }"
              >
                <input v-model="formState.gender" type="radio" value="male" class="sr-only" />
                <span class="cat-radio-indicator cat-radio-indicator--male" />
                <div class="cat-radio-text">
                  <strong>DEWASA PUTRA</strong>
                  <span class="cat-sub font-mono">Sudut Merah</span>
                </div>
              </label>

              <label
                class="cat-gender-radio"
                :class="{
                  'cat-gender-radio--active-female':
                    formState.gender === 'female' || formState.gender === 'putri',
                }"
              >
                <input v-model="formState.gender" type="radio" value="female" class="sr-only" />
                <span class="cat-radio-indicator cat-radio-indicator--female" />
                <div class="cat-radio-text">
                  <strong>DEWASA PUTRI</strong>
                  <span class="cat-sub font-mono">Sudut Biru</span>
                </div>
              </label>
            </div>
            <span v-if="formErrors.gender" class="cat-form-error">{{ formErrors.gender }}</span>
          </div>

          <!-- Weight Class Code -->
          <div class="cat-form-group">
            <label for="cat-weight-class-input" class="cat-form-label font-mono">
              KODE KELAS TANDING <span class="cat-req">*</span>
            </label>
            <input
              id="cat-weight-class-input"
              v-model="formState.weight_class"
              type="text"
              class="cat-form-input font-mono"
              :class="{ 'cat-form-input--error': formErrors.weight_class }"
              placeholder="Contoh: A, B, C, D, OPEN"
              maxlength="10"
              required
            />
            <span v-if="formErrors.weight_class" class="cat-form-error">{{
              formErrors.weight_class
            }}</span>
            <span class="cat-form-help">Kode kelas tanding resmi (A sampai J atau BEBAS).</span>
          </div>

          <footer class="cat-modal__footer font-mono">
            <AppButton type="button" variant="secondary" size="md" @click="isModalOpen = false">
              Batal
            </AppButton>
            <AppButton
              type="submit"
              variant="emerald"
              size="md"
              :loading="createMutation.isPending.value"
            >
              Simpan Kategori
            </AppButton>
          </footer>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cat-ops {
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: 100%;
  color: var(--text-main, #f8fafc);
  box-sizing: border-box;
}

/* Toast */
.cat-toast {
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
.cat-toast--success {
  border-left: 4px solid var(--color-emerald-500, #10b981);
}
.cat-toast--danger {
  border-left: 4px solid var(--color-primary-600, #dc2626);
}
.cat-toast--info {
  border-left: 4px solid var(--color-secondary-600, #2563eb);
}
.cat-toast__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: currentColor;
}
.cat-toast__content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.cat-toast__title {
  font-size: 0.875rem;
  font-weight: 700;
}
.cat-toast__msg {
  font-size: 0.8125rem;
  color: var(--color-neutral-400, #94a3b8);
}
.cat-toast__close {
  background: none;
  border: none;
  color: var(--color-neutral-500, #64748b);
  cursor: pointer;
  font-size: 1rem;
}
.cat-fade-enter-active,
.cat-fade-leave-active {
  transition:
    opacity 0.2s ease,
    transform 0.2s ease;
}
.cat-fade-enter-from,
.cat-fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* Header */
.cat-header {
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
.cat-header__left {
  display: flex;
  align-items: center;
  gap: 16px;
}
.cat-header__icon-box {
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
.cat-header__sub {
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
.cat-bullet {
  color: var(--color-neutral-600, #475569);
}
.cat-tag {
  color: var(--color-primary-400, #f87171);
}
.cat-pulse-wrapper {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--color-emerald-400, #34d399);
}
.cat-pulse-dot {
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
.cat-header__title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.5rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #ffffff;
  line-height: 1.2;
}
.cat-header__actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

/* Metrics Strip */
.cat-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 12px;
}
.cat-metric-card {
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
.cat-metric-card--male {
  border-left: 3px solid var(--color-primary-600, #dc2626);
}
.cat-metric-card--female {
  border-left: 3px solid var(--color-secondary-600, #2563eb);
}
.cat-metric-corner-tag {
  position: absolute;
  top: 10px;
  right: 12px;
  font-size: 0.625rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--color-neutral-500, #64748b);
}
.cat-metric-label {
  font-family: var(--font-mono, monospace);
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--color-neutral-400, #94a3b8);
}
.cat-metric-value-row {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-top: 2px;
}
.cat-metric-val {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.875rem;
  font-weight: 800;
  color: #ffffff;
  line-height: 1;
}
.cat-val-male {
  color: #fca5a5;
}
.cat-val-female {
  color: #93c5fd;
}
.cat-metric-val-text {
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--color-emerald-400, #34d399);
}
.cat-metric-badge {
  font-size: 0.6875rem;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--color-neutral-400, #94a3b8);
}
.cat-badge-male {
  background: rgba(220, 38, 38, 0.15);
  color: #f87171;
}
.cat-badge-female {
  background: rgba(37, 99, 235, 0.15);
  color: #60a5fa;
}
.cat-metric-desc {
  font-size: 0.75rem;
  color: var(--color-neutral-500, #64748b);
  margin-top: 2px;
}

/* Toolbar Controls */
.cat-toolbar {
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
.cat-toolbar__tabs {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.cat-tab-btn {
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
.cat-tab-btn:hover {
  background: rgba(255, 255, 255, 0.05);
  color: #ffffff;
}
.cat-tab-btn--active {
  background: rgba(255, 255, 255, 0.1);
  color: #ffffff;
  border-color: rgba(255, 255, 255, 0.15);
}
.cat-tab-btn--active-male {
  background: rgba(220, 38, 38, 0.15);
  color: #fca5a5;
  border-color: rgba(220, 38, 38, 0.4);
}
.cat-tab-btn--active-female {
  background: rgba(37, 99, 235, 0.15);
  color: #93c5fd;
  border-color: rgba(37, 99, 235, 0.4);
}
.cat-tab-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}
.cat-dot-male {
  background-color: var(--color-primary-500, #ef4444);
}
.cat-dot-female {
  background-color: var(--color-secondary-500, #3b82f6);
}
.cat-tab-badge {
  font-size: 0.6875rem;
  padding: 1px 6px;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.25);
}
.cat-toolbar__search-group {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  flex: 1;
  max-width: 520px;
  justify-content: flex-end;
}
.cat-search-box {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 220px;
}
.cat-search-icon {
  position: absolute;
  left: 10px;
  color: var(--color-neutral-500, #64748b);
  pointer-events: none;
}
.cat-search-input {
  width: 100%;
  padding: 8px 32px 8px 34px;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  color: #ffffff;
  font-size: 0.75rem;
  outline: none;
  transition: border-color 0.15s ease;
}
.cat-search-input:focus {
  border-color: var(--color-emerald-400, #34d399);
}
.cat-search-clear {
  position: absolute;
  right: 10px;
  background: none;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  cursor: pointer;
  font-size: 0.8125rem;
}
.cat-sort-select {
  padding: 8px 12px;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  color: var(--color-neutral-300, #cbd5e1);
  font-size: 0.75rem;
  outline: none;
  cursor: pointer;
}

/* Card Grid with Container Queries */
.cat-grid {
  display: grid;
  grid-template-columns: repeat(1, minmax(0, 1fr));
  gap: 16px;
}
@container (min-width: 640px) {
  .cat-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@container (min-width: 1024px) {
  .cat-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

/* Category Card */
.cat-card {
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
.cat-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.35);
}
.cat-card__accent {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
}
.cat-accent--male {
  background: linear-gradient(90deg, #dc2626, #ef4444);
}
.cat-accent--female {
  background: linear-gradient(90deg, #2563eb, #3b82f6);
}

.cat-card__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 16px 18px 10px 18px;
}
.cat-card__badges {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.cat-badge-gender {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.6875rem;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 4px;
  letter-spacing: 0.05em;
}
.cat-badge-gender--male {
  background: rgba(220, 38, 38, 0.18);
  color: #fca5a5;
  border: 1px solid rgba(220, 38, 38, 0.35);
}
.cat-badge-gender--female {
  background: rgba(37, 99, 235, 0.18);
  color: #93c5fd;
  border: 1px solid rgba(37, 99, 235, 0.35);
}
.cat-badge-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}
.cat-badge-rule {
  font-size: 0.625rem;
  color: var(--color-neutral-500, #64748b);
  letter-spacing: 0.06em;
}
.cat-card__code {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  font-size: 0.625rem;
  color: var(--color-neutral-400, #94a3b8);
}
.cat-card__code strong {
  font-family: var(--font-mono, monospace);
  font-size: 1.5rem;
  font-weight: 800;
  color: #ffffff;
  line-height: 1;
  margin-top: 2px;
}

.cat-card__content {
  padding: 0 18px 14px 18px;
}
.cat-card__title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.05rem;
  font-weight: 700;
  color: #ffffff;
  line-height: 1.35;
  margin-bottom: 12px;
}
.cat-card__spec-box {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 6px;
}
.cat-spec-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.cat-spec-label {
  font-size: 0.5625rem;
  font-weight: 600;
  color: var(--color-neutral-500, #64748b);
  letter-spacing: 0.06em;
}
.cat-spec-value {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--color-neutral-200, #e2e8f0);
}
.cat-spec-divider {
  width: 1px;
  height: 20px;
  background: rgba(255, 255, 255, 0.08);
}

.cat-card__footer {
  padding: 12px 18px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(0, 0, 0, 0.15);
}
.cat-view-bracket-btn {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 8px 12px;
  border-radius: 6px;
  border: 1px solid transparent;
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.15s ease;
}
.cat-view-bracket-btn--male {
  background: rgba(220, 38, 38, 0.12);
  color: #fca5a5;
  border-color: rgba(220, 38, 38, 0.25);
}
.cat-view-bracket-btn--male:hover {
  background: rgba(220, 38, 38, 0.25);
  color: #ffffff;
  border-color: rgba(220, 38, 38, 0.45);
}
.cat-view-bracket-btn--female {
  background: rgba(37, 99, 235, 0.12);
  color: #93c5fd;
  border-color: rgba(37, 99, 235, 0.25);
}
.cat-view-bracket-btn--female:hover {
  background: rgba(37, 99, 235, 0.25);
  color: #ffffff;
  border-color: rgba(37, 99, 235, 0.45);
}

/* Skeleton Loading */
.cat-skeleton-card {
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
  width: 50%;
  height: 20px;
}
.skeleton-title {
  width: 85%;
  height: 28px;
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
.cat-empty-state {
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
.cat-empty-icon {
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
.cat-empty-icon--error {
  background: rgba(220, 38, 38, 0.15);
  color: var(--color-primary-400, #f87171);
}
.cat-empty-title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.25rem;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 8px;
}
.cat-empty-desc {
  max-width: 520px;
  font-size: 0.875rem;
  color: var(--color-neutral-400, #94a3b8);
  line-height: 1.5;
  margin-bottom: 20px;
}
.cat-empty-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

/* Modal Dialog */
.cat-modal-overlay {
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
.cat-modal {
  width: 100%;
  max-width: 560px;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
  overflow: hidden;
}
.cat-modal__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-card, #1e293b);
}
.cat-modal__tag {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--color-emerald-400, #34d399);
  margin-bottom: 4px;
}
.cat-modal__title {
  font-family: var(--font-sans, 'Space Grotesk', sans-serif);
  font-size: 1.25rem;
  font-weight: 800;
  color: #ffffff;
}
.cat-modal__close {
  background: none;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  cursor: pointer;
  font-size: 1.25rem;
}
.cat-modal__body {
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.cat-form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.cat-form-label {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--color-neutral-300, #cbd5e1);
}
.cat-req {
  color: var(--color-primary-500, #ef4444);
}
.cat-form-input,
.cat-form-select {
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
.cat-form-input:focus,
.cat-form-select:focus {
  border-color: var(--color-emerald-400, #34d399);
}
.cat-form-input--error {
  border-color: var(--color-primary-500, #ef4444) !important;
}
.cat-form-error {
  font-size: 0.6875rem;
  color: var(--color-primary-400, #f87171);
}
.cat-form-help {
  font-size: 0.6875rem;
  color: var(--color-neutral-500, #64748b);
}

/* Gender Toggle Radio */
.cat-gender-toggle-group {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.cat-gender-radio {
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
.cat-radio-indicator {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 2px solid var(--color-neutral-600, #475569);
  transition: all 0.15s ease;
}
.cat-radio-indicator--male {
  border-color: #dc2626;
}
.cat-radio-indicator--female {
  border-color: #2563eb;
}
.cat-radio-text {
  display: flex;
  flex-direction: column;
}
.cat-radio-text strong {
  font-size: 0.75rem;
  color: #ffffff;
}
.cat-radio-text .cat-sub {
  font-size: 0.625rem;
  color: var(--color-neutral-400, #94a3b8);
}
.cat-gender-radio--active-male {
  background: rgba(220, 38, 38, 0.15);
  border-color: #dc2626;
}
.cat-gender-radio--active-male .cat-radio-indicator {
  background: #dc2626;
  box-shadow: 0 0 6px #dc2626;
}
.cat-gender-radio--active-female {
  background: rgba(37, 99, 235, 0.15);
  border-color: #2563eb;
}
.cat-gender-radio--active-female .cat-radio-indicator {
  background: #2563eb;
  box-shadow: 0 0 6px #2563eb;
}

.cat-modal__footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 10px;
}

.cat-comp-badge {
  color: #f59e0b;
  font-weight: 700;
}
.cat-target-comp {
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
  color: #f59e0b;
  letter-spacing: 0.06em;
}
.target-comp-name {
  font-size: 0.8125rem;
  font-weight: 700;
  color: #f8fafc;
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
