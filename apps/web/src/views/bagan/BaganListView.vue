<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { AGE_BANDS, ageGroupLabel } from '@/features/categories/lib/ageGroups'
import {
  usePublicBracketListQuery,
  usePublicCompetitionsQuery,
} from '@/features/tournament/api/publicBracketQueries'
import type { PublicBracketSummary } from '@/features/tournament/api/publicBracketService'

type GenderFilter = 'all' | 'male' | 'female'

const competitionsQuery = usePublicCompetitionsQuery()
const competitions = computed(() => competitionsQuery.data.value ?? [])

// Kejuaraan aktif pertama dipilih otomatis; penonton bisa berpindah bila ada lebih dari satu.
const selectedCompId = ref('')
watch(
  competitions,
  (list) => {
    if (!list.some((c) => c.id === selectedCompId.value)) selectedCompId.value = list[0]?.id ?? ''
  },
  { immediate: true },
)
const competition = computed(() => competitions.value.find((c) => c.id === selectedCompId.value))

const bracketsQuery = usePublicBracketListQuery(selectedCompId)
const brackets = computed(() => bracketsQuery.data.value ?? [])

const gender = ref<GenderFilter>('all')
const search = ref('')

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return brackets.value.filter(
    (b) =>
      (gender.value === 'all' || normalizeGender(b.gender) === gender.value) &&
      (!q || b.name.toLowerCase().includes(q)),
  )
})

function normalizeGender(g: string): 'male' | 'female' {
  return g === 'female' || g === 'putri' ? 'female' : 'male'
}

// Urutan golongan umur mengikuti tabel resmi (Pra Usia Dini -> Master B), sisanya di belakang.
const AGE_ORDER = [...AGE_BANDS.ipsi, ...AGE_BANDS.persilat_2026].map((b) => b.code as string)
const groups = computed(() => {
  const byAge = new Map<string, PublicBracketSummary[]>()
  for (const b of filtered.value) {
    const list = byAge.get(b.age_group) ?? []
    list.push(b)
    byAge.set(b.age_group, list)
  }
  const rank = (code: string) => {
    const i = AGE_ORDER.indexOf(code)
    return i === -1 ? AGE_ORDER.length : i
  }
  return [...byAge.entries()]
    .sort(([a], [b]) => rank(a) - rank(b))
    .map(([code, items]) => ({ code, label: ageGroupLabel(code), items }))
})

const MONTHS = [
  'Januari',
  'Februari',
  'Maret',
  'April',
  'Mei',
  'Juni',
  'Juli',
  'Agustus',
  'September',
  'Oktober',
  'November',
  'Desember',
]
function dateRange(start: string, end: string): string {
  const [sy, sm, sd] = start.split('-').map(Number)
  const [ey, em, ed] = end.split('-').map(Number)
  if (!sy || !sm || !sd) return ''
  const first = `${sd} ${MONTHS[sm - 1]} ${sy}`
  if (!ey || !em || !ed || (sy === ey && sm === em && sd === ed)) return first
  if (sy === ey && sm === em) return `${sd}–${ed} ${MONTHS[sm - 1]} ${sy}`
  return `${first} – ${ed} ${MONTHS[em - 1]} ${ey}`
}

const failed = computed(() => competitionsQuery.isError.value || bracketsQuery.isError.value)
const loading = computed(
  () =>
    competitionsQuery.isLoading.value || (!!selectedCompId.value && bracketsQuery.isLoading.value),
)
</script>

