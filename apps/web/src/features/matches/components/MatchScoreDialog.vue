<script setup lang="ts">
import { computed, ref } from 'vue'
import { getErrorMessage } from '@/api/error'
import { toLogEntry, formatClock } from '@/features/referee/lib/matchLog'
import { useMatchEventsQuery, useMatchScoresQuery } from '../api/matchQueries'
import type { MatchDTO } from '../api/matchService'
import { summarizeRounds } from '../lib/matchDetail'

const props = defineProps<{
  open: boolean
  competitionId: string
  match: MatchDTO | null
  redName: string
  blueName: string
  arenaName?: string
}>()
const emit = defineEmits<{ (e: 'close'): void }>()

const compId = computed(() => (props.open ? props.competitionId : ''))
const matchId = computed(() => (props.open ? (props.match?.id ?? '') : ''))
const scoresQuery = useMatchScoresQuery(compId, matchId)
const eventsQuery = useMatchEventsQuery(compId, matchId)

const rounds = computed(() => summarizeRounds(scoresQuery.data.value?.rounds ?? []))
const totalRed = computed(() => scoresQuery.data.value?.total_red ?? 0)
const totalBlue = computed(() => scoresQuery.data.value?.total_blue ?? 0)

const showRaw = ref(false)
const entries = computed(() =>
  (eventsQuery.data.value ?? [])
    .map((e) => toLogEntry(e))
    .filter((e): e is NonNullable<typeof e> => !!e && (showRaw.value || e.category !== 'juri')),
)

const winnerCorner = computed<'red' | 'blue' | ''>(() => {
  const m = props.match
  if (!m?.winner_id) return ''
  return m.winner_id === m.athlete_red_id ? 'red' : m.winner_id === m.athlete_blue_id ? 'blue' : ''
})

const STATUS_LABEL: Record<string, string> = {
  idle: 'BELUM DIMULAI',
  ready: 'SIAP TANDING',
  running: 'BERLANGSUNG',
  paused: 'DIJEDA',
  break: 'ISTIRAHAT',
  finished: 'SELESAI',
  cancelled: 'DIBATALKAN',
}
</script>

