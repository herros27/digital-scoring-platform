<script setup lang="ts">
import { getErrorMessage } from '@/api/error'
import { ref, computed } from 'vue'
import { AppButton } from '@/components/ui'
import { useRulesetsQuery, useSaveRulesetMutation } from '@/features/rulesets/api/rulesetQueries'
import { useCompetitionQuery } from '@/features/competitions/api/competitionQueries'
import type { RulesetDTO } from '@/features/rulesets/types'
import {
  useCategoriesQuery,
  useAssignRulesetsMutation,
  useCreateCategoryMutation,
  useDeleteCategoryMutation,
  useResetCategoriesMutation,
  useUpdateCategoryMutation,
  useSeedPresetsMutation,
} from '../api/categoryQueries'
import {
  BRACKET_MODE_LABELS,
  createCategorySchema,
  groupNote,
  isJurusOnly,
  presetsFor,
  type BracketMode,
  type Category,
  type CreateCategoryFormValues,
  type StandardClassPreset,
} from '../types/category.schema'
import {
  AGE_BANDS,
  PRESET_LABELS,
  ageGroupLabel,
  type AgeGroupCode,
  type PresetStandard,
} from '../lib/ageGroups'

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
const resetMutation = useResetCategoriesMutation(activeCompId)
const updateMutation = useUpdateCategoryMutation(activeCompId)
const deleteMutation = useDeleteCategoryMutation(activeCompId)

const categories = computed(() => rawCategories.value || [])

// Satu kejuaraan hanya boleh memakai SATU standar. Bila sudah ada kategori, standar terkunci
// sampai kategori di-reset (server juga menolak pencampuran standar).
const activeStandard = computed<PresetStandard | null>(() => {
  const first = categories.value[0]
  return first ? (first.preset_standard ?? 'ipsi') : null
})
const availableStandards = computed<PresetStandard[]>(() =>
  activeStandard.value ? [activeStandard.value] : ['ipsi', 'persilat_2026'],
)

// ---------- Ruleset (durasi/jumlah babak) per golongan umur ----------
const { data: rawRulesets } = useRulesetsQuery()
const { data: competition } = useCompetitionQuery(activeCompId)
const saveRulesetMutation = useSaveRulesetMutation()
const assignMutation = useAssignRulesetsMutation(activeCompId)
const rulesetList = computed(() => rawRulesets.value ?? [])
const rulesetById = computed(() => new Map(rulesetList.value.map((r) => [r.id, r])))

const fmtMinutes = (sec: number) => String(Math.round((sec / 60) * 10) / 10).replace('.', ',')
const rulesetDuration = (r?: RulesetDTO) =>
  r ? `${r.scoring.total_rounds} babak × ${fmtMinutes(r.scoring.round_duration)} mnt` : ''
const categoryDuration = (c: Category) =>
  c.ruleset_id ? rulesetDuration(rulesetById.value.get(c.ruleset_id)) : ''

/** Ruleset kejuaraan (dipakai kategori yang tidak punya ruleset khusus). */
const competitionRuleset = computed(() => {
  const c = competition.value
  if (!c) return undefined
  return (
    rulesetList.value.find((r) => r.id === c.ruleset_id) ??
    rulesetList.value.find((r) => r.name === c.ruleset_name) ??
    rulesetList.value.find(
      (r) => r.name.split(' ')[0].toLowerCase() === c.ruleset_name?.toLowerCase(),
    )
  )
})

const isRulesetOpen = ref(false)
const MIXED = '__mixed__'
const rulesetChoice = ref<Record<string, string>>({})

/** Golongan umur yang punya kategori, berurutan sesuai standar aktif. */
const rulesetGroups = computed(() => {
  const std = activeStandard.value ?? 'ipsi'
  return AGE_BANDS[std]
    .map((b) => ({
      code: b.code,
      label: b.label,
      count: categories.value.filter((c) => (c.age_group ?? 'dewasa') === b.code).length,
    }))
    .filter((g) => g.count > 0)
})

/** Nilai ruleset saat ini untuk satu golongan ('' = ikut kejuaraan, MIXED = beragam). */
function currentRulesetOf(code: string): string {
  const ids = new Set(
    categories.value
      .filter((c) => (c.age_group ?? 'dewasa') === code)
      .map((c) => c.ruleset_id ?? ''),
  )
  return ids.size === 1 ? [...ids][0] : MIXED
}

function openRulesetDialog() {
  const choice: Record<string, string> = {}
  for (const g of rulesetGroups.value) choice[g.code] = currentRulesetOf(g.code)
  rulesetChoice.value = choice
  isRulesetOpen.value = true
}

const pendingAssignments = computed(() =>
  rulesetGroups.value
    .filter((g) => {
      const v = rulesetChoice.value[g.code]
      return v !== undefined && v !== MIXED && v !== currentRulesetOf(g.code)
    })
    .map((g) => ({ age_group: g.code, ruleset_id: rulesetChoice.value[g.code] || null })),
)

// Pintasan sesuai regulasi: durasi golongan usia muda berbeda dari golongan di atasnya.
const durationPreset = computed(() =>
  (activeStandard.value ?? 'ipsi') === 'persilat_2026'
    ? {
        label: 'Singa / Macan / Pre-Teen: 3 babak × 1,5 menit',
        rounds: 3,
        seconds: 90,
        suffix: 'Usia Muda',
        groups: ['singa', 'macan', 'pre_teen'],
      }
    : {
        label: 'Usia Dini 1 & 2: 2 babak × 1,5 menit',
        rounds: 2,
        seconds: 90,
        suffix: 'Usia Dini',
        groups: ['usia_dini_1', 'usia_dini_2'],
      },
)
const presetApplicableGroups = computed(() =>
  rulesetGroups.value.filter((g) => durationPreset.value.groups.includes(g.code)),
)

