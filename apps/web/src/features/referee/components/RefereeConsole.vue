<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useRefereeConsole } from '../composables/useRefereeConsole'
import ControlPanel from './ControlPanel.vue'
import JuryPanel from './JuryPanel.vue'
import SanctionsPanel from './SanctionsPanel.vue'
import LogPanel from './LogPanel.vue'
import QueuePanel from './QueuePanel.vue'
import { finishRequest, isConfirmRequest, type default as ConfirmRequest } from './confirmRequest'
import '../referee.css'

const props = defineProps<{ arenaId: string; compId: string }>()

const router = useRouter()
const authStore = useAuthStore()
const c = useRefereeConsole({
  arenaId: props.arenaId,
  compId: props.compId,
  token: authStore.token || undefined,
})

type TabKey =
  'kontrol-tanding' | 'verifikasi-juri' | 'sanksi-disiplin' | 'log-poin' | 'bagan-antrean'
const TABS: Array<{ key: TabKey; label: string }> = [
  { key: 'kontrol-tanding', label: 'Kontrol Tanding' },
  { key: 'verifikasi-juri', label: 'Verifikasi Juri 1-5' },
  { key: 'sanksi-disiplin', label: 'Sanksi & Disiplin' },
  { key: 'log-poin', label: 'Log Poin & Teknik' },
  { key: 'bagan-antrean', label: 'Bagan & Antrean' },
]
const activeTab = ref<TabKey>('kontrol-tanding')
const isTab = (v: string): v is TabKey => TABS.some((t) => t.key === v)
function goto(tab: string) {
  if (isTab(tab)) activeTab.value = tab
}

const ROLE_LABEL: Record<string, string> = {
  wasit: 'WASIT',
  referee: 'WASIT',
  admin: 'ADMIN',
  dewan: 'DEWAN HAKIM',
}
const roleLabel = computed(() => ROLE_LABEL[authStore.role] ?? authStore.role.toUpperCase())

const judgesOnline = computed(
  () => c.presence?.judges.filter((j) => j.connected && j.seat <= c.judgeCount).length ?? 0,
)
const judgeSeats = computed(() => c.judgeCount)
const tvOnline = computed(() => c.presence?.scoreboards ?? 0)
const arenaTitle = computed(() => (c.arenaName || 'GELANGGANG').toUpperCase())
const latencyText = computed(() => (c.latencyMs === null ? '-' : `${c.latencyMs} ms`))

// --- dialog konfirmasi ---
const confirmRequest = ref<ConfirmRequest | null>(null)
const confirmChoice = ref('')
const confirmMethod = ref('')
function askConfirm(request: ConfirmRequest) {
  confirmChoice.value = ''
  confirmMethod.value = ''
  confirmRequest.value = request
}
const confirmReady = computed(() => {
  const r = confirmRequest.value
  return !!r && (!r.choices || !!confirmChoice.value) && (!r.methods || !!confirmMethod.value)
})
async function runConfirmed() {
  const request = confirmRequest.value
  if (!request || !confirmReady.value) return
  confirmRequest.value = null
  const next = await request.action(
    request.choices ? confirmChoice.value : undefined,
    request.methods ? confirmMethod.value : undefined,
  )
  if (isConfirmRequest(next)) askConfirm(next)
}

function askLogout() {
  askConfirm({
    title: 'Keluar dari Konsol Wasit?',
    message:
      'Sesi meja pengawas akan diakhiri. Pertandingan yang berjalan tetap berjalan di server.',
    confirmLabel: 'Keluar',
    action: async () => {
      await authStore.logout()
      await router.push('/login')
    },
  })
}

// --- pintasan keyboard ---
function onKeyDown(e: KeyboardEvent) {
  if (['INPUT', 'TEXTAREA', 'SELECT'].includes((e.target as HTMLElement | null)?.tagName ?? ''))
    return
  if (confirmRequest.value) {
    if (e.key === 'Escape') confirmRequest.value = null
    return
  }
  if (e.code === 'Space') {
    e.preventDefault()
    if (c.canStart) void c.start()
    else if (c.canPause) void c.pause()
    else if (c.canResume) void c.resume()
  } else if (e.key === 'p' || e.key === 'P') {
    if (c.canPause) void c.pause()
  } else if (e.key === 'l' || e.key === 'L') {
    if (c.canResume) void c.resume()
  } else if (e.key === 'g' || e.key === 'G') {
    if (c.canSignal) c.ringGong()
  } else if (e.key === 'Escape') {
    if (c.canFinish) askConfirm(finishRequest(c))
  }
}
onMounted(() => window.addEventListener('keydown', onKeyDown))
onUnmounted(() => window.removeEventListener('keydown', onKeyDown))
</script>

