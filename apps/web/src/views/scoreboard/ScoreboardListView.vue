<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { arenaService } from '@/features/arena/api/arenaService'
import type { LiveArena } from '@/features/arena/types/arena.schema'
import { MATCH_STATUS_LABEL, matchStatusFromState } from '@/composables/matchStatus'

const REFRESH_MS = 5000

const arenas = ref<LiveArena[]>([])
const loaded = ref(false)
const failed = ref(false)
let timer: ReturnType<typeof setInterval> | undefined

async function refresh() {
  try {
    arenas.value = await arenaService.getLiveArenas()
    failed.value = false
  } catch {
    failed.value = true
  } finally {
    loaded.value = true
  }
}

function statusLabel(a: LiveArena): string {
  const status = matchStatusFromState(a.state)
  return status ? MATCH_STATUS_LABEL[status] : a.state.toUpperCase()
}

onMounted(() => {
  void refresh()
  timer = setInterval(refresh, REFRESH_MS)
})
onBeforeUnmount(() => clearInterval(timer))
</script>

<template>
  <main class="sbl">
    <header class="sbl__head">
      <h1>SCOREBOARD LIVE</h1>
      <p class="font-mono">
        Pilih gelanggang untuk membuka papan skor. Daftar diperbarui otomatis.
      </p>
    </header>

    <p v-if="failed" class="sbl__msg sbl__msg--error font-mono">
      Tidak dapat terhubung ke server. Mencoba lagi…
    </p>
    <p v-else-if="loaded && !arenas.length" class="sbl__msg font-mono">
      Belum ada pertandingan yang sedang berlangsung.
    </p>

    <ul v-if="arenas.length" class="sbl__grid">
      <li v-for="a in arenas" :key="a.arena_id">
        <router-link :to="`/scoreboard/${a.arena_id}`" class="sbl-card">
          <div class="sbl-card__top font-mono">
            <strong>{{ a.arena_name || 'GELANGGANG' }}</strong>
            <span class="sbl-card__status"> BABAK {{ a.round }} · {{ statusLabel(a) }} </span>
          </div>
          <div class="sbl-card__fight">
            <span class="sbl-card__red">{{ a.red_athlete || 'Sudut Merah' }}</span>
            <span class="sbl-card__score font-mono">{{ a.red_total }} : {{ a.blue_total }}</span>
            <span class="sbl-card__blue">{{ a.blue_athlete || 'Sudut Biru' }}</span>
          </div>
        </router-link>
      </li>
    </ul>
  </main>
</template>

<style scoped>
.sbl {
  min-height: 100vh;
  padding: 32px 24px;
  background: #05070d;
  color: #e5e7eb;
}
.sbl__head h1 {
  margin: 0;
  font-size: 28px;
  letter-spacing: 0.08em;
}
.sbl__head p {
  margin: 6px 0 24px;
  font-size: 13px;
  color: #94a3b8;
}
.sbl__msg {
  padding: 24px;
  border: 1px dashed #334155;
  border-radius: 12px;
  text-align: center;
  color: #94a3b8;
}
.sbl__msg--error {
  color: #fca5a5;
  border-color: #7f1d1d;
}
.sbl__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
  padding: 0;
  margin: 0;
  list-style: none;
}
.sbl-card {
  display: block;
  padding: 16px;
  border: 1px solid #1e293b;
  border-radius: 14px;
  background: #0b1220;
  color: inherit;
  text-decoration: none;
  transition: border-color 0.15s;
}
.sbl-card:hover,
.sbl-card:focus-visible {
  border-color: #3b82f6;
}
.sbl-card__top {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 13px;
}
.sbl-card__status {
  color: #34d399;
}
.sbl-card__fight {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 10px;
  margin-top: 14px;
}
.sbl-card__red {
  color: #f87171;
  font-weight: 700;
}
.sbl-card__blue {
  color: #60a5fa;
  font-weight: 700;
  text-align: right;
}
.sbl-card__score {
  font-size: 22px;
  font-weight: 800;
}
</style>
