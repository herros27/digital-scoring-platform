<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { arenaService } from '@/features/arena/api/arenaService'
import type { LiveArena, LiveSchedule } from '@/features/arena/types/arena.schema'
import { MATCH_STATUS_LABEL, matchStatusFromState } from '@/composables/matchStatus'

const REFRESH_MS = 5000

const arenas = ref<LiveArena[]>([])
const schedule = ref<LiveSchedule[]>([])
const loaded = ref(false)
const failed = ref(false)
let timer: ReturnType<typeof setInterval> | undefined

async function refresh() {
  try {
    arenas.value = await arenaService.getLiveArenas()
    failed.value = false
    // Jadwal pelengkap: kegagalannya tidak boleh menyembunyikan daftar pertandingan live.
    schedule.value = await arenaService.getLiveSchedule().catch(() => schedule.value)
  } catch {
    failed.value = true
  } finally {
    loaded.value = true
  }
}

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

/** "10–15 Oktober 2026", atau "10 Oktober 2026" bila sehari. Format tak dikenal dikembalikan apa adanya. */
function dateRange(start: string, end: string): string {
  const [sy, sm, sd] = start.split('-').map(Number)
  const [ey, em, ed] = end.split('-').map(Number)
  if (!sy || !sm || !sd) return ''
  const first = `${sd} ${MONTHS[sm - 1]} ${sy}`
  if (!ey || !em || !ed || (sy === ey && sm === em && sd === ed)) return first
  if (sy === ey && sm === em) return `${sd}–${ed} ${MONTHS[sm - 1]} ${sy}`
  return `${first} – ${ed} ${MONTHS[em - 1]} ${ey}`
}

const headline = computed(() => schedule.value[0])
const hasSchedule = computed(() => schedule.value.some((c) => c.arenas.length))

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
    <div class="sbl__inner">
      <header class="sbl__head">
        <div class="sbl__brand" aria-hidden="true">
          <svg
            width="28"
            height="28"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M12 2L2 7l10 5 10-5-10-5z" />
            <path d="M2 17l10 5 10-5" />
            <path d="M2 12l10 5 10-5" />
          </svg>
        </div>
        <div>
          <p class="sbl__kicker font-mono">PENCAK SILAT DIGITAL SCORING</p>
          <h1>{{ headline ? headline.name : 'SCOREBOARD LIVE' }}</h1>
          <p v-if="headline" class="sbl__meta font-mono">
            <span v-if="headline.location">{{ headline.location }}</span>
            <span v-if="dateRange(headline.start_date, headline.end_date)">
              {{ dateRange(headline.start_date, headline.end_date) }}
            </span>
            <span v-if="schedule.length > 1">+{{ schedule.length - 1 }} kejuaraan lain</span>
          </p>
        </div>
      </header>

      <router-link to="/bagan" class="sbl__bagan font-mono" data-test="link-bagan">
        LIHAT BAGAN PERTANDINGAN →
      </router-link>

      <h2 class="sbl__title font-mono">SEDANG BERLANGSUNG</h2>
      <p class="sbl__hint font-mono">
        Pilih gelanggang untuk membuka papan skor. Diperbarui otomatis.
      </p>

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

      <section v-if="hasSchedule" class="sbl__next" aria-labelledby="sbl-next-title">
        <h2 id="sbl-next-title" class="sbl__title font-mono">JADWAL BERIKUTNYA</h2>
        <template v-for="c in schedule" :key="c.competition_id">
          <p v-if="schedule.length > 1" class="sbl__comp font-mono">{{ c.name }}</p>
          <div class="sbl__next-grid">
            <article v-for="a in c.arenas" :key="a.arena_id" class="sbl-next">
              <h3 class="font-mono">{{ a.arena_name }}</h3>
              <ol v-if="a.upcoming.length">
                <li v-for="m in a.upcoming" :key="m.match_id">
                  <span class="sbl-next__no font-mono">#{{ m.match_number }}</span>
                  <span class="sbl-next__body">
                    <span class="sbl-next__class font-mono">{{ m.class_label }}</span>
                    <span class="sbl-next__fight">
                      <b class="sbl-card__red">{{ m.red_athlete || 'Sudut Merah' }}</b>
                      <i>vs</i>
                      <b class="sbl-card__blue">{{ m.blue_athlete || 'Sudut Biru' }}</b>
                    </span>
                    <span class="sbl-next__team font-mono"
                      >{{ m.red_team }} · {{ m.blue_team }}</span
                    >
                  </span>
                </li>
              </ol>
              <p v-else class="sbl-next__empty font-mono">Tidak ada partai menunggu.</p>
            </article>
          </div>
        </template>
      </section>

      <section class="sbl__staff" aria-labelledby="sbl-staff-title">
        <h2 id="sbl-staff-title" class="font-mono">AKSES PETUGAS</h2>
        <div class="sbl__staff-grid">
          <router-link to="/login" class="sbl-action">
            <strong>Login Petugas</strong>
            <span>Admin, dewan, operator, dan kontingen</span>
            <em aria-hidden="true">Masuk →</em>
          </router-link>
          <router-link to="/judge/login" class="sbl-action">
            <strong>Login Juri / Operator Meja</strong>
            <span>Masuk dengan PIN dari panitia</span>
            <em aria-hidden="true">Masuk →</em>
          </router-link>
          <router-link to="/register" class="sbl-action">
            <strong>Daftar Kontingen</strong>
            <span>Buat akun untuk mendaftarkan atlet</span>
            <em aria-hidden="true">Daftar →</em>
          </router-link>
        </div>
      </section>
    </div>
  </main>
