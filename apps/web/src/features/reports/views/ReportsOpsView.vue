<script setup lang="ts">
import { getErrorMessage } from '@/api/error'
import { ref, computed } from 'vue'
import { AppButton } from '@/components/ui'
import { useAuthStore } from '@/stores/auth'
import { useCompetitionSummaryQuery, useMedalStandingsQuery } from '../api/reportQueries'
import { reportService } from '../api/reportService'
import { useMatchesQuery } from '@/features/matches/api/matchQueries'

interface Props {
  competitionId?: string
}

const props = withDefaults(defineProps<Props>(), {
  competitionId: 'b26d7145-d341-4795-ac58-bff86d13cbf8',
})

const authStore = useAuthStore()
const compIdRef = computed(
  () => props.competitionId || authStore.competitionId || 'b26d7145-d341-4795-ac58-bff86d13cbf8',
)

// Queries
const { data: summary, refetch: refetchSummary } = useCompetitionSummaryQuery(compIdRef)
const {
  data: medals,
  isLoading: isMedalsLoading,
  refetch: refetchMedals,
} = useMedalStandingsQuery(compIdRef)
const { data: matches, isLoading: isMatchesLoading } = useMatchesQuery(compIdRef)

// Active Subtab
const activeSubtab = ref<'medali' | 'partai' | 'statistik'>('medali')

// Toast State
const toast = ref<{
  show: boolean
  title: string
  message: string
  type: 'success' | 'info' | 'warning'
} | null>(null)

function showToast(title: string, message: string, type: 'success' | 'info' | 'warning' = 'info') {
  toast.value = { show: true, title, message, type }
  setTimeout(() => {
    if (toast.value?.title === title) toast.value = null
  }, 3500)
}

// Top Leader (Juara Umum 1)
const topLeader = computed(() => {
  if (medals.value && medals.value.length > 0) {
    return medals.value[0]
  }
  return null
})

// Match status counts
const finishedCount = computed(() => summary.value?.matches_by_status?.finished || 0)
const totalMatches = computed(() => summary.value?.total_matches || matches.value?.length || 0)
const completionRate = computed(() => {
  if (!totalMatches.value) return 0
  return Math.round((finishedCount.value / totalMatches.value) * 100)
})

// Export CSV - Medals
function exportMedalsCsv() {
  if (!medals.value || medals.value.length === 0) {
    showToast('Data Kosong', 'Belum ada data perolehan medali untuk diekspor.', 'warning')
    return
  }

  const headers = ['Peringkat', 'Kontingen / Tim', 'Emas', 'Perak', 'Perunggu', 'Total Medali']
  const rows = medals.value.map((m, idx) => [
    idx + 1,
    `"${m.team.replace(/"/g, '""')}"`,
    m.gold,
    m.silver,
    m.bronze,
    m.total,
  ])

  const csvContent = [headers.join(','), ...rows.map((r) => r.join(','))].join('\n')
  downloadFile(
    csvContent,
    `Klasemen_Medali_${compIdRef.value.slice(0, 8)}.csv`,
    'text/csv;charset=utf-8;',
  )
  showToast('Ekspor Berhasil', 'Klasemen medali berhasil diunduh dalam format CSV.', 'success')
}

// Export CSV - Matches
function exportMatchesCsv() {
  if (!matches.value || matches.value.length === 0) {
    showToast('Data Kosong', 'Belum ada data pertandingan untuk diekspor.', 'warning')
    return
  }

  const headers = ['ID Pertandingan', 'Gelanggang', 'Babak', 'Status', 'Pemenang']
  const rows = matches.value.map((m) => [
    m.id,
    `"${(m.arena_id || '').replace(/"/g, '""')}"`,
    m.current_round || 1,
    m.status || 'scheduled',
    m.winner_id || '-',
  ])

  const csvContent = [headers.join(','), ...rows.map((r) => r.join(','))].join('\n')
  downloadFile(
    csvContent,
    `Rekap_Pertandingan_${compIdRef.value.slice(0, 8)}.csv`,
    'text/csv;charset=utf-8;',
  )
  showToast('Ekspor Berhasil', 'Data pertandingan berhasil diunduh dalam format CSV.', 'success')
}

