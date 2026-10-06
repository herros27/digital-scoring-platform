<script setup lang="ts">
import { computed } from 'vue'
import type { RefereeConsole } from '../composables/useRefereeConsole'
import { formatClock, SCORE_TYPE_LABEL, TARGET_LABEL } from '../lib/matchLog'

const props = defineProps<{ c: RefereeConsole }>()
defineEmits<{ goto: [tab: string] }>()

/** Konsensus butuh minimal 3 juri online (2 dari 3 sepakat dalam jendela waktu). */
const MIN_JUDGES = 3

const judges = computed(() => props.c.presence?.judges ?? [])
const online = computed(() => judges.value.filter((j) => j.connected))
const missing = computed(() => judges.value.filter((j) => !j.connected).map((j) => j.seat))
const ready = computed(() => online.value.length >= MIN_JUDGES)

function lastInputText(seat: number): string {
  const j = judges.value.find((x) => x.seat === seat)
  const last = j?.last_input
  if (!last) return 'Belum ada input'
  const type = SCORE_TYPE_LABEL[last.score_type] ?? last.score_type
  return `${type} ${TARGET_LABEL[last.target] ?? last.target} • B${last.round} ${formatClock(last.clock_ms)}`
}
</script>

<template>
  <div>
    <div v-if="!c.hasMatch" class="ref-empty ref-section">
      <strong>Belum ada partai di gelanggang ini</strong>
      <span>Status juri dipantau setelah partai dimuat ke gelanggang.</span>
      <button class="ref-btn ref-btn--primary" @click="$emit('goto', 'bagan-antrean')">
        Buka Bagan &amp; Antrean
      </button>
    </div>

    <template v-else>
      <section class="ref-card ref-section">
        <div class="ref-card__head font-mono">
          <span>VERIFIKASI JURI 1-5</span>
          <span :class="ready ? 'ref-seat-state--online' : ''" class="ref-seat-state">
            <span class="ref-dot" :class="{ 'ref-dot--online': ready }" />
            {{ online.length }} dari {{ judges.length }} kursi online
          </span>
        </div>
        <p class="ref-card__sub font-mono" style="margin-bottom: 12px">
          <template v-if="ready && !missing.length"
            >Semua juri online. Siap memulai pertandingan.</template
          >
          <template v-else-if="ready"
            >Juri online cukup untuk konsensus (minimal {{ MIN_JUDGES }}). Belum terhubung: kursi
            {{ missing.join(', ') }}.</template
          >
          <template v-else
            >Menunggu juri: konsensus butuh minimal {{ MIN_JUDGES }} juri online. Belum terhubung:
            kursi {{ missing.join(', ') }}.</template
          >
        </p>

        <div class="ref-seat-grid font-mono">
          <div
            v-for="j in judges"
            :key="j.seat"
            class="ref-seat-card"
            :class="{ 'ref-seat-card--online': j.connected }"
          >
            <div class="ref-seat-card__head">
              <span>JURI {{ j.seat }}</span>
              <span class="ref-seat-state" :class="{ 'ref-seat-state--online': j.connected }">
                <span class="ref-dot" :class="{ 'ref-dot--online': j.connected }" />
                {{ j.connected ? 'ONLINE' : 'OFFLINE' }}
              </span>
            </div>
            <div class="ref-kv">
              <span>Input pada partai ini</span><strong>{{ j.inputs }}</strong>
            </div>
            <div class="ref-kv">
              <span>Input terakhir</span><strong>{{ lastInputText(j.seat) }}</strong>
            </div>
          </div>
        </div>
      </section>

      <section class="ref-card ref-section">
        <div class="ref-card__head font-mono"><span>PERANGKAT TERHUBUNG DI GELANGGANG</span></div>
        <div class="ref-seat-grid font-mono">
          <div class="ref-seat-card">
            <div class="ref-kv">
              <span>Meja wasit</span><strong>{{ c.presence?.referees ?? 0 }}</strong>
            </div>
          </div>
          <div class="ref-seat-card">
            <div class="ref-kv">
              <span>Admin / operator</span><strong>{{ c.presence?.admins ?? 0 }}</strong>
            </div>
          </div>
          <div class="ref-seat-card">
            <div class="ref-kv">
              <span>Layar scoreboard TV</span><strong>{{ c.presence?.scoreboards ?? 0 }}</strong>
            </div>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>