</template>

<style scoped>
.sbl {
  min-height: 100dvh;
  padding: clamp(14px, 3.5vw, 32px) clamp(12px, 3vw, 24px);
  padding-top: max(clamp(14px, 3.5vw, 32px), env(safe-area-inset-top));
  padding-bottom: max(clamp(14px, 3.5vw, 32px), env(safe-area-inset-bottom));
  padding-left: max(clamp(12px, 3vw, 24px), env(safe-area-inset-left));
  padding-right: max(clamp(12px, 3vw, 24px), env(safe-area-inset-right));
  background: #05070d;
  color: #e5e7eb;
  box-sizing: border-box;
}

.sbl__inner {
  max-width: 1200px;
  margin: 0 auto;
}

.sbl__bagan {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  margin: 4px 0 8px;
  padding: 0 16px;
  border: 1px solid #1d4ed8;
  border-radius: 10px;
  background: #0f1a30;
  color: #93c5fd;
  font-size: 0.8125rem;
  text-decoration: none;
}
.sbl__bagan:hover,
.sbl__bagan:focus-visible {
  border-color: #3b82f6;
  color: #fff;
}

.sbl__head {
  display: flex;
  align-items: center;
  gap: clamp(12px, 2.5vw, 18px);
  margin-bottom: clamp(20px, 3vh, 32px);
}

.sbl__brand {
  display: grid;
  place-items: center;
  flex: none;
  width: clamp(44px, 7vw, 56px);
  height: clamp(44px, 7vw, 56px);
  border-radius: var(--radius-md, 14px);
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  box-shadow: 0 4px 14px rgba(37, 99, 235, 0.35);
}

.sbl__kicker {
  margin: 0;
  font-size: clamp(0.625rem, 1.2vw, 0.6875rem);
  letter-spacing: 0.14em;
  color: #64748b;
}

.sbl__head h1 {
  margin: 2px 0 0;
  font-size: clamp(1.25rem, 3.5vw, 1.625rem);
  font-weight: 800;
  line-height: 1.2;
  color: #ffffff;
}

.sbl__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 16px;
  margin: 6px 0 0;
  font-size: clamp(0.75rem, 1.8vw, 0.8125rem);
  color: #94a3b8;
}

.sbl__title {
  margin: 0 0 4px;
  font-size: clamp(0.75rem, 1.8vw, 0.8125rem);
  letter-spacing: 0.12em;
  color: #94a3b8;
}

.sbl__hint {
  margin: 0 0 16px;
  font-size: 12px;
  color: #64748b;
}

.sbl__msg {
  padding: 24px;
  border: 1px dashed #334155;
  border-radius: 12px;
  text-align: center;
  color: #94a3b8;
  font-size: clamp(0.75rem, 2vw, 0.875rem);
}
.sbl__msg--error {
  color: #fca5a5;
  border-color: #7f1d1d;
}