async function applyDurationPreset() {
  const base = competitionRuleset.value ?? rulesetList.value.find((r) => r.status === 'active')
  if (!base) {
    showToast('Ruleset Belum Ada', 'Pilih ruleset kejuaraan terlebih dahulu.', 'danger')
    return
  }
  const dp = durationPreset.value
  const name = `${base.name} — ${dp.suffix} (${dp.rounds}×${fmtMinutes(dp.seconds)} mnt)`
  try {
    let target = rulesetList.value.find((r) => r.name === name)
    if (!target) {
      target = await saveRulesetMutation.mutateAsync({
        payload: {
          name,
          version: '1.0',
          status: 'active',
          scoring: { ...base.scoring, total_rounds: dp.rounds, round_duration: dp.seconds },
        },
      })
    }
    const next = { ...rulesetChoice.value }
    for (const g of presetApplicableGroups.value) next[g.code] = target.id
    rulesetChoice.value = next
    showToast(
      'Ruleset Siap',
      `${name} dipilih untuk golongan terkait. Klik Simpan untuk menerapkan.`,
      'info',
    )
  } catch (err) {
    showToast(
      'Gagal Membuat Ruleset',
      getErrorMessage(err, 'Ruleset varian tidak dapat dibuat.'),
      'danger',
    )
  }
}

async function saveRulesetAssignments() {
  if (pendingAssignments.value.length === 0) {
    isRulesetOpen.value = false
    return
  }
  try {
    const res = await assignMutation.mutateAsync(pendingAssignments.value)
    isRulesetOpen.value = false
    showToast('Ruleset Diterapkan', `${res.updated} kategori diperbarui.`, 'success')
  } catch (err) {
    showToast(
      'Gagal Menerapkan',
      getErrorMessage(err, 'Ruleset kategori tidak dapat diubah.'),
      'danger',
    )
  }
}

const isResetOpen = ref(false)
async function handleResetCategories() {
  try {
    const res = await resetMutation.mutateAsync()
    isResetOpen.value = false
    showToast(
      'Kategori Di-reset',
      `${res.deleted} kategori dihapus. Pilih standar PB IPSI atau PERSILAT 2026 untuk memulai ulang.`,
      'success',
    )
  } catch (err) {
    showToast('Reset Ditolak', getErrorMessage(err, 'Kategori tidak dapat di-reset.'), 'danger')
  }
}

// UI Filter & Search State
const activeGenderTab = ref<'all' | 'male' | 'female'>('all')
// Filter golongan umur: "master" menggabungkan Master A & B.
type AgeTab =
  | 'all'
  | 'pra_usia_dini'
  | 'usia_dini_1'
  | 'usia_dini_2'
  | 'pra_remaja'
  | 'remaja'
  | 'dewasa'
  | 'master'
const activeAgeTab = ref<AgeTab>('all')
const AGE_TABS: { key: AgeTab; label: string }[] = [
  { key: 'all', label: 'Semua' },
  { key: 'pra_usia_dini', label: 'Pra Usia Dini' },
  { key: 'usia_dini_1', label: 'Usia Dini 1' },
  { key: 'usia_dini_2', label: 'Usia Dini 2' },
  { key: 'pra_remaja', label: 'Pra-Remaja' },
  { key: 'remaja', label: 'Remaja' },
  { key: 'dewasa', label: 'Dewasa' },
  { key: 'master', label: 'Master' },
]
function matchesAgeTab(c: Category, tab: AgeTab): boolean {
  if (tab === 'all') return true
  const g = c.age_group ?? 'dewasa'
  if (tab === 'master') return g === 'master_a' || g === 'master_b'
  return g === tab
}
function ageTabCount(tab: AgeTab): number {
  return categories.value.filter((c) => matchesAgeTab(c, tab)).length
}

function formatWeightRange(c: Category): string {
  const min = c.min_weight ?? null
  const max = c.max_weight ?? null
  if (min === null && max === null) return 'TANPA BATAS'
  if (min === null) return `≤ ${max} KG`
  if (max === null) return `> ${min} KG`
  return `${min}–${max} KG`
}
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
// Kategori yang sedang diubah; null = mode tambah. Server menolak (409) perubahan bobot/gender/golongan bila sudah
// ada peserta, dan pesannya ditampilkan di dalam dialog.
const editingCategory = ref<Category | null>(null)
const submitError = ref('')
const selectedPresetIndex = ref<number | ''>('')
const formErrors = ref<Record<string, string>>({})
interface CategoryFormState {
  name: string
  gender: 'male' | 'female' | 'putra' | 'putri'
  weight_class: string
  age_group: AgeGroupCode
  preset_standard: PresetStandard
  min_weight: number | null
  max_weight: number | null
  bracket_mode: BracketMode
}
const emptyForm = (): CategoryFormState => ({
  name: '',
  gender: 'male',
  weight_class: 'A',
  age_group: 'dewasa',
  preset_standard: activeStandard.value ?? 'ipsi',
  min_weight: null,
  max_weight: null,
  bracket_mode: 'single_elimination',
})
const formState = ref<CategoryFormState>(emptyForm())

const modalPresets = computed(() => presetsFor(formState.value.preset_standard))
const modalAgeBands = computed(() => AGE_BANDS[formState.value.preset_standard])

function openCreateModal() {
  editingCategory.value = null
  submitError.value = ''
  selectedPresetIndex.value = ''
  formState.value = emptyForm()
  formErrors.value = {}
  isModalOpen.value = true
}