<template>
  <div
    v-if="open && match"
    class="msd-overlay"
    role="dialog"
    aria-modal="true"
    aria-labelledby="msd-title"
    data-test="score-dialog"
    @click.self="emit('close')"
  >
    <div class="msd-modal">
      <header class="msd-header">
        <div>
          <div class="msd-tag font-mono">
            RINCIAN SKOR • {{ STATUS_LABEL[match.status] ?? match.status.toUpperCase() }}
            <template v-if="arenaName"> • {{ arenaName.toUpperCase() }}</template>
          </div>
          <h3 id="msd-title" class="msd-title">{{ redName }} vs {{ blueName }}</h3>
        </div>
        <button type="button" class="msd-close" aria-label="Tutup" @click="emit('close')">✕</button>
      </header>

      <div class="msd-body">
        <div v-if="scoresQuery.isLoading.value" class="msd-state font-mono">Memuat skor…</div>
        <div v-else-if="scoresQuery.error.value" class="msd-state msd-state--error" role="alert">
          {{ getErrorMessage(scoresQuery.error.value, 'Gagal memuat skor') }}
        </div>
        <template v-else>
          <section class="msd-totals" aria-label="Total skor">
            <div
              class="msd-total msd-total--red"
              :class="{ 'msd-total--win': winnerCorner === 'red' }"
            >
              <span class="msd-corner font-mono">SUDUT MERAH</span>
              <strong class="msd-name">{{ redName }}</strong>
              <span class="msd-score tabular-nums" data-test="total-red">{{ totalRed }}</span>
              <span v-if="winnerCorner === 'red'" class="msd-win font-mono">MENANG</span>
            </div>
            <div
              class="msd-total msd-total--blue"
              :class="{ 'msd-total--win': winnerCorner === 'blue' }"
            >
              <span class="msd-corner font-mono">SUDUT BIRU</span>
              <strong class="msd-name">{{ blueName }}</strong>
              <span class="msd-score tabular-nums" data-test="total-blue">{{ totalBlue }}</span>
              <span v-if="winnerCorner === 'blue'" class="msd-win font-mono">MENANG</span>
            </div>
          </section>

          <section>
            <h4 class="msd-h font-mono">SKOR PER BABAK</h4>
            <p v-if="rounds.length === 0" class="msd-empty" data-test="no-rounds">
              Belum ada skor yang tercatat untuk partai ini.
            </p>
            <table v-else class="msd-table" data-test="score-rounds">
              <thead>
                <tr class="font-mono">
                  <th>BABAK</th>
                  <th>MERAH</th>
                  <th>SANKSI</th>
                  <th>BERSIH</th>
                  <th>BIRU</th>
                  <th>SANKSI</th>
                  <th>BERSIH</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="r in rounds" :key="r.round" data-test="score-round-row">
                  <td class="font-mono">{{ r.round }}</td>
                  <td class="tabular-nums">{{ r.red }}</td>
                  <td class="tabular-nums msd-pen">
                    {{ r.redPenalty ? `-${r.redPenalty}` : '0' }}
                  </td>
                  <td class="tabular-nums msd-net" :class="{ 'msd-net--win': r.winner === 'red' }">
                    {{ r.redNet }}
                  </td>
                  <td class="tabular-nums">{{ r.blue }}</td>
                  <td class="tabular-nums msd-pen">
                    {{ r.bluePenalty ? `-${r.bluePenalty}` : '0' }}
                  </td>
                  <td class="tabular-nums msd-net" :class="{ 'msd-net--win': r.winner === 'blue' }">
                    {{ r.blueNet }}
                  </td>
                </tr>
              </tbody>
            </table>
          </section>

          <section>
            <div class="msd-events-head">
              <h4 class="msd-h font-mono">RIWAYAT KEJADIAN</h4>
              <label class="msd-raw font-mono">
                <input v-model="showRaw" type="checkbox" data-test="show-raw" />
                Tampilkan input mentah juri
              </label>
            </div>
            <p v-if="eventsQuery.isLoading.value" class="msd-empty">Memuat riwayat…</p>
            <p v-else-if="entries.length === 0" class="msd-empty" data-test="no-events">
              Belum ada kejadian yang tercatat.
            </p>
            <ol v-else class="msd-events" data-test="score-events">
              <li
                v-for="e in entries"
                :key="e.id"
                class="msd-event"
                :class="`msd-event--${e.category}`"
              >
                <span class="msd-event-time font-mono"
                  >B{{ e.round || '-' }} {{ formatClock(e.clockMs) }}</span
                >
                <span class="msd-event-title">{{ e.title }}</span>
                <span v-if="e.detail" class="msd-event-detail">{{ e.detail }}</span>
              </li>
            </ol>
          </section>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.msd-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(4px);
}
.msd-modal {
  width: 100%;
  max-width: 720px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
  overflow: hidden;
}
.msd-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  padding: 18px 24px;
  border-bottom: 1px solid var(--border-card, #1e293b);
}
.msd-tag {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--color-emerald-400, #34d399);
  margin-bottom: 4px;
}
.msd-title {
  font-size: 1.125rem;
  font-weight: 800;
  color: #fff;
}
.msd-close {
  background: none;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  cursor: pointer;
  font-size: 1.25rem;
}
.msd-body {
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  overflow-y: auto;
}
.msd-state,
.msd-empty {
  font-size: 0.8125rem;
  color: var(--color-neutral-400, #94a3b8);
}
.msd-state--error {
  color: var(--color-primary-500, #ef4444);
}
.msd-totals {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.msd-total {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 14px 16px;
  border-radius: 10px;
  border: 1px solid var(--border-card, #334155);
  background: rgba(15, 23, 42, 0.6);
}
.msd-total--red {
  border-left: 4px solid #dc2626;
}
.msd-total--blue {
  border-left: 4px solid #2563eb;
}
.msd-total--win {
  background: rgba(16, 185, 129, 0.1);
  border-color: rgba(16, 185, 129, 0.5);
}
.msd-corner {
  font-size: 0.625rem;
  letter-spacing: 0.08em;
  color: var(--color-neutral-400, #94a3b8);
}
.msd-name {
  color: #fff;
}
.msd-score {
  font-size: 2rem;
  font-weight: 800;
  color: #fff;
  line-height: 1.1;
}
.msd-win {
  position: absolute;
  top: 10px;
  right: 12px;
  font-size: 0.625rem;
  font-weight: 700;
  color: var(--color-emerald-400, #34d399);
}
.msd-h {
  font-size: 0.6875rem;
  letter-spacing: 0.08em;
  color: var(--color-neutral-300, #cbd5e1);
  margin-bottom: 8px;
}
.msd-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8125rem;
}
.msd-table th {
  text-align: left;
  padding: 8px 10px;
  font-size: 0.625rem;
  letter-spacing: 0.06em;
  color: var(--color-neutral-400, #94a3b8);
  border-bottom: 1px solid var(--border-card, #334155);
}
.msd-table td {
  padding: 8px 10px;
  border-bottom: 1px solid rgba(51, 65, 85, 0.5);
}
.msd-pen {
  color: #fca5a5;
}
.msd-net {
  font-weight: 700;
  color: #fff;
}
.msd-net--win {
  color: var(--color-emerald-400, #34d399);
}
.msd-events-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}
.msd-raw {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.6875rem;
  color: var(--color-neutral-400, #94a3b8);
}
.msd-events {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 260px;
  overflow-y: auto;
}
.msd-event {
  display: grid;
  grid-template-columns: 90px 1fr;
  gap: 2px 10px;
  padding: 6px 10px;
  border-radius: 6px;
  background: rgba(15, 23, 42, 0.5);
  font-size: 0.8125rem;
}
.msd-event-time {
  grid-row: span 2;
  color: var(--color-neutral-400, #94a3b8);
  font-size: 0.6875rem;
}
.msd-event-title {
  color: #fff;
}
.msd-event-detail {
  color: var(--color-neutral-400, #94a3b8);
  font-size: 0.75rem;
}
.msd-event--poin {
  border-left: 3px solid var(--color-emerald-400, #34d399);
}
.msd-event--sanksi {
  border-left: 3px solid #f59e0b;
}
.msd-event--kontrol {
  border-left: 3px solid #60a5fa;
}
.msd-event--juri {
  border-left: 3px solid #64748b;
}
@media (max-width: 560px) {
  .msd-totals {
    grid-template-columns: 1fr;
  }
}
</style>