// Export Full JSON Bundle
const isExportingJson = ref(false)
async function exportFullJson() {
  isExportingJson.value = true
  try {
    const data = await reportService.getExport(compIdRef.value)
    const jsonStr = JSON.stringify(data, null, 2)
    downloadFile(jsonStr, `Turnamen_Export_${compIdRef.value.slice(0, 8)}.json`, 'application/json')
    showToast('Ekspor Sukses', 'Arsip data turnamen lengkap berhasil diunduh (.JSON)', 'success')
  } catch (err) {
    showToast(
      'Gagal Ekspor',
      getErrorMessage(err, 'Terjadi kesalahan saat mengunduh data turnamen.'),
      'warning',
    )
  } finally {
    isExportingJson.value = false
  }
}

function downloadFile(content: string, filename: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

function handlePrint() {
  window.print()
}

function refreshAll() {
  refetchSummary()
  refetchMedals()
  showToast('Data Diperbarui', 'Laporan dan klasemen turnamen telah disinkronkan.', 'info')
}
</script>

<template>
  <div class="reports-view">
    <!-- SUB-BANNER / TITLE ROW -->
    <div class="reports-banner no-print">
      <div class="reports-title-group">
        <div class="reports-icon-badge">
          <svg
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            stroke="#10B981"
            stroke-width="2.2"
          >
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
            <polyline points="14 2 14 8 20 8" />
            <line x1="16" y1="13" x2="8" y2="13" />
            <line x1="16" y1="17" x2="8" y2="17" />
            <polyline points="10 9 9 9 8 9" />
          </svg>
        </div>
        <div>
          <div class="reports-sub font-mono">
            <span>REKAPITULASI HASIL, KLASEMEN MEDALI & ARSIP PB IPSI</span>
            <span class="bullet">•</span>
            <span class="text-emerald-400">STANDAR RESMI NASIONAL</span>
          </div>
          <h2 class="reports-title">LAPORAN & EKSPOR KOMPETISI</h2>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="reports-actions font-mono">
        <button class="reports-action-btn font-mono" title="Refresh Data" @click="refreshAll">
          <svg
            width="15"
            height="15"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2"
            />
          </svg>
          <span>Segarkan</span>
        </button>

        <button
          class="reports-action-btn font-mono"
          :disabled="isExportingJson"
          @click="exportFullJson"
        >
          <svg
            width="15"
            height="15"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="7 10 12 15 17 10" />
            <line x1="12" y1="15" x2="12" y2="3" />
          </svg>
          <span>{{ isExportingJson ? 'Mengunduh...' : 'Ekspor JSON' }}</span>
        </button>

        <button class="reports-action-btn font-mono" @click="exportMedalsCsv">
          <svg
            width="15"
            height="15"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <polyline points="6 9 6 2 18 2 18 9" />
            <path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2" />
            <rect width="12" height="8" x="6" y="14" />
          </svg>
          <span>Unduh CSV Medali</span>
        </button>

        <AppButton variant="emerald" size="sm" @click="handlePrint">
          <template #icon>
            <svg
              width="15"
              height="15"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <polyline points="6 9 6 2 18 2 18 9" />
              <path
                d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"
              />
              <rect width="12" height="8" x="6" y="14" />
            </svg>
          </template>
          Cetak Rekap Resmi
        </AppButton>
      </div>
    </div>

    <!-- PRINT HEADER (ONLY VISIBLE ON PRINT) -->
    <div class="print-header font-mono">
      <div class="print-logo">IPSI</div>
      <div class="print-titles">
        <h2>PENGURUS BESAR IKATAN PENCAK SILAT INDONESIA</h2>
        <h3>REKAPITULASI RESMI & KLASEMEN AKHIR PERTANDINGAN</h3>
        <p>{{ summary?.competition_name || 'Kejuaraan Nasional Pencak Silat' }}</p>
      </div>
      <div class="print-date">TANGGAL: {{ new Date().toLocaleDateString('id-ID') }}</div>
    </div>

    <!-- KPI STATS CARDS ROW -->
    <div class="kpi-grid font-mono">
      <!-- Total Partai Card -->
      <div class="kpi-card">
        <div class="kpi-label">TOTAL PARTAI TANDING</div>
        <div class="kpi-val-row">
          <span class="kpi-num text-emerald-400 tabular-nums">{{ finishedCount }}</span>
          <span class="kpi-divider">/</span>
          <span class="kpi-total tabular-nums">{{ totalMatches }} Selesai</span>
        </div>
        <div class="kpi-progress-bar">
          <div class="kpi-progress-fill" :style="{ width: `${completionRate}%` }" />
        </div>
        <div class="kpi-footer-text">Progress Kejuaraan: {{ completionRate }}%</div>
      </div>

      <!-- Peserta & Kontingen Card -->
      <div class="kpi-card">
        <div class="kpi-label">TOTAL ATLET & KONTINGEN</div>
        <div class="kpi-val-row">
          <span class="kpi-num text-sky-400 tabular-nums">{{ summary?.total_athletes || 0 }}</span>
          <span class="kpi-subtext">Atlet Terdaftar</span>
        </div>
        <div class="kpi-meta-line">
          <span
            >Kategori: <strong>{{ summary?.total_categories || 0 }}</strong></span
          >
          <span>•</span>
          <span
            >Gelanggang: <strong>{{ summary?.total_arenas || 0 }}</strong></span
          >
        </div>
        <div class="kpi-footer-text">Terverifikasi PB IPSI</div>
      </div>

      <!-- Juara Umum Card -->
      <div class="kpi-card kpi-card--gold">
        <div class="kpi-label">PEMIMPIN KLASEMEN (JUARA UMUM)</div>
        <div class="kpi-val-row">
          <span class="kpi-crown">👑</span>
          <span class="kpi-leader-name truncate">{{
            topLeader ? topLeader.team : 'Belum Ada'
          }}</span>
        </div>
        <div v-if="topLeader" class="kpi-medal-tally tabular-nums">
          <span class="medal-tally-item gold">🥇 {{ topLeader.gold }} Emas</span>
          <span class="medal-tally-item silver">🥈 {{ topLeader.silver }} Perak</span>
          <span class="medal-tally-item bronze">🥉 {{ topLeader.bronze }} Perunggu</span>
        </div>
        <div v-else class="kpi-footer-text">Pertandingan belum menghasilkan juara</div>
      </div>
    </div>

    <!-- SUBTABS BAR -->
    <div class="subtabs-bar font-mono no-print">
      <button
        class="subtab-btn"
        :class="{ 'subtab-btn--active': activeSubtab === 'medali' }"
        @click="activeSubtab = 'medali'"
      >
        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path d="M6 9H4.5a2.5 2.5 0 0 1 0-5H6" />
          <path d="M18 9h1.5a2.5 2.5 0 0 0 0-5H18" />
          <path d="M4 22h16" />
          <path d="M10 14.66V17c0 .55-.47.98-.97 1.21C7.85 18.75 7 20.24 7 22" />
          <path d="M14 14.66V17c0 .55.47.98.97 1.21C16.15 18.75 17 20.24 17 22" />
          <path d="M18 2H6v7a6 6 0 0 0 12 0V2Z" />
        </svg>
        <span>Klasemen Perolehan Medali</span>
        <span class="subtab-count">{{ medals?.length || 0 }}</span>
      </button>

      <button
        class="subtab-btn"
        :class="{ 'subtab-btn--active': activeSubtab === 'partai' }"
        @click="activeSubtab = 'partai'"
      >
        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <rect width="18" height="18" x="3" y="4" rx="2" />
          <line x1="16" y1="2" x2="16" y2="6" />
          <line x1="8" y1="2" x2="8" y2="6" />
          <line x1="3" y1="10" x2="21" y2="10" />
        </svg>
        <span>Rekapitulasi Partai Selesai</span>
        <span class="subtab-count">{{ finishedCount }}</span>
      </button>

      <button
        class="subtab-btn"
        :class="{ 'subtab-btn--active': activeSubtab === 'statistik' }"
        @click="activeSubtab = 'statistik'"
      >
        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <line x1="18" y1="20" x2="18" y2="10" />
          <line x1="12" y1="20" x2="12" y2="4" />
          <line x1="6" y1="20" x2="6" y2="14" />
        </svg>
        <span>Ringkasan Status Turnamen</span>
      </button>
    </div>

    <!-- VIEW 1: KLASEMEN MEDALI -->
    <div v-if="activeSubtab === 'medali'" class="report-card">
      <div class="report-card-header no-print">
        <div>
          <h3 class="report-card-title font-mono">TABEL PEROLEHAN MEDALI RESMI</h3>
          <p class="report-card-sub">
            Urutan peringkat berdasarkan jumlah perolehan Emas, Perak, lalu Perunggu.
          </p>
        </div>
        <button class="export-chip-btn font-mono" @click="exportMedalsCsv">
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="7 10 12 15 17 10" />
            <line x1="12" y1="15" x2="12" y2="3" />
          </svg>
          Unduh CSV
        </button>
      </div>

      <!-- Loading State -->
      <div v-if="isMedalsLoading" class="table-loading-box font-mono">
        <span class="animate-spin text-xl">⏳</span>
        <span>Memuat data perolehan medali...</span>
      </div>

      <!-- Empty State -->
      <div v-else-if="!medals || medals.length === 0" class="table-empty-box font-mono">
        <svg
          width="40"
          height="40"
          viewBox="0 0 24 24"
          fill="none"
          stroke="#64748B"
          stroke-width="1.5"
        >
          <circle cx="12" cy="12" r="10" />
          <path d="M12 6v6l4 2" />
        </svg>
        <div class="empty-title">Belum Ada Perolehan Medali</div>
        <div class="empty-desc">
          Medali akan otomatis terhitung saat pertandingan final atau penyisihan diselesaikan oleh
          dewan wasit.
        </div>
      </div>

      <!-- Standings Table -->
      <div v-else class="table-responsive">
        <table class="report-table">
          <thead>
            <tr class="font-mono">
              <th class="th-rank">NO</th>
              <th class="th-team">KONTINGEN / TIM</th>
              <th class="th-gold">🥇 EMAS</th>
              <th class="th-silver">🥈 PERAK</th>
              <th class="th-bronze">🥉 PERUNGGU</th>
              <th class="th-total">TOTAL</th>
              <th class="th-bar no-print">DISTRIBUSI</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(standing, index) in medals"
              :key="standing.team"
              class="standing-row font-mono"
              :class="{
                'standing-row--first': index === 0,
                'standing-row--second': index === 1,
                'standing-row--third': index === 2,
              }"
            >
              <td class="td-rank">
                <div class="rank-badge" :class="`rank-badge--${index < 3 ? index + 1 : 'default'}`">
                  {{ index + 1 }}
                </div>
              </td>
              <td class="td-team font-bold">
                <div class="team-name-cell">
                  <span>{{ standing.team }}</span>
                  <span v-if="index === 0" class="leader-badge">JUARA 1</span>
                </div>
              </td>
              <td class="td-gold tabular-nums">
                <span class="medal-pill gold-pill">{{ standing.gold }}</span>
              </td>
              <td class="td-silver tabular-nums">
                <span class="medal-pill silver-pill">{{ standing.silver }}</span>
              </td>
              <td class="td-bronze tabular-nums">
                <span class="medal-pill bronze-pill">{{ standing.bronze }}</span>
              </td>
              <td class="td-total tabular-nums font-bold text-white">
                {{ standing.total }}
              </td>
              <td class="td-bar no-print">
                <div class="mini-bar-track">
                  <div
                    class="mini-bar-gold"
                    :style="{
                      width: `${standing.total ? (standing.gold / standing.total) * 100 : 0}%`,
                    }"
                    title="Emas"
                  />
                  <div
                    class="mini-bar-silver"
                    :style="{
                      width: `${standing.total ? (standing.silver / standing.total) * 100 : 0}%`,
                    }"
                    title="Perak"
                  />
                  <div
                    class="mini-bar-bronze"
                    :style="{
                      width: `${standing.total ? (standing.bronze / standing.total) * 100 : 0}%`,
                    }"
                    title="Perunggu"
                  />
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- VIEW 2: REKAPITULASI PARTAI SELESAI -->
    <div v-else-if="activeSubtab === 'partai'" class="report-card">
      <div class="report-card-header no-print">
        <div>
          <h3 class="report-card-title font-mono">HASIL PERTANDINGAN SELESAI</h3>
          <p class="report-card-sub">
            Daftar partai yang telah selesai dengan penentuan pemenang resmi.
          </p>
        </div>
        <button class="export-chip-btn font-mono" @click="exportMatchesCsv">
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="7 10 12 15 17 10" />
            <line x1="12" y1="15" x2="12" y2="3" />
          </svg>
          Unduh CSV
        </button>
      </div>

      <!-- Loading State -->
      <div v-if="isMatchesLoading" class="table-loading-box font-mono">
        <span class="animate-spin text-xl">⏳</span>
        <span>Memuat data pertandingan...</span>
      </div>

      <!-- Empty State -->
      <div v-else-if="!matches || matches.length === 0" class="table-empty-box font-mono">
        <div class="empty-title">Belum Ada Pertandingan Selesai</div>
        <div class="empty-desc">
          Pertandingan yang diselesaikan di meja wasit akan muncul di sini beserta ringkasan
          pemenang.
        </div>
      </div>

      <!-- Matches Table -->
      <div v-else class="table-responsive">
        <table class="report-table">
          <thead>
            <tr class="font-mono">
              <th>ID PARTAI</th>
              <th>GELANGGANG</th>
              <th>ROUND</th>
              <th>STATUS</th>
              <th>PEMENANG</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in matches" :key="m.id" class="match-row font-mono">
              <td class="font-bold text-slate-300">#{{ m.id.slice(0, 8) }}</td>
              <td>
                <span class="arena-tag">{{ m.arena_id?.slice(0, 8) || 'GELANGGANG A' }}</span>
              </td>
              <td class="tabular-nums">Babak {{ m.current_round || 1 }}</td>
              <td>
                <span
                  class="status-chip"
                  :class="m.status === 'finished' ? 'status-chip--finished' : 'status-chip--live'"
                >
                  {{ m.status === 'finished' ? 'SELESAI' : m.status.toUpperCase() }}
                </span>
              </td>
              <td class="text-emerald-400 font-bold">
                {{ m.winner_id ? `Pemenang Terkonfirmasi (#${m.winner_id.slice(0, 6)})` : '-' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- VIEW 3: RINGKASAN STATUS TURNAMEN -->
    <div v-else-if="activeSubtab === 'statistik'" class="report-card">
      <div class="report-card-header no-print">
        <div>
          <h3 class="report-card-title font-mono">STATISTIK OPERASIONAL TURNAMEN</h3>
          <p class="report-card-sub">
            Distribusi status partai, ketersediaan gelanggang dan integritas sistem pertandingan.
          </p>
        </div>
      </div>

      <div class="stats-breakdown-grid font-mono">
        <div class="stat-box">
          <div class="stat-box-label">KOMPETISI AKTIF</div>
          <div class="stat-box-val text-white">
            {{ summary?.competition_name || 'Turnamen Silat' }}
          </div>
          <div class="stat-box-sub">ID: {{ compIdRef }}</div>
        </div>

        <div class="stat-box">
          <div class="stat-box-label">GELANGGANG BEROPERASI</div>
          <div class="stat-box-val text-sky-400 tabular-nums">
            {{ summary?.total_arenas || 0 }} Gelanggang
          </div>
          <div class="stat-box-sub">Siap digunakan secara paralel</div>
        </div>

        <div class="stat-box">
          <div class="stat-box-label">KATEGORI TANDING DIBUKA</div>
          <div class="stat-box-val text-amber-400 tabular-nums">
            {{ summary?.total_categories || 0 }} Kelas
          </div>
          <div class="stat-box-sub">Putra & Putri Standar Persilat</div>
        </div>

        <div class="stat-box">
          <div class="stat-box-label">DISTRIBUSI PARTAI SELESAI</div>
          <div class="stat-box-val text-emerald-400 tabular-nums">{{ finishedCount }} Selesai</div>
          <div class="stat-box-sub">{{ totalMatches - finishedCount }} Belum Selesai</div>
        </div>
      </div>
    </div>

    <!-- PRINT SIGNATURE SECTION (ONLY PRINT) -->
    <div class="print-signatures font-mono">
      <div class="sig-block">
        <div class="sig-title">Mengetahui,</div>
        <div class="sig-role">Delegasi Teknik (Technical Delegate)</div>
        <div class="sig-space" />
        <div class="sig-name">( ..................................................... )</div>
      </div>

      <div class="sig-block">
        <div class="sig-title">Disahkan oleh,</div>
        <div class="sig-role">Ketua Pertandingan (Competition Director)</div>
        <div class="sig-space" />
        <div class="sig-name">( ..................................................... )</div>
      </div>
    </div>

    <!-- TOAST NOTIFICATION -->
    <Transition name="fade">
      <div v-if="toast" class="report-toast font-mono" :class="`report-toast--${toast.type}`">
        <div class="toast-title font-bold">{{ toast.title }}</div>
        <div class="toast-msg">{{ toast.message }}</div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.reports-view {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* Banner */
.reports-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  background-color: #0e1626;
  border: 1px solid rgba(255, 255, 255, 0.08);
  padding: 18px 24px;
  border-radius: 10px;
}

.reports-title-group {
  display: flex;
  align-items: center;
  gap: 16px;
}

.reports-icon-badge {
  width: 44px;
  height: 44px;
  background-color: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.3);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.reports-sub {
  font-size: 0.75rem;
  color: #94a3b8;
  letter-spacing: 0.06em;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.reports-title {
  font-size: 1.375rem;
  font-weight: 800;
  letter-spacing: 0.02em;
  color: #ffffff;
}

.reports-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.reports-action-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background-color: #162032;
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #e2e8f0;
  padding: 8px 14px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.reports-action-btn:hover {
  background-color: #1e293b;
  border-color: rgba(255, 255, 255, 0.2);
}

/* KPI Grid */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.kpi-card {
  background-color: #0e1626;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.kpi-card--gold {
  border-color: rgba(245, 158, 11, 0.3);
  background: linear-gradient(180deg, #1a1710 0%, #0e1626 100%);
}

.kpi-label {
  font-size: 0.6875rem;
  color: #94a3b8;
  letter-spacing: 0.06em;
}

.kpi-val-row {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.kpi-num {
  font-size: 1.875rem;
  font-weight: 800;
}

.kpi-divider {
  font-size: 1.25rem;
  color: #64748b;
}

.kpi-total {
  font-size: 1rem;
  color: #94a3b8;
}

.kpi-subtext {
  font-size: 0.8125rem;
  color: #94a3b8;
}

.kpi-crown {
  font-size: 1.5rem;
}

.kpi-leader-name {
  font-size: 1.25rem;
  font-weight: 800;
  color: #f59e0b;
}

.kpi-progress-bar {
  height: 6px;
  background-color: #1e293b;
  border-radius: 3px;
  overflow: hidden;
  margin-top: 4px;
}

.kpi-progress-fill {
  height: 100%;
  background-color: #10b981;
  transition: width 0.3s ease;
}

.kpi-meta-line {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.75rem;
  color: #cbd5e1;
}

.kpi-meta-line strong {
  color: #38bdf8;
}

.kpi-medal-tally {
  display: flex;
  gap: 10px;
  font-size: 0.75rem;
  margin-top: 2px;
}

.medal-tally-item.gold {
  color: #f59e0b;
}
.medal-tally-item.silver {
  color: #94a3b8;
}
.medal-tally-item.bronze {
  color: #d97706;
}

.kpi-footer-text {
  font-size: 0.6875rem;
  color: #64748b;
}

/* Subtabs Bar */
.subtabs-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  padding-bottom: 12px;
}

.subtab-btn {
  background: #0e1626;
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #94a3b8;
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: all 0.15s ease;
}

.subtab-btn:hover {
  color: #ffffff;
  background-color: #162032;
}

.subtab-btn--active {
  background-color: #1e293b;
  border-color: #10b981;
  color: #10b981;
}

.subtab-count {
  font-size: 0.625rem;
  background-color: rgba(255, 255, 255, 0.08);
  padding: 2px 6px;
  border-radius: 4px;
}

/* Report Card & Tables */
.report-card {
  background-color: #0e1626;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.report-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.report-card-title {
  font-size: 1rem;
  font-weight: 800;
  color: #ffffff;
}

.report-card-sub {
  font-size: 0.75rem;
  color: #94a3b8;
}

.export-chip-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  background-color: #162032;
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #cbd5e1;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 0.6875rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.export-chip-btn:hover {
  color: #ffffff;
  background-color: #1e293b;
}

.table-responsive {
  overflow-x: auto;
}

.report-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}

.report-table th {
  padding: 10px 14px;
  font-size: 0.6875rem;
  color: #64748b;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  letter-spacing: 0.05em;
}

.report-table td {
  padding: 12px 14px;
  font-size: 0.8125rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  color: #cbd5e1;
}

.standing-row:hover,
.match-row:hover {
  background-color: rgba(255, 255, 255, 0.02);
}

.rank-badge {
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  font-weight: 800;
  font-size: 0.75rem;
  background-color: #162032;
  color: #94a3b8;
}

.rank-badge--1 {
  background-color: rgba(245, 158, 11, 0.2);
  color: #f59e0b;
  border: 1px solid rgba(245, 158, 11, 0.4);
}

.rank-badge--2 {
  background-color: rgba(148, 163, 184, 0.2);
  color: #e2e8f0;
  border: 1px solid rgba(148, 163, 184, 0.4);
}

.rank-badge--3 {
  background-color: rgba(217, 119, 6, 0.2);
  color: #f97316;
  border: 1px solid rgba(217, 119, 6, 0.4);
}

.team-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.leader-badge {
  font-size: 0.5625rem;
  background-color: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  padding: 2px 6px;
  border-radius: 4px;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.medal-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  height: 24px;
  border-radius: 6px;
  font-weight: 700;
  font-size: 0.8125rem;
}

.gold-pill {
  background-color: rgba(245, 158, 11, 0.12);
  color: #f59e0b;
}

.silver-pill {
  background-color: rgba(148, 163, 184, 0.12);
  color: #cbd5e1;
}

.bronze-pill {
  background-color: rgba(217, 119, 6, 0.12);
  color: #f97316;
}

.mini-bar-track {
  display: flex;
  height: 8px;
  width: 140px;
  background-color: #1e293b;
  border-radius: 4px;
  overflow: hidden;
}

.mini-bar-gold {
  background-color: #f59e0b;
}
.mini-bar-silver {
  background-color: #94a3b8;
}
.mini-bar-bronze {
  background-color: #d97706;
}

.status-chip {
  padding: 3px 8px;
  border-radius: 4px;
  font-size: 0.6875rem;
  font-weight: 700;
}

.status-chip--finished {
  background-color: rgba(16, 185, 129, 0.12);
  color: #10b981;
}

.status-chip--live {
  background-color: rgba(56, 189, 248, 0.12);
  color: #38bdf8;
}

.arena-tag {
  background-color: #162032;
  padding: 3px 6px;
  border-radius: 4px;
  font-size: 0.6875rem;
  color: #94a3b8;
}

/* Stats Breakdown Grid */
.stats-breakdown-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.stat-box {
  background-color: #080d1a;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-box-label {
  font-size: 0.6875rem;
  color: #64748b;
  letter-spacing: 0.05em;
}

.stat-box-val {
  font-size: 1.125rem;
  font-weight: 800;
}

.stat-box-sub {
  font-size: 0.6875rem;
  color: #94a3b8;
}

/* States */
.table-loading-box,
.table-empty-box {
  padding: 48px 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: #94a3b8;
  font-size: 0.8125rem;
}

.empty-title {
  font-weight: 700;
  color: #e2e8f0;
  font-size: 0.9375rem;
}

.empty-desc {
  max-width: 400px;
  text-align: center;
  font-size: 0.75rem;
  color: #64748b;
}

/* Toast */
.report-toast {
  position: fixed;
  top: 76px;
  right: 28px;
  background-color: #0f172a;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 8px;
  padding: 12px 18px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.5);
  z-index: 150;
}

.report-toast--success {
  border-color: #10b981;
}
.report-toast--info {
  border-color: #38bdf8;
}
.report-toast--warning {
  border-color: #f59e0b;
}

/* Print styles */
.print-header,
.print-signatures {
  display: none;
}

@media print {
  .no-print {
    display: none !important;
  }

  body * {
    visibility: hidden;
  }

  .reports-view,
  .reports-view * {
    visibility: visible;
  }

  .reports-view {
    position: absolute;
    left: 0;
    top: 0;
    width: 100%;
    color: #000000;
    background: #ffffff;
    padding: 24px;
  }

  .print-header {
    display: flex;
    align-items: center;
    gap: 20px;
    border-bottom: 2px solid #000000;
    padding-bottom: 16px;
    margin-bottom: 24px;
  }

  .print-logo {
    font-size: 2rem;
    font-weight: 900;
    border: 2px solid #000000;
    padding: 6px 12px;
  }

  .print-titles h2 {
    font-size: 1.125rem;
    font-weight: 800;
  }

  .print-titles h3 {
    font-size: 0.9375rem;
    font-weight: 700;
  }

  .print-titles p {
    font-size: 0.8125rem;
  }

  .report-card {
    background: transparent;
    border: none;
    padding: 0;
  }

  .report-table th,
  .report-table td {
    color: #000000;
    border-color: #cbd5e1;
  }

  .print-signatures {
    display: flex;
    justify-content: space-between;
    margin-top: 60px;
    padding: 0 40px;
  }

  .sig-block {
    text-align: center;
  }

  .sig-title {
    font-size: 0.75rem;
  }

  .sig-role {
    font-weight: 700;
    font-size: 0.8125rem;
  }

  .sig-space {
    height: 70px;
  }

  .sig-name {
    font-size: 0.8125rem;
  }
}

@media (max-width: 900px) {
  .kpi-grid {
    grid-template-columns: 1fr;
  }
  .reports-banner {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