function openEditModal(cat: Category) {
  editingCategory.value = cat
  submitError.value = ''
  selectedPresetIndex.value = ''
  formState.value = {
    name: cat.name,
    gender: cat.gender,
    weight_class: cat.weight_class,
    age_group: cat.age_group ?? 'dewasa',
    preset_standard: cat.preset_standard ?? 'ipsi',
    min_weight: cat.min_weight ?? null,
    max_weight: cat.max_weight ?? null,
    bracket_mode: cat.bracket_mode ?? 'single_elimination',
  }
  formErrors.value = {}
  isModalOpen.value = true
}

// Hapus satu kategori
const deleteTarget = ref<Category | null>(null)
const deleteError = ref('')

function openDeleteDialog(cat: Category) {
  deleteTarget.value = cat
  deleteError.value = ''
}

async function confirmDeleteCategory() {
  if (!deleteTarget.value) return
  deleteError.value = ''
  try {
    await deleteMutation.mutateAsync(deleteTarget.value.id)
    showToast('Kategori Dihapus', `${deleteTarget.value.name} telah dihapus.`, 'success')
    deleteTarget.value = null
  } catch (err) {
    deleteError.value = getErrorMessage(err, 'Kategori tidak dapat dihapus.')
  }
}

// Mengganti standar mereset template & memastikan golongan umur valid untuk standar tsb.
function handleStandardChange() {
  selectedPresetIndex.value = ''
  if (!modalAgeBands.value.some((b) => b.code === formState.value.age_group)) {
    formState.value.age_group = modalAgeBands.value[modalAgeBands.value.length - 1].code
  }
}