<template>
  <main class="bgl">
    <div class="bgl__inner">
      <nav class="bgl__nav font-mono" aria-label="Navigasi publik">
        <router-link to="/scoreboard">← Scoreboard Live</router-link>
      </nav>

      <header class="bgl__head">
        <p class="bgl__kicker font-mono">PENCAK SILAT DIGITAL SCORING</p>
        <h1>BAGAN PERTANDINGAN</h1>
        <p v-if="competition" class="bgl__meta font-mono">
          <span>{{ competition.name }}</span>
          <span v-if="competition.location">{{ competition.location }}</span>
          <span v-if="dateRange(competition.start_date, competition.end_date)">
            {{ dateRange(competition.start_date, competition.end_date) }}
          </span>
        </p>
      </header>

      <p v-if="failed" class="bgl__msg bgl__msg--error font-mono" role="alert">
        Tidak dapat memuat bagan. Mencoba lagi…
      </p>
      <p v-else-if="loading" class="bgl__msg font-mono" aria-busy="true">Memuat bagan…</p>
      <p v-else-if="!competitions.length" class="bgl__msg font-mono" data-test="empty-competition">
        Belum ada kejuaraan yang dibuka. Bagan tampil setelah panitia mengaktifkan kejuaraan.
      </p>

      <template v-else>
        <div v-if="competitions.length > 1" class="bgl__comp">
          <label for="bgl-comp" class="font-mono">KEJUARAAN</label>
          <select id="bgl-comp" v-model="selectedCompId">
            <option v-for="c in competitions" :key="c.id" :value="c.id">{{ c.name }}</option>
          </select>
        </div>

        <p v-if="!brackets.length" class="bgl__msg font-mono" data-test="empty-bracket">
          Bagan belum diumumkan oleh panitia.
        </p>

        <template v-else>
          <div class="bgl__filters">
            <div class="bgl__chips font-mono" role="group" aria-label="Filter jenis kelamin">
              <button
                v-for="opt in [
                  { v: 'all', l: 'SEMUA' },
                  { v: 'male', l: 'PUTRA' },
                  { v: 'female', l: 'PUTRI' },
                ] as Array<{ v: GenderFilter; l: string }>"
                :key="opt.v"
                type="button"
                class="bgl-chip"
                :class="{ 'bgl-chip--active': gender === opt.v }"
                :aria-pressed="gender === opt.v"
                @click="gender = opt.v"
              >
                {{ opt.l }}
              </button>
            </div>
            <input
              v-model="search"
              type="search"
              class="bgl__search font-mono"
              placeholder="Cari kategori…"
              aria-label="Cari kategori"
            />
          </div>

          <p class="bgl__count font-mono">{{ filtered.length }} bagan</p>
          <p v-if="!filtered.length" class="bgl__msg font-mono" data-test="no-match">
            Tidak ada bagan yang cocok dengan filter.
          </p>

          <section v-for="g in groups" :key="g.code" class="bgl__group" :aria-label="g.label">
            <h2 class="bgl__group-title font-mono">{{ g.label.toUpperCase() }}</h2>
            <ul class="bgl__grid">
              <li v-for="b in g.items" :key="b.category_id">
                <router-link
                  :to="`/bagan/${selectedCompId}/${b.category_id}`"
                  class="bgl-card"
                  data-test="bracket-card"
                >
                  <span class="bgl-card__top font-mono">
                    <b :class="normalizeGender(b.gender) === 'female' ? 'is-female' : 'is-male'">
                      {{ normalizeGender(b.gender) === 'female' ? 'PUTRI' : 'PUTRA' }}
                    </b>
                    <span v-if="b.completed" class="bgl-card__done">SELESAI</span>
                  </span>
                  <strong class="bgl-card__name">{{ b.name }}</strong>
                  <span class="bgl-card__meta font-mono">
                    Kelas {{ b.weight_class }} · {{ b.participants }} peserta · {{ b.rounds }} babak
                  </span>
                  <em class="bgl-card__cta" aria-hidden="true">Lihat bagan →</em>
                </router-link>
              </li>
            </ul>
          </section>
        </template>
      </template>
    </div>
  </main>
</template>

