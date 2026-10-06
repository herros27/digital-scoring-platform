<script setup lang="ts">
import { computed, ref } from 'vue'
import type { RefereeConsole } from '../composables/useRefereeConsole'
import { formatClock, TARGET_LABEL, type LogCategory } from '../lib/matchLog'

const props = defineProps<{ c: RefereeConsole }>()
defineEmits<{ goto: [tab: string] }>()

type Filter = 'semua' | LogCategory
const FILTERS: Array<{ key: Filter; label: string }> = [
  { key: 'semua', label: 'Semua' },
  { key: 'poin', label: 'Poin sah' },
  { key: 'sanksi', label: 'Sanksi' },
  { key: 'kontrol', label: 'Kontrol' },
  { key: 'sinyal', label: 'Sinyal arena' },
  { key: 'juri', label: 'Input juri' },
]

// Input mentah juri sangat banyak, jadi disembunyikan dari tampilan "Semua"
const filter = ref<Filter>('semua')

const visible = computed(() => {
  const list = props.c.logEntries.filter((e) =>
    filter.value === 'semua' ? e.category !== 'juri' : e.category === filter.value,
  )
  return [...list].reverse()
})

const countOf = (key: Filter) =>
  props.c.logEntries.filter((e) => (key === 'semua' ? e.category !== 'juri' : e.category === key))
    .length

const rounds = computed(() =>
  props.c.roundScores.slice(0, props.c.totalRounds).map((r, i) => ({
    round: i + 1,
    redTech: r.RedTechnical,
    redPen: r.RedPenalties,
    blueTech: r.BlueTechnical,
    bluePen: r.BluePenalties,
    redNet: r.RedTechnical - r.RedPenalties,
    blueNet: r.BlueTechnical - r.BluePenalties,
  })),
)
</script>

<template>
  <div>
    <div v-if="!c.hasMatch" class="ref-empty ref-section">
      <strong>Belum ada partai di gelanggang ini</strong>
      <span>Log poin dan teknik tersedia setelah partai dimuat ke gelanggang.</span>
      <button class="ref-btn ref-btn--primary" @click="$emit('goto', 'bagan-antrean')">
        Buka Bagan &amp; Antrean
      </button>
    </div>

    <template v-else>
      <section class="ref-two-col ref-section">
        <div class="ref-card">
          <div class="ref-card__head font-mono"><span>SKOR PER BABAK</span></div>
          <table class="ref-table font-mono">
            <thead>
              <tr>
                <th>Babak</th>
                <th class="ref-red">Merah (poin)</th>
                <th class="ref-red">Sanksi</th>
                <th class="ref-blue">Biru (poin)</th>
                <th class="ref-blue">Sanksi</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in rounds" :key="r.round">
                <td>B{{ r.round }}</td>
                <td class="ref-red">{{ r.redTech }}</td>
                <td>{{ r.redPen ? `-${r.redPen}` : 0 }}</td>
                <td class="ref-blue">{{ r.blueTech }}</td>
                <td>{{ r.bluePen ? `-${r.bluePen}` : 0 }}</td>
              </tr>
            </tbody>
            <tfoot>
              <tr>
                <td>TOTAL</td>
                <td class="ref-red" colspan="2">{{ c.redTotal }}</td>
                <td class="ref-blue" colspan="2">{{ c.blueTotal }}</td>
              </tr>
            </tfoot>
          </table>
        </div>

        <div class="ref-card">
          <div class="ref-card__head font-mono"><span>TEKNIK SAH (KONSENSUS JURI)</span></div>
          <div v-for="t in ['red', 'blue'] as const" :key="t" style="margin-bottom: 12px">
            <div class="ref-card__sub font-mono" style="margin-bottom: 6px">
              SUDUT {{ TARGET_LABEL[t] }}
            </div>
            <div class="ref-tech-grid font-mono">
              <div v-for="k in c.techniqueCatalog" :key="k.type" class="ref-tech-cell">
                <strong>{{ c.techniques[t].byType[k.type] ?? 0 }}</strong
                ><span>{{ k.label }}</span>
              </div>
              <div class="ref-tech-cell">
                <strong>{{ c.techniques[t].total }}</strong
                ><span>Total</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="ref-card ref-section">
        <div class="ref-card__head font-mono">
          <span>LOG REAL-TIME EVENT</span>
          <span class="ref-sync-live-tag">SYNC LIVE</span>
        </div>
        <div class="ref-chip-row" style="margin-bottom: 10px">
          <button
            v-for="f in FILTERS"
            :key="f.key"
            class="ref-chip-btn"
            :class="{ 'ref-chip-btn--active': filter === f.key }"
            @click="filter = f.key"
          >
            {{ f.label }} ({{ countOf(f.key) }})
          </button>
        </div>
        <div v-if="!visible.length" class="ref-card__sub font-mono">
          Belum ada kejadian pada filter ini.
        </div>
        <div v-else class="ref-log-list ref-log-list--tall font-mono">
          <div v-for="e in visible" :key="e.id" class="ref-log-card">
            <span class="ref-log-time">{{ formatClock(e.clockMs) }}</span>
            <div class="ref-log-content">
              <strong class="ref-log-title" :class="`ref-log-title--${e.category}`">{{
                e.title
              }}</strong>
              <span class="ref-log-detail"
                >{{ e.detail }} <span class="ref-log-meta">• Babak {{ e.round || '-' }}</span></span
              >
            </div>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>
