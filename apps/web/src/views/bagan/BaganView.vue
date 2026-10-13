<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { isNotFoundError } from '@/api/error'
import { ageGroupLabel } from '@/features/categories/lib/ageGroups'
import TournamentTreeBracket from '@/features/tournament/components/TournamentTreeBracket.vue'
import { usePublicBracketQuery } from '@/features/tournament/api/publicBracketQueries'
import { buildPublicBracketRounds } from '@/features/tournament/lib/publicBracketRounds'

const route = useRoute()
const compId = computed(() => String(route.params.comp_id ?? ''))
const categoryId = computed(() => String(route.params.category_id ?? ''))

const query = usePublicBracketQuery(compId, categoryId)
const detail = computed(() => query.data.value)
const rounds = computed(() => buildPublicBracketRounds(detail.value?.slots ?? []))

// Juara = pemenang di slot babak final (slot tanpa next_slot_id).
const champion = computed(() => {
  const slots = detail.value?.slots ?? []
  return slots.find((s) => s.next_slot_id === null && s.is_winner && s.participant)?.participant
})

const notFound = computed(() => isNotFoundError(query.error.value))
const failed = computed(() => query.isError.value && !notFound.value)
</script>

<template>
  <main class="bgv">
    <div class="bgv__inner">
      <nav class="bgv__nav font-mono" aria-label="Navigasi publik">
        <router-link to="/bagan" data-test="back-to-list">← Semua Bagan</router-link>
        <router-link to="/scoreboard">Scoreboard Live</router-link>
      </nav>

      <p v-if="query.isLoading.value" class="bgv__msg font-mono" aria-busy="true">Memuat bagan…</p>
      <p v-else-if="notFound" class="bgv__msg font-mono" data-test="not-found" role="alert">
        Bagan tidak ditemukan atau belum dibuka untuk umum.
      </p>
      <p v-else-if="failed" class="bgv__msg bgv__msg--error font-mono" role="alert">
        Tidak dapat memuat bagan. Mencoba lagi…
      </p>

      <template v-else-if="detail">
        <header class="bgv__head">
          <p class="bgv__kicker font-mono">{{ detail.competition.name }}</p>
          <h1 data-test="category-name">{{ detail.category.name }}</h1>
          <p class="bgv__meta font-mono">
            <span>{{ ageGroupLabel(detail.category.age_group) }}</span>
            <span>Kelas {{ detail.category.weight_class }}</span>
            <span>{{ detail.category.participants }} peserta</span>
          </p>
          <p v-if="champion" class="bgv__champion font-mono" data-test="champion">
            JUARA: <b>{{ champion.name }}</b> <span>· {{ champion.team }}</span>
          </p>
        </header>

        <TournamentTreeBracket
          v-if="rounds.length"
          :rounds="rounds"
          :category-name="detail.category.name"
          read-only
        />
        <p v-else class="bgv__msg font-mono">Bagan belum memiliki pertandingan.</p>
      </template>
    </div>
  </main>
</template>

<style scoped>
.bgv {
  min-height: 100dvh;
  padding: clamp(14px, 3.5vw, 32px) clamp(12px, 3vw, 24px);
  padding-top: max(clamp(14px, 3.5vw, 32px), env(safe-area-inset-top));
  padding-bottom: max(clamp(14px, 3.5vw, 32px), env(safe-area-inset-bottom));
  background: #05070d;
  color: #e5e7eb;
  box-sizing: border-box;
}
.bgv__inner {
  max-width: 1400px;
  margin: 0 auto;
}
.bgv__nav {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 0.8125rem;
}
.bgv__nav a {
  color: #93c5fd;
  text-decoration: none;
}
.bgv__nav a:hover,
.bgv__nav a:focus-visible {
  text-decoration: underline;
}
.bgv__head {
  margin: 14px 0 16px;
}
.bgv__kicker {
  margin: 0;
  color: #64748b;
  font-size: 0.6875rem;
  letter-spacing: 0.1em;
}
.bgv__head h1 {
  margin: 4px 0 6px;
  font-size: clamp(1.25rem, 4vw, 1.75rem);
}
.bgv__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 14px;
  margin: 0;
  color: #94a3b8;
  font-size: 0.75rem;
}
.bgv__champion {
  display: inline-block;
  margin: 12px 0 0;
  padding: 8px 14px;
  border: 1px solid #065f46;
  border-radius: 10px;
  background: #052e22;
  color: #6ee7b7;
  font-size: 0.8125rem;
}
.bgv__champion b {
  color: #fff;
}
.bgv__msg {
  margin: 24px 0;
  padding: 24px;
  border: 1px dashed #334155;
  border-radius: 12px;
  text-align: center;
  color: #94a3b8;
  font-size: clamp(0.75rem, 2vw, 0.875rem);
}
.bgv__msg--error {
  color: #fca5a5;
  border-color: #7f1d1d;
}
</style>