.sbl__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 300px), 1fr));
  gap: clamp(12px, 2vw, 16px);
  padding: 0;
  margin: 0;
  list-style: none;
}

.sbl-card {
  display: flex;
  flex-direction: column;
  padding: clamp(12px, 2vw, 16px);
  border: 1px solid #1e293b;
  border-radius: var(--radius-md, 14px);
  background: #0b1220;
  color: inherit;
  text-decoration: none;
  transition:
    border-color 0.15s,
    background-color 0.15s;
  box-sizing: border-box;
}
.sbl-card:hover,
.sbl-card:focus-visible {
  border-color: #3b82f6;
  background-color: #0f1a30;
}

.sbl-card__top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  font-size: clamp(0.75rem, 1.5vw, 0.8125rem);
  flex-wrap: wrap;
}

.sbl-card__status {
  color: #34d399;
  font-weight: 700;
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
  font-size: clamp(0.8125rem, 1.8vw, 0.9375rem);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sbl-card__blue {
  color: #60a5fa;
  font-weight: 700;
  font-size: clamp(0.8125rem, 1.8vw, 0.9375rem);
  text-align: right;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sbl-card__score {
  font-size: clamp(1.2rem, 2.5vw, 1.375rem);
  font-weight: 800;
  background: rgba(15, 23, 42, 0.8);
  padding: 4px 8px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  white-space: nowrap;
}

.sbl__next {
  margin-top: clamp(24px, 4vh, 40px);
}

.sbl__comp {
  margin: 12px 0 0;
  font-size: 13px;
  color: #e5e7eb;
}

.sbl__next-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 300px), 1fr));
  gap: clamp(12px, 2vw, 16px);
  margin-top: 12px;
}

.sbl-next {
  padding: clamp(12px, 2vw, 16px);
  border: 1px solid #1e293b;
  border-radius: var(--radius-md, 14px);
  background: #0b1220;
  box-sizing: border-box;
}

.sbl-next h3 {
  margin: 0 0 10px;
  font-size: 13px;
  letter-spacing: 0.06em;
  color: #38bdf8;
}

.sbl-next ol {
  display: grid;
  gap: 12px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.sbl-next li {
  display: flex;
  gap: 10px;
}

.sbl-next__no {
  flex: none;
  min-width: 34px;
  color: #64748b;
  font-weight: 700;
  font-size: 0.75rem;
}

.sbl-next__body {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.sbl-next__class {
  font-size: 11px;
  color: #34d399;
}

.sbl-next__fight {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: baseline;
}

.sbl-next__fight i {
  font-style: normal;
  font-size: 12px;
  color: #64748b;
}

.sbl-next__team {
  font-size: 11px;
  color: #64748b;
  word-break: break-word;
}

.sbl-next__empty {
  margin: 0;
  font-size: 12px;
  color: #64748b;
}

.sbl__staff {
  margin-top: clamp(28px, 5vh, 48px);
  padding-top: clamp(16px, 3vh, 24px);
  border-top: 1px solid #1f2937;
}

.sbl__staff h2 {
  margin: 0 0 14px;
  font-size: 13px;
  letter-spacing: 0.12em;
  color: #94a3b8;
}

.sbl__staff-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 220px), 1fr));
  gap: clamp(10px, 2vw, 14px);
}

.sbl-action {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: clamp(14px, 2vw, 16px) clamp(14px, 2.5vw, 18px);
  border: 1px solid #334155;
  border-radius: var(--radius-md, 12px);
  background: #0b1220;
  color: #e5e7eb;
  text-decoration: none;
  transition:
    border-color 0.15s,
    background-color 0.15s;
  box-sizing: border-box;
}

.sbl-action strong {
  font-size: clamp(0.9375rem, 1.8vw, 1rem);
}

.sbl-action span {
  font-size: 0.8125rem;
  color: #94a3b8;
  line-height: 1.4;
}

.sbl-action em {
  margin-top: 6px;
  font-style: normal;
  font-size: 0.8125rem;
  font-weight: 700;
  color: #60a5fa;
}

.sbl-action:hover,
.sbl-action:focus-visible {
  border-color: #3b82f6;
  background: #0f1a30;
}
</style>