<style scoped>
.bgl {
  min-height: 100dvh;
  padding: clamp(14px, 3.5vw, 32px) clamp(12px, 3vw, 24px);
  padding-top: max(clamp(14px, 3.5vw, 32px), env(safe-area-inset-top));
  padding-bottom: max(clamp(14px, 3.5vw, 32px), env(safe-area-inset-bottom));
  background: #05070d;
  color: #e5e7eb;
  box-sizing: border-box;
}
.bgl__inner {
  max-width: 1200px;
  margin: 0 auto;
}
.bgl__nav a {
  color: #93c5fd;
  text-decoration: none;
  font-size: 0.8125rem;
}
.bgl__nav a:hover,
.bgl__nav a:focus-visible {
  text-decoration: underline;
}
.bgl__head {
  margin: 14px 0 18px;
}
.bgl__kicker {
  margin: 0;
  color: #64748b;
  font-size: 0.6875rem;
  letter-spacing: 0.12em;
}
.bgl__head h1 {
  margin: 4px 0 6px;
  font-size: clamp(1.25rem, 4vw, 1.75rem);
  letter-spacing: 0.02em;
}
.bgl__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 14px;
  margin: 0;
  color: #94a3b8;
  font-size: 0.75rem;
}
.bgl__msg {
  margin: 18px 0;
  padding: 24px;
  border: 1px dashed #334155;
  border-radius: 12px;
  text-align: center;
  color: #94a3b8;
  font-size: clamp(0.75rem, 2vw, 0.875rem);
}
.bgl__msg--error {
  color: #fca5a5;
  border-color: #7f1d1d;
}
.bgl__comp {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}
.bgl__comp label {
  color: #64748b;
  font-size: 0.6875rem;
  letter-spacing: 0.1em;
}
.bgl__comp select,
.bgl__search {
  min-height: 44px;
  padding: 0 12px;
  border: 1px solid #1e293b;
  border-radius: 10px;
  background: #0b1220;
  color: inherit;
  font: inherit;
  box-sizing: border-box;
}
.bgl__filters {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}
.bgl__search {
  flex: 1 1 200px;
  max-width: 320px;
}
.bgl__chips {
  display: flex;
  gap: 6px;
}
.bgl-chip {
  min-height: 44px;
  padding: 0 14px;
  border: 1px solid #1e293b;
  border-radius: 999px;
  background: #0b1220;
  color: #94a3b8;
  font: inherit;
  font-size: 0.75rem;
  cursor: pointer;
}
.bgl-chip--active {
  border-color: #3b82f6;
  color: #fff;
  background: #10264d;
}
.bgl__count {
  margin: 14px 0 0;
  color: #64748b;
  font-size: 0.75rem;
}
.bgl__group {
  margin-top: 22px;
}
.bgl__group-title {
  margin: 0 0 10px;
  color: #94a3b8;
  font-size: 0.8125rem;
  letter-spacing: 0.1em;
}
.bgl__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 260px), 1fr));
  gap: clamp(10px, 2vw, 14px);
  padding: 0;
  margin: 0;
  list-style: none;
}
.bgl-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  height: 100%;
  padding: clamp(12px, 2vw, 16px);
  border: 1px solid #1e293b;
  border-radius: var(--radius-md, 14px);
  background: #0b1220;
  color: inherit;
  text-decoration: none;
  box-sizing: border-box;
  transition:
    border-color 0.15s,
    background-color 0.15s;
}
.bgl-card:hover,
.bgl-card:focus-visible {
  border-color: #3b82f6;
  background-color: #0f1a30;
}
.bgl-card__top {
  display: flex;
  justify-content: space-between;
  font-size: 0.6875rem;
  letter-spacing: 0.08em;
}
.bgl-card__top .is-male {
  color: #60a5fa;
}
.bgl-card__top .is-female {
  color: #f472b6;
}
.bgl-card__done {
  color: #34d399;
  font-weight: 700;
}
.bgl-card__name {
  font-size: clamp(0.9375rem, 2.2vw, 1.0625rem);
}
.bgl-card__meta {
  color: #94a3b8;
  font-size: 0.75rem;
}
.bgl-card__cta {
  margin-top: auto;
  padding-top: 6px;
  color: #93c5fd;
  font-size: 0.8125rem;
  font-style: normal;
}
</style>