function handlePresetSelect(event: Event) {
  const target = event.target as HTMLSelectElement
  const val = target.value
  if (val === '') return

  const idx = Number(val)
  selectedPresetIndex.value = idx
  const preset = modalPresets.value[idx]
  if (preset) {
    formState.value = {
      name: preset.categoryName,
      gender: preset.gender,
      weight_class: preset.code,
      age_group: preset.ageGroup,
      preset_standard: preset.standard,
      min_weight: preset.minWeight,
      max_weight: preset.maxWeight,
      bracket_mode: preset.bracketMode,
    }
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
      age_group: formState.value.age_group,
      preset_standard: formState.value.preset_standard,
      min_weight: formState.value.min_weight,
      max_weight: formState.value.max_weight,
      bracket_mode: formState.value.bracket_mode,
    }

    if (editingCategory.value) {
      submitError.value = ''
      try {
        const updated = await updateMutation.mutateAsync({ id: editingCategory.value.id, payload })
        isModalOpen.value = false
        showToast('Kategori Diperbarui', `${updated.name} berhasil diperbarui.`, 'success')
      } catch (err) {
        submitError.value = getErrorMessage(err, 'Kategori tidak dapat diperbarui.')
      }
      return
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

// Batch Seed Preset (PB IPSI / PERSILAT 2026) — admin memilih golongan umur yang dipertandingkan.
const isSeedOpen = ref(false)
const seedStandard = ref<PresetStandard>('ipsi')
// Pilihan dilacak per kelas (nama kategori unik). Kelas yang sudah ada di kejuaraan
// ditandai "sudah diterapkan" dan tidak bisa dipilih lagi.
const seedSelected = ref<Set<string>>(new Set())
const seedExpanded = ref<Set<AgeGroupCode>>(new Set())

const seedBands = computed(() => AGE_BANDS[seedStandard.value])
const seedAllPresets = computed(() => presetsFor(seedStandard.value))
const existingNames = computed(
  () => new Set(categories.value.map((c) => c.name.toLowerCase().trim())),
)
const isApplied = (p: StandardClassPreset) =>
  existingNames.value.has(p.categoryName.toLowerCase().trim())

function presetsOfGroup(code: AgeGroupCode) {
  return seedAllPresets.value.filter((p) => p.ageGroup === code)
}

interface GroupState {
  total: number
  applied: number
  selectable: number
  selected: number
}
function groupState(code: AgeGroupCode): GroupState {
  const list = presetsOfGroup(code)
  const applied = list.filter(isApplied).length
  const selected = list.filter(
    (p) => !isApplied(p) && seedSelected.value.has(p.categoryName),
  ).length
  return { total: list.length, applied, selectable: list.length - applied, selected }
}

const seedPreview = computed(() =>
  seedAllPresets.value.filter((p) => !isApplied(p) && seedSelected.value.has(p.categoryName)),
)

// Default: semua kelas yang belum diterapkan tercentang.
function resetSeedSelection() {
  seedSelected.value = new Set(
    seedAllPresets.value.filter((p) => !isApplied(p)).map((p) => p.categoryName),
  )
  seedExpanded.value = new Set()
}

function openSeedDialog(standard: PresetStandard = 'ipsi') {
  seedStandard.value = standard
  resetSeedSelection()
  isSeedOpen.value = true
}

function setSeedStandard(standard: PresetStandard) {
  seedStandard.value = standard
  resetSeedSelection()
}

function toggleSeedGroup(code: AgeGroupCode) {
  const st = groupState(code)
  const next = new Set(seedSelected.value)
  const free = presetsOfGroup(code).filter((p) => !isApplied(p))
  if (st.selected === st.selectable) free.forEach((p) => next.delete(p.categoryName))
  else free.forEach((p) => next.add(p.categoryName))
  seedSelected.value = next
}

function toggleSeedClass(name: string) {
  const next = new Set(seedSelected.value)
  if (next.has(name)) next.delete(name)
  else next.add(name)
  seedSelected.value = next
}

function toggleSeedExpand(code: AgeGroupCode) {
  const next = new Set(seedExpanded.value)
  if (next.has(code)) next.delete(code)
  else next.add(code)
  seedExpanded.value = next
}

async function handleSeedPresets() {
  if (seedPreview.value.length === 0) {
    showToast('Pilih Kelas', 'Pilih minimal satu kelas yang dipertandingkan.', 'info')
    return
  }
  const label = PRESET_LABELS[seedStandard.value]
  try {
    const result = await seedMutation.mutateAsync(seedPreview.value)
    isSeedOpen.value = false
    if (result.created > 0) {
      showToast(
        'Sinkronisasi Kategori Berhasil',
        `${result.created} kelas standar ${label} berhasil dibuat (${result.skipped} dilewati karena sudah ada).`,
        'success',
      )
    } else {
      showToast(
        'Kategori Sudah Lengkap',
        `Semua ${result.skipped} kelas standar ${label} yang dipilih sudah terdaftar di kompetisi ini.`,
        'info',
      )
    }
  } catch (err) {
    showToast(
      'Gagal Sinkronisasi',
      getErrorMessage(err, `Gagal memuat standar ${label}.`),
      'danger',
    )
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

  // Filter golongan umur
  list = list.filter((c) => matchesAgeTab(c, activeAgeTab.value))

  // Filter search
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase().trim()
    list = list.filter(
      (c) =>
        c.name.toLowerCase().includes(q) ||
        c.weight_class.toLowerCase().includes(q) ||
        ageGroupLabel(c.age_group, c.preset_standard).toLowerCase().includes(q) ||
        c.gender.toLowerCase().includes(q),
    )
  }

  // Sort
  list.sort((a, b) => {
    if (sortBy.value === 'class') {
      const ageOrder = (c: Category) => {
        const idx = AGE_BANDS.ipsi.findIndex((x) => x.code === (c.age_group ?? 'dewasa'))
        return idx === -1 ? 99 : idx
      }
      if (ageOrder(a) !== ageOrder(b)) return ageOrder(a) - ageOrder(b)
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
            <span>REGULASI PB IPSI TERBARU · PERSILAT 2026</span>
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
          v-for="std in availableStandards"
          :key="std"
          variant="secondary"
          size="sm"
          :loading="seedMutation.isPending.value"
          @click="openSeedDialog(std)"
        >
          Terapkan Standar {{ std === 'ipsi' ? 'PB IPSI' : 'PERSILAT 2026' }}
        </AppButton>

        <AppButton v-if="activeStandard" variant="secondary" size="sm" @click="openRulesetDialog">
          Durasi per Golongan
        </AppButton>

        <AppButton
          v-if="activeStandard"
          variant="secondary"
          size="sm"
          :loading="resetMutation.isPending.value"
          @click="isResetOpen = true"
        >
          Reset Kategori
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
        <span class="cat-metric-label">PUTRA</span>
        <div class="cat-metric-value-row">
          <span class="cat-metric-val cat-val-male tabular-nums">{{ stats.male }}</span>
          <span class="cat-metric-badge cat-badge-male font-mono">KELAS A - J</span>
        </div>
        <span class="cat-metric-desc">Seluruh golongan umur</span>
      </div>

      <div class="cat-metric-card cat-metric-card--female">
        <div class="cat-metric-corner-tag font-mono">SUDUT BIRU</div>
        <span class="cat-metric-label">PUTRI</span>
        <div class="cat-metric-value-row">
          <span class="cat-metric-val cat-val-female tabular-nums">{{ stats.female }}</span>
          <span class="cat-metric-badge cat-badge-female font-mono">KELAS A - F</span>
        </div>
        <span class="cat-metric-desc">Seluruh golongan umur</span>
      </div>

      <div class="cat-metric-card">
        <span class="cat-metric-label">STANDAR ATURAN</span>
        <div class="cat-metric-value-row">
          <span class="cat-metric-val-text font-mono">PB IPSI</span>
        </div>
        <span class="cat-metric-desc">Preset PB IPSI / PERSILAT 2026</span>
      </div>
    </section>

    <!-- Filter Golongan Umur -->
    <nav class="cat-age-tabs font-mono" role="tablist" aria-label="Filter golongan umur">
      <button
        v-for="tab in AGE_TABS"
        :key="tab.key"
        type="button"
        class="cat-age-tab"
        :class="{ 'cat-age-tab--active': activeAgeTab === tab.key }"
        role="tab"
        :aria-selected="activeAgeTab === tab.key"
        @click="activeAgeTab = tab.key"
      >
        {{ tab.label }}
        <span class="cat-tab-badge tabular-nums">{{ ageTabCount(tab.key) }}</span>
      </button>
    </nav>

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
          Putra
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
          Putri
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
            <span class="cat-badge-rule font-mono">{{
              (cat.preset_standard ?? 'ipsi') === 'persilat_2026' ? 'PERSILAT 2026' : 'PB IPSI'
            }}</span>
            <span v-if="categoryDuration(cat)" class="cat-badge-rule font-mono">{{
              categoryDuration(cat)
            }}</span>
            <span v-if="cat.bracket_mode === 'pemasalan_4'" class="cat-badge-rule font-mono"
              >PEMASALAN</span
            >
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
              <span class="cat-spec-label">RENTANG BOBOT</span>
              <span class="cat-spec-value tabular-nums">{{ formatWeightRange(cat) }}</span>
            </div>
            <div class="cat-spec-divider" />
            <div class="cat-spec-item">
              <span class="cat-spec-label">GOLONGAN</span>
              <span class="cat-spec-value"
                >{{ ageGroupLabel(cat.age_group ?? 'dewasa', cat.preset_standard).toUpperCase() }}
                {{ isMaleGender(cat.gender) ? 'PUTRA' : 'PUTRI' }}</span
              >
            </div>
          </div>
        </div>

        <!-- Card Footer: Direct Bracket Action -->
        <footer class="cat-card__footer">
          <div class="cat-card__manage font-mono">
            <button
              type="button"
              class="cat-manage-btn"
              :aria-label="`Ubah ${cat.name}`"
              data-test="category-edit"
              @click="openEditModal(cat)"
            >
              Ubah
            </button>
            <button
              type="button"
              class="cat-manage-btn cat-manage-btn--danger"
              :aria-label="`Hapus ${cat.name}`"
              data-test="category-delete"
              @click="openDeleteDialog(cat)"
            >
              Hapus
            </button>
          </div>
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

    <!-- Ruleset / Durasi per Golongan Dialog -->
    <div
      v-if="isRulesetOpen"
      class="cat-modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="ruleset-title"
      @click.self="isRulesetOpen = false"
    >
      <div class="cat-modal cat-modal--compact">
        <header class="cat-modal__header">
          <div>
            <div class="cat-modal__tag font-mono">ATURAN PERTANDINGAN</div>
            <h3 id="ruleset-title" class="cat-modal__title">Durasi & Babak per Golongan</h3>
          </div>
          <button class="cat-modal__close" aria-label="Tutup dialog" @click="isRulesetOpen = false">
            ✕
          </button>
        </header>
        <div class="cat-modal__body cat-modal__body--compact">
          <p class="cat-form-help">
            Secara default semua kategori mengikuti ruleset kejuaraan
            <strong>{{
              competitionRuleset
                ? `${competitionRuleset.name} (${rulesetDuration(competitionRuleset)})`
                : '—'
            }}</strong
            >. Pilih ruleset lain untuk golongan tertentu, misalnya durasi lebih pendek untuk usia
            dini.
          </p>

          <div v-if="presetApplicableGroups.length" class="cat-ruleset-preset">
            <span class="font-mono">Sesuai regulasi — {{ durationPreset.label }}</span>
            <AppButton
              type="button"
              variant="secondary"
              size="sm"
              :loading="saveRulesetMutation.isPending.value"
              @click="applyDurationPreset"
            >
              Pilih untuk golongan ini
            </AppButton>
          </div>

          <ul class="cat-ruleset-list">
            <li v-for="g in rulesetGroups" :key="g.code" class="cat-ruleset-row">
              <label :for="`rs-${g.code}`" class="cat-ruleset-label">
                <strong>{{ g.label }}</strong>
                <span class="cat-sub font-mono">{{ g.count }} kelas</span>
              </label>
              <select
                :id="`rs-${g.code}`"
                v-model="rulesetChoice[g.code]"
                class="cat-form-select font-mono"
              >
                <option v-if="rulesetChoice[g.code] === MIXED" :value="MIXED" disabled>
                  (beragam)
                </option>
                <option value="">
                  Ikut kejuaraan{{
                    competitionRuleset ? ` — ${rulesetDuration(competitionRuleset)}` : ''
                  }}
                </option>
                <option
                  v-for="r in rulesetList.filter((x) => x.status === 'active')"
                  :key="r.id"
                  :value="r.id"
                >
                  {{ r.name }} — {{ rulesetDuration(r) }}
                </option>
              </select>
            </li>
          </ul>
          <span class="cat-form-help"
            >Tidak dapat diubah untuk golongan yang sudah punya pertandingan berjalan atau
            selesai.</span
          >

          <footer class="cat-modal__footer font-mono">
            <span class="cat-sub">{{ pendingAssignments.length }} golongan diubah</span>
            <AppButton type="button" variant="secondary" size="md" @click="isRulesetOpen = false">
              Batal
            </AppButton>
            <AppButton
              type="button"
              variant="emerald"
              size="md"
              :loading="assignMutation.isPending.value"
              :disabled="pendingAssignments.length === 0"
              @click="saveRulesetAssignments"
            >
              Simpan
            </AppButton>
          </footer>
        </div>
      </div>
    </div>

    <!-- Reset Kategori Dialog -->
    <div
      v-if="isResetOpen"
      class="cat-modal-overlay"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="reset-title"
      @click.self="isResetOpen = false"
    >
      <div class="cat-modal">
        <header class="cat-modal__header">
          <div>
            <div class="cat-modal__tag font-mono">GANTI STANDAR</div>
            <h3 id="reset-title" class="cat-modal__title">Reset Seluruh Kategori?</h3>
          </div>
          <button class="cat-modal__close" aria-label="Tutup dialog" @click="isResetOpen = false">
            ✕
          </button>
        </header>
        <div class="cat-modal__body">
          <p class="cat-form-help">
            Semua <strong>{{ categories.length }}</strong> kategori standar
            <strong>{{ activeStandard ? PRESET_LABELS[activeStandard] : '' }}</strong> pada
            kejuaraan ini akan dihapus beserta bagan yang terkait, lalu Anda dapat memilih standar
            lain (PB IPSI atau PERSILAT 2026).
          </p>
          <p class="cat-form-help">
            Reset ditolak bila masih ada peserta yang terdaftar pada kategori. Pindahkan atau hapus
            peserta tersebut terlebih dahulu.
          </p>
          <footer class="cat-modal__footer font-mono">
            <AppButton type="button" variant="secondary" size="md" @click="isResetOpen = false">
              Batal
            </AppButton>
            <AppButton
              type="button"
              variant="danger"
              size="md"
              :loading="resetMutation.isPending.value"
              @click="handleResetCategories"
            >
              Ya, Reset Kategori
            </AppButton>
          </footer>
        </div>
      </div>
    </div>

    <!-- Seed Standar Dialog -->
    <div
      v-if="isSeedOpen"
      class="cat-modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="seed-title"
      @click.self="isSeedOpen = false"
    >
      <div class="cat-modal cat-modal--compact">
        <header class="cat-modal__header">
          <div>
            <div class="cat-modal__tag font-mono">TEMPLATE KATEGORI</div>
            <h3 id="seed-title" class="cat-modal__title">
              Terapkan Standar {{ PRESET_LABELS[seedStandard] }}
            </h3>
          </div>
          <button class="cat-modal__close" aria-label="Tutup dialog" @click="isSeedOpen = false">
            ✕
          </button>
        </header>

        <div class="cat-modal__body cat-modal__body--compact">
          <div class="cat-seed-standards font-mono" role="tablist" aria-label="Standar regulasi">
            <button
              v-for="std in ['ipsi', 'persilat_2026'] as PresetStandard[]"
              :key="std"
              type="button"
              class="cat-age-tab"
              :class="{ 'cat-age-tab--active': seedStandard === std }"
              role="tab"
              :aria-selected="seedStandard === std"
              @click="setSeedStandard(std)"
            >
              {{ PRESET_LABELS[std] }}
            </button>
          </div>

          <div class="cat-form-group">
            <span class="cat-form-label font-mono">GOLONGAN UMUR & KELAS YANG DIPERTANDINGKAN</span>
            <ul class="cat-seed-list">
              <li
                v-for="band in seedBands"
                :key="band.code"
                class="cat-seed-item"
                :class="{ 'cat-seed-item--done': groupState(band.code).selectable === 0 }"
              >
                <div class="cat-seed-row">
                  <label class="cat-seed-check">
                    <input
                      type="checkbox"
                      :checked="
                        groupState(band.code).selectable > 0 &&
                        groupState(band.code).selected === groupState(band.code).selectable
                      "
                      :indeterminate="
                        groupState(band.code).selected > 0 &&
                        groupState(band.code).selected < groupState(band.code).selectable
                      "
                      :disabled="groupState(band.code).selectable === 0"
                      @change="toggleSeedGroup(band.code)"
                    />
                    <span>
                      <strong>{{ band.label }}</strong>
                      <span class="cat-sub font-mono">{{ band.hint }}</span>
                    </span>
                  </label>
                  <span class="cat-seed-count font-mono">
                    <template v-if="isJurusOnly(seedStandard, band.code)">Hanya Jurus</template>
                    <template v-else-if="groupState(band.code).selectable === 0"
                      >Sudah diterapkan</template
                    >
                    <template v-else
                      >{{ groupState(band.code).selected }}/{{
                        groupState(band.code).selectable
                      }}
                      kelas<template v-if="groupState(band.code).applied > 0">
                        · {{ groupState(band.code).applied }} sudah ada</template
                      ></template
                    >
                  </span>
                  <button
                    type="button"
                    class="cat-seed-toggle"
                    :aria-expanded="seedExpanded.has(band.code)"
                    :aria-label="`Tampilkan kelas ${band.label}`"
                    @click="toggleSeedExpand(band.code)"
                  >
                    {{ seedExpanded.has(band.code) ? '▲' : '▼' }}
                  </button>
                </div>

                <p v-if="groupNote(seedStandard, band.code)" class="cat-seed-note font-mono">
                  {{ groupNote(seedStandard, band.code) }}
                </p>
                <div v-if="seedExpanded.has(band.code)" class="cat-seed-classes">
                  <div v-for="sex in ['male', 'female']" :key="sex" class="cat-seed-col">
                    <span class="cat-seed-col-title font-mono">{{
                      sex === 'male' ? 'PUTRA' : 'PUTRI'
                    }}</span>
                    <label
                      v-for="p in presetsOfGroup(band.code).filter((x) => x.gender === sex)"
                      :key="p.categoryName"
                      class="cat-seed-class"
                      :class="{ 'cat-seed-class--done': isApplied(p) }"
                    >
                      <input
                        type="checkbox"
                        :checked="isApplied(p) || seedSelected.has(p.categoryName)"
                        :disabled="isApplied(p)"
                        @change="toggleSeedClass(p.categoryName)"
                      />
                      <span>{{ p.label }}</span>
                    </label>
                  </div>
                </div>
              </li>
            </ul>
            <span class="cat-form-help"
              >Klik panah untuk memilih kelas satu per satu; default semua kelas tercentang. Kelas
              yang sudah ada di kejuaraan ditandai abu-abu dan dilewati. Golongan tanpa tabel bobot
              resmi dibuat sebagai "Kelas Bebas" dan dapat disesuaikan.</span
            >
          </div>

          <footer class="cat-modal__footer font-mono">
            <span class="cat-sub">{{ seedPreview.length }} kelas akan dibuat</span>
            <AppButton type="button" variant="secondary" size="md" @click="isSeedOpen = false">
              Batal
            </AppButton>
            <AppButton
              type="button"
              variant="emerald"
              size="md"
              :loading="seedMutation.isPending.value"
              :disabled="seedPreview.length === 0"
              @click="handleSeedPresets"
            >
              Terapkan
            </AppButton>
          </footer>
        </div>
      </div>
    </div>

    <!-- Hapus Satu Kategori -->
    <div
      v-if="deleteTarget"
      class="cat-modal-overlay"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="cat-delete-title"
      data-test="category-delete-dialog"
      @click.self="deleteTarget = null"
    >
      <div class="cat-modal cat-modal--compact">
        <header class="cat-modal__header">
          <div>
            <div class="cat-modal__tag font-mono">HAPUS KATEGORI</div>
            <h3 id="cat-delete-title" class="cat-modal__title">Hapus Kategori Ini?</h3>
          </div>
        </header>
        <div class="cat-modal__body cat-modal__body--compact">
          <p class="cat-form-help">
            <strong>{{ deleteTarget.name }}</strong> beserta bagan kosongnya akan dihapus. Kategori
            yang masih memiliki peserta atau partai yang sudah berjalan tidak dapat dihapus.
          </p>
          <p
            v-if="deleteError"
            class="cat-form-error"
            role="alert"
            data-test="category-delete-error"
          >
            {{ deleteError }}
          </p>
          <footer class="cat-modal__footer font-mono">
            <AppButton type="button" variant="secondary" size="md" @click="deleteTarget = null">
              Batal
            </AppButton>
            <AppButton
              type="button"
              variant="danger"
              size="md"
              :loading="deleteMutation.isPending.value"
              data-test="category-delete-confirm"
              @click="confirmDeleteCategory"
            >
              Hapus Kategori
            </AppButton>
          </footer>
        </div>
      </div>
    </div>

    <!-- Create / Edit Category Modal Dialog -->
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
            <div class="cat-modal__tag font-mono">
              {{ editingCategory ? 'UBAH KATEGORI' : 'FORM KATEGORI BARU' }}
            </div>
            <h3 id="modal-title" class="cat-modal__title">
              {{ editingCategory ? 'Ubah Kategori Tanding' : 'Tambah Kategori Tanding' }}
            </h3>
          </div>
          <button class="cat-modal__close" aria-label="Tutup dialog" @click="isModalOpen = false">
            ✕
          </button>
        </header>

        <form
          class="cat-modal__body"
          data-test="category-form"
          @submit.prevent="handleSubmitCategory"
        >
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

          <!-- Standar & Preset Selector -->
          <div class="cat-form-group">
            <label for="cat-standard-select" class="cat-form-label font-mono">
              STANDAR REGULASI
            </label>
            <select
              id="cat-standard-select"
              v-model="formState.preset_standard"
              class="cat-form-select font-mono"
              :disabled="activeStandard !== null"
              @change="handleStandardChange"
            >
              <option value="ipsi">{{ PRESET_LABELS.ipsi }} (Default)</option>
              <option value="persilat_2026">{{ PRESET_LABELS.persilat_2026 }}</option>
            </select>
            <span v-if="activeStandard" class="cat-form-help"
              >Kejuaraan ini memakai standar {{ PRESET_LABELS[activeStandard] }}. Gunakan "Reset
              Kategori" untuk beralih standar.</span
            >
          </div>

          <div v-if="!editingCategory" class="cat-form-group">
            <label for="cat-preset-select" class="cat-form-label font-mono">
              PILIH TEMPLATE KELAS STANDAR (OPSIONAL)
            </label>
            <select
              id="cat-preset-select"
              v-model="selectedPresetIndex"
              class="cat-form-select font-mono"
              @change="handlePresetSelect"
            >
              <option value="">— Input Kustom / Manual —</option>
              <optgroup
                v-for="band in modalAgeBands"
                :key="band.code"
                :label="`${band.label} (${band.hint})`"
              >
                <template v-for="(preset, idx) in modalPresets" :key="idx">
                  <option v-if="preset.ageGroup === band.code" :value="idx">
                    {{ preset.gender === 'male' ? 'Putra' : 'Putri' }} · {{ preset.label }}
                  </option>
                </template>
              </optgroup>
            </select>
            <span class="cat-form-help"
              >Memilih template akan otomatis mengisi nama, golongan umur, rentang bobot, dan mode
              bagan.</span
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
              JENIS KELAMIN <span class="cat-req">*</span>
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
                  <strong>PUTRA</strong>
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
                  <strong>PUTRI</strong>
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

          <!-- Golongan Umur -->
          <div class="cat-form-group">
            <label for="cat-age-group-select" class="cat-form-label font-mono">
              GOLONGAN UMUR <span class="cat-req">*</span>
            </label>
            <select
              id="cat-age-group-select"
              v-model="formState.age_group"
              class="cat-form-select font-mono"
            >
              <option v-for="band in modalAgeBands" :key="band.code" :value="band.code">
                {{ band.label }} — {{ band.hint }}
              </option>
            </select>
            <span v-if="formErrors.age_group" class="cat-form-error">{{
              formErrors.age_group
            }}</span>
          </div>

          <!-- Rentang Bobot -->
          <div class="cat-form-row">
            <div class="cat-form-group">
              <label for="cat-min-weight" class="cat-form-label font-mono">BOBOT MIN (KG)</label>
              <input
                id="cat-min-weight"
                v-model.number="formState.min_weight"
                type="number"
                min="0"
                step="0.1"
                class="cat-form-input font-mono"
                placeholder="Kosong = tanpa batas"
              />
            </div>
            <div class="cat-form-group">
              <label for="cat-max-weight" class="cat-form-label font-mono">BOBOT MAKS (KG)</label>
              <input
                id="cat-max-weight"
                v-model.number="formState.max_weight"
                type="number"
                min="0"
                step="0.1"
                class="cat-form-input font-mono"
                :class="{ 'cat-form-input--error': formErrors.max_weight }"
                placeholder="Kosong = tanpa batas"
              />
              <span v-if="formErrors.max_weight" class="cat-form-error">{{
                formErrors.max_weight
              }}</span>
            </div>
          </div>
          <span class="cat-form-help"
            >Penimbangan resmi tanpa toleransi (0 kg): kelebihan/kekurangan 0,1 kg berakibat
            diskualifikasi. Kosongkan kedua kolom untuk kelas tanpa batas bobot.</span
          >

          <!-- Mode Bagan -->
          <div class="cat-form-group">
            <label for="cat-bracket-mode" class="cat-form-label font-mono">MODE BAGAN</label>
            <select
              id="cat-bracket-mode"
              v-model="formState.bracket_mode"
              class="cat-form-select font-mono"
            >
              <option v-for="(label, key) in BRACKET_MODE_LABELS" :key="key" :value="key">
                {{ label }}
              </option>
            </select>
            <span class="cat-form-help"
              >Pemasalan: peserta dibagi ke mini bagan maks. 4 orang (semifinal + final) untuk Pra
              Usia Dini / Usia Dini 1.</span
            >
          </div>

          <p v-if="submitError" class="cat-form-error" role="alert" data-test="category-form-error">
            {{ submitError }}
          </p>
          <p v-if="editingCategory" class="cat-form-help">
            Bila kategori sudah memiliki peserta, hanya nama yang dapat diubah. Standar regulasi
            tidak dapat diubah (gunakan "Reset Kategori").
          </p>

          <footer class="cat-modal__footer font-mono">
            <AppButton type="button" variant="secondary" size="md" @click="isModalOpen = false">
              Batal
            </AppButton>
            <AppButton
              type="submit"
              variant="emerald"
              size="md"
              :loading="createMutation.isPending.value || updateMutation.isPending.value"
              data-test="category-save"
            >
              {{ editingCategory ? 'Simpan Perubahan' : 'Simpan Kategori' }}
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
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}
@media (max-width: 640px) {
  .cat-grid {
    grid-template-columns: 1fr;
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
.cat-card__manage {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}
.cat-manage-btn {
  flex: 1;
  padding: 6px 10px;
  border-radius: 6px;
  border: 1px solid var(--border-card, #334155);
  background: rgba(15, 23, 42, 0.6);
  color: var(--color-neutral-200, #e2e8f0);
  font-size: 0.6875rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.15s ease;
}
.cat-manage-btn:hover {
  background: rgba(51, 65, 85, 0.7);
  color: #fff;
}
.cat-manage-btn--danger {
  color: #fca5a5;
  border-color: rgba(220, 38, 38, 0.35);
}
.cat-manage-btn--danger:hover {
  background: rgba(220, 38, 38, 0.25);
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
  /* Tinggi dibatasi viewport; header tetap, isi form yang di-scroll. */
  display: flex;
  flex-direction: column;
  max-height: calc(100dvh - 32px);
}
.cat-modal__header {
  flex-shrink: 0;
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
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
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
.cat-age-tabs,
.cat-seed-standards {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin: 0 0 0.75rem;
}
.cat-age-tab {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.4rem 0.8rem;
  font-size: 0.75rem;
  border-radius: 999px;
  border: 1px solid var(--color-border, rgba(255, 255, 255, 0.12));
  background: transparent;
  color: inherit;
  cursor: pointer;
}
.cat-age-tab:hover {
  border-color: currentColor;
}
.cat-age-tab--active {
  background: rgba(16, 185, 129, 0.16);
  border-color: rgb(16, 185, 129);
}
.cat-form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}
.cat-seed-check {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.35rem 0;
  cursor: pointer;
}
.cat-modal--compact {
  max-width: 480px;
  font-size: 0.8125rem;
}
.cat-modal--compact .cat-modal__header {
  padding: 12px 18px;
}
.cat-modal--compact .cat-modal__title {
  font-size: 1rem;
}
.cat-modal__body--compact {
  padding: 14px 18px;
  gap: 12px;
}
.cat-seed-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border-card, #334155);
  border-radius: 8px;
  overflow: hidden;
}
.cat-seed-item + .cat-seed-item {
  border-top: 1px solid var(--border-card, #334155);
}
.cat-seed-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 10px;
}
.cat-seed-row .cat-seed-check {
  flex: 1 1 auto;
  min-width: 0;
}
.cat-seed-count {
  font-size: 0.68rem;
  opacity: 0.7;
  white-space: nowrap;
}
.cat-seed-toggle {
  background: none;
  border: 0;
  color: inherit;
  font-size: 0.7rem;
  padding: 4px 6px;
  cursor: pointer;
  opacity: 0.8;
}
.cat-seed-item--done {
  opacity: 0.5;
}
.cat-seed-classes {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px 14px;
  padding: 6px 12px 10px 34px;
  max-height: 220px;
  overflow-y: auto;
  background: rgba(255, 255, 255, 0.02);
}
@media (max-width: 520px) {
  .cat-seed-classes {
    grid-template-columns: 1fr;
  }
}
.cat-seed-col {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.cat-seed-col-title {
  font-size: 0.62rem;
  letter-spacing: 0.06em;
  opacity: 0.6;
  margin-bottom: 2px;
}
.cat-seed-class {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.72rem;
  padding: 2px 0;
  cursor: pointer;
}
.cat-seed-class--done {
  opacity: 0.45;
  cursor: not-allowed;
}
.cat-modal--compact .cat-seed-check {
  padding: 4px 0;
  gap: 8px;
  font-size: 0.78rem;
  line-height: 1.25;
}
.cat-modal--compact .cat-seed-check .cat-sub {
  display: block;
  font-size: 0.68rem;
  opacity: 0.65;
}
.cat-modal--compact .cat-age-tab {
  padding: 0.3rem 0.65rem;
  font-size: 0.7rem;
}
.cat-modal--compact .cat-form-label {
  font-size: 0.65rem;
}
.cat-modal--compact .cat-form-help {
  font-size: 0.68rem;
}
.cat-modal--compact .cat-modal__footer {
  gap: 8px;
  align-items: center;
}
.cat-modal--compact .cat-modal__footer > .cat-sub {
  margin-right: auto;
  font-size: 0.72rem;
}
.cat-seed-note {
  margin: 0;
  padding: 0 12px 8px 34px;
  font-size: 0.66rem;
  opacity: 0.65;
}
.cat-ruleset-preset {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border: 1px dashed var(--border-card, #334155);
  border-radius: 8px;
  font-size: 0.72rem;
}
.cat-ruleset-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.cat-ruleset-row {
  display: grid;
  grid-template-columns: 120px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
}
.cat-ruleset-label {
  display: flex;
  flex-direction: column;
  font-size: 0.78rem;
}
@media (max-width: 520px) {
  .cat-ruleset-row {
    grid-template-columns: 1fr;
  }
}
</style>