<template>
  <div class="ref-app">
    <aside class="ref-sidebar">
      <div class="ref-sidebar__brand">
        <div class="ref-logo-badge">
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="#DC2626"
            stroke-width="2.5"
          >
            <path d="M18 6 6 18" />
            <path d="m6 6 12 12" />
          </svg>
        </div>
        <div>
          <span class="ref-brand-sub font-mono">PERSILAT ARENA</span>
          <h2 class="ref-brand-title">MEJA PENGAWAS</h2>
        </div>
      </div>

      <div class="ref-arena-status">
        <div class="ref-arena-status__row">
          <span class="font-mono font-bold text-sm">{{ arenaTitle }}</span>
          <span class="ref-live-chip font-mono">{{
            c.hasMatch && !c.isFinished ? 'LIVE OPS' : 'SIAGA'
          }}</span>
        </div>
        <div class="ref-arena-status__state font-mono">
          <span class="ref-pulse-dot" :class="{ 'ref-pulse-dot--connected': c.isConnected }" />
          {{ c.isConnected ? 'SISTEM ONLINE' : 'TERPUTUS — MENYAMBUNG ULANG' }}
        </div>
      </div>

      <nav class="ref-nav font-mono">
        <button
          v-for="t in TABS"
          :key="t.key"
          class="ref-nav__item"
          :class="{ 'ref-nav__item--active': activeTab === t.key }"
          @click="activeTab = t.key"
        >
          <span>{{ t.label }}</span>
        </button>
      </nav>

      <div class="ref-telemetry-box font-mono">
        <div class="ref-telem-row">
          <span>TELEMETRI SYNC</span>
          <strong :class="c.isConnected ? 'text-emerald-400' : 'text-red-400'">{{
            c.isConnected ? 'ONLINE' : 'OFFLINE'
          }}</strong>
        </div>
        <div class="ref-telem-stats-list">
          <span>LATENSI: {{ latencyText }}</span>
          <span>JURI ONLINE: {{ judgesOnline }}/{{ judgeSeats }}</span>
          <span>LAYAR TV: {{ tvOnline }}</span>
        </div>
      </div>
    </aside>

    <div class="ref-main-content">
      <header class="ref-topbar">
        <div class="ref-topbar__left">
          <span class="ref-top-series font-mono">PERSILAT DIGITAL ARENA &bull; MEJA PENGAWAS</span>
          <div class="ref-top-headline">
            <h1 class="ref-top-title">{{ arenaTitle }} - MEJA PERTANDINGAN</h1>
            <span class="ref-status-badge font-mono">
              <span class="ref-pulse-dot" />
              {{ c.isConnected ? 'SISTEM KONTROL AKTIF' : 'KONTROL TERPUTUS' }}
            </span>
          </div>
        </div>
        <div class="ref-topbar__right font-mono">
          <div class="ref-referee-info">
            <span class="ref-referee-role">OPERATOR &amp; {{ roleLabel }} RESMI</span>
            <strong class="ref-referee-name">{{ authStore.name || '-' }}</strong>
          </div>
          <span class="ref-lan-chip">
            <span class="ref-pulse-dot" /> {{ judgesOnline }} JURI ONLINE
          </span>
          <div
            class="ref-avatar"
            title="Keluar dari Konsol Wasit (Logout)"
            style="cursor: pointer"
            @click="askLogout"
          />
        </div>
      </header>

      <ControlPanel
        v-if="activeTab === 'kontrol-tanding'"
        :c="c"
        @confirm="askConfirm"
        @goto="goto"
      />
      <JuryPanel v-else-if="activeTab === 'verifikasi-juri'" :c="c" @goto="goto" />
      <SanctionsPanel
        v-else-if="activeTab === 'sanksi-disiplin'"
        :c="c"
        @confirm="askConfirm"
        @goto="goto"
      />
      <LogPanel v-else-if="activeTab === 'log-poin'" :c="c" @goto="goto" />
      <QueuePanel v-else :c="c" @goto="goto" />

      <footer class="ref-footer-strip font-mono">
        <div class="ref-footer-left">
          <span
            >&bull; KONEKSI SERVER:
            <strong :class="c.isConnected ? 'text-emerald-400' : 'text-red-400'">{{
              c.isConnected ? 'TERHUBUNG' : 'TERPUTUS'
            }}</strong></span
          >
          <span class="ref-sep">&bull;</span>
          <span
            >&bull; JURI ONLINE: <strong>{{ judgesOnline }}/{{ judgeSeats }}</strong></span
          >
          <span class="ref-sep">&bull;</span>
          <span
            >&bull; LAYAR TV: <strong>{{ tvOnline }}</strong></span
          >
        </div>
        <div class="ref-footer-right">{{ c.competitionName }}</div>
      </footer>
    </div>

    <div class="ref-toast-stack">
      <div
        v-for="t in c.toasts"
        :key="t.id"
        class="ref-toast font-mono"
        :class="`ref-toast--${t.kind}`"
      >
        {{ t.message }}
      </div>
    </div>

    <div v-if="confirmRequest" class="ref-modal-backdrop">
      <div class="ref-modal-box">
        <h3 class="ref-modal-title">{{ confirmRequest.title }}</h3>
        <p class="ref-modal-msg">{{ confirmRequest.message }}</p>
        <div v-if="confirmRequest.methods" class="ref-choice-list font-mono">
          <label v-for="m in confirmRequest.methods" :key="m.value" class="ref-choice">
            <input v-model="confirmMethod" type="radio" name="confirm-method" :value="m.value" />
            <span>{{ m.label }}</span>
          </label>
        </div>
        <div v-if="confirmRequest.choices" class="ref-choice-list font-mono">
          <label v-for="ch in confirmRequest.choices" :key="ch.value" class="ref-choice">
            <input v-model="confirmChoice" type="radio" name="confirm-choice" :value="ch.value" />
            <span>{{ ch.label }}</span>
          </label>
        </div>
        <div class="ref-modal-actions font-mono">
          <button class="ref-modal-cancel" @click="confirmRequest = null">Batal</button>
          <button
            class="ref-modal-confirm"
            :class="{ 'ref-modal-confirm--danger': confirmRequest.danger }"
            :disabled="!confirmReady"
            @click="runConfirmed"
          >
            {{ confirmRequest.confirmLabel }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
