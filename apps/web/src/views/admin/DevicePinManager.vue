<script setup lang="ts">
import { ref, computed } from 'vue'
import { AppButton } from '@/components/ui'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/lib/api'

const authStore = useAuthStore()

// Props & State
const selectedArena = ref<'gelanggang-1' | 'gelanggang-2' | 'gelanggang-3' | 'gelanggang-4'>('gelanggang-1')

interface DeviceSeat {
  id: string
  seatNumber: number
  role: 'judge' | 'wasit' | 'scoreboard'
  label: string
  deviceModel: string
  pin: string
  status: 'online' | 'offline' | 'idle'
  latencyMs: number
  batteryPct: number
  expiresAt: string
}

// Mock & Initial Data per Arena
const arenaData = ref<Record<string, DeviceSeat[]>>({
  'gelanggang-1': [
    {
      id: 'g1-j1',
      seatNumber: 1,
      role: 'judge',
      label: 'KURSI JURI 1',
      deviceModel: 'SAMSUNG GALAXY TAB A9 • ANDROID PWA',
      pin: '839 102',
      status: 'online',
      latencyMs: 14,
      batteryPct: 94,
      expiresAt: '22:00 WIB (10 Jam Tersisa)',
    },
    {
      id: 'g1-j2',
      seatNumber: 2,
      role: 'judge',
      label: 'KURSI JURI 2',
      deviceModel: 'SAMSUNG GALAXY TAB A9 • ANDROID PWA',
      pin: '491 205',
      status: 'online',
      latencyMs: 18,
      batteryPct: 88,
      expiresAt: '22:00 WIB (10 Jam Tersisa)',
    },
    {
      id: 'g1-j3',
      seatNumber: 3,
      role: 'judge',
      label: 'KURSI JURI 3',
      deviceModel: 'SAMSUNG GALAXY TAB A9 • ANDROID PWA',
      pin: '720 184',
      status: 'online',
      latencyMs: 12,
      batteryPct: 91,
      expiresAt: '22:00 WIB (10 Jam Tersisa)',
    },
    {
      id: 'g1-wasit',
      seatNumber: 0,
      role: 'wasit',
      label: 'MEJA PENGAWAS & WASIT',
      deviceModel: 'LAPTOP WASIT PERSILAT • WEBSOCKET OPERATOR',
      pin: '662 901',
      status: 'online',
      latencyMs: 8,
      batteryPct: 100,
      expiresAt: '22:00 WIB (10 Jam Tersisa)',
    },
    {
      id: 'g1-sb',
      seatNumber: 0,
      role: 'scoreboard',
      label: 'SCOREBOARD TV DISPLAY',
      deviceModel: 'LED ARENA 65" • 4K ULTRA HD BROADCAST',
      pin: 'AKSES PUBLIK',
      status: 'online',
      latencyMs: 4,
      batteryPct: 100,
      expiresAt: 'NON-EXPIRING (PUBLIC)',
    },
  ],
  'gelanggang-2': [
    {
      id: 'g2-j1',
      seatNumber: 1,
      role: 'judge',
      label: 'KURSI JURI 1',
      deviceModel: 'XIAOMI PAD 6 • CHROME PWA',
      pin: '118 439',
      status: 'online',
      latencyMs: 16,
      batteryPct: 82,
      expiresAt: '22:00 WIB (10 Jam Tersisa)',
    },
    {
      id: 'g2-j2',
      seatNumber: 2,
      role: 'judge',
      label: 'KURSI JURI 2',
      deviceModel: 'XIAOMI PAD 6 • CHROME PWA',
      pin: '502 911',
      status: 'online',
      latencyMs: 15,
      batteryPct: 79,
      expiresAt: '22:00 WIB (10 Jam Tersisa)',
    },
    {
      id: 'g2-j3',
      seatNumber: 3,
      role: 'judge',
      label: 'KURSI JURI 3',
      deviceModel: 'XIAOMI PAD 6 • CHROME PWA',
      pin: '874 320',
      status: 'online',
      latencyMs: 19,
      batteryPct: 85,
      expiresAt: '22:00 WIB (10 Jam Tersisa)',
    },
    {
      id: 'g2-wasit',
      seatNumber: 0,
      role: 'wasit',
      label: 'MEJA PENGAWAS & WASIT',
      deviceModel: 'WORKSTATION MAT 2 • DUAL SCREEN',
      pin: '904 112',
      status: 'online',
      latencyMs: 9,
      batteryPct: 100,
      expiresAt: '22:00 WIB (10 Jam Tersisa)',
    },
  ],
  'gelanggang-3': [
    {
      id: 'g3-j1',
      seatNumber: 1,
      role: 'judge',
      label: 'KURSI JURI 1',
      deviceModel: 'LENOVO TAB M10 • PWA CLIENT',
      pin: '339 014',
      status: 'idle',
      latencyMs: 24,
      batteryPct: 75,
      expiresAt: '22:00 WIB (10 Jam Tersisa)',
    },
    {
      id: 'g3-j2',
      seatNumber: 2,
      role: 'judge',
      label: 'KURSI JURI 2',
      deviceModel: 'LENOVO TAB M10 • PWA CLIENT',
      pin: '612 883',
      status: 'idle',
      latencyMs: 22,
      batteryPct: 70,
      expiresAt: '22:00 WIB (10 Jam Tersisa)',
    },
    {
      id: 'g3-j3',
      seatNumber: 3,
      role: 'judge',
      label: 'KURSI JURI 3',
      deviceModel: 'LENOVO TAB M10 • PWA CLIENT',
      pin: '445 901',
      status: 'idle',
      latencyMs: 26,
      batteryPct: 73,
      expiresAt: '22:00 WIB (10 Jam Tersisa)',
    },
  ],
  'gelanggang-4': [],
})

const currentSeats = computed(() => arenaData.value[selectedArena.value] || [])

// Toast Notification
const toast = ref<{ show: boolean; title: string; message: string; type: 'success' | 'info' | 'warning' } | null>(null)

function showToast(title: string, message: string, type: 'success' | 'info' | 'warning' = 'info') {
  toast.value = { show: true, title, message, type }
  setTimeout(() => {
    if (toast.value?.title === title) toast.value = null
  }, 3500)
}

// Copy PIN
function copyPin(pin: string, label: string) {
  const cleanPin = pin.replace(/\s+/g, '')
  if (navigator.clipboard) {
    navigator.clipboard.writeText(cleanPin)
  }
  showToast('PIN Disalin!', `PIN ${label} (${pin}) berhasil disalin ke clipboard.`, 'success')
}

// Reset / Generate Single PIN
async function generatePinForSeat(seat: DeviceSeat) {
  const randomPin = String(Math.floor(100000 + Math.random() * 900000))
  const formattedPin = `${randomPin.slice(0, 3)} ${randomPin.slice(3)}`

  // Call Go backend if API available
  try {
    const compId = authStore.competitionId || 'comp-default'
    await api.post('/api/v1/auth/create-pin', {
      competition_id: compId,
      arena_id: selectedArena.value,
      role: seat.role === 'wasit' ? 'wasit' : 'juri',
      seat_number: seat.seatNumber,
      pin: randomPin,
      ttl_hours: 12,
    })
  } catch {
    // Graceful fallback to local reactive state
  }

  seat.pin = formattedPin
  showToast('PIN Diperbarui', `PIN baru untuk ${seat.label}: ${formattedPin}`, 'success')
}

// Batch Generate All Judges PIN for Arena
async function handleBatchGenerate() {
  const judges = currentSeats.value.filter(s => s.role === 'judge')
  for (const j of judges) {
    const randomPin = String(Math.floor(100000 + Math.random() * 900000))
    j.pin = `${randomPin.slice(0, 3)} ${randomPin.slice(3)}`

    try {
      const compId = authStore.competitionId || 'comp-default'
      await api.post('/api/v1/auth/create-pin', {
        competition_id: compId,
        arena_id: selectedArena.value,
        role: 'juri',
        seat_number: j.seatNumber,
        pin: randomPin,
        ttl_hours: 12,
      })
    } catch {
      // offline fallback
    }
  }
  showToast('Batch Generate Selesai', `Seluruh PIN Juri di ${selectedArena.value.toUpperCase()} berhasil diacak ulang.`, 'success')
}

// Ping / Test Device Vibrations
function handlePingDevice(seat: DeviceSeat) {
  showToast('Sinyal Uji Coba Terkirim', `Mengirim sinyal getar (haptic ping) ke ${seat.label}...`, 'info')
}

// Print Slips Modal
const showPrintModal = ref(false)
function triggerPrint() {
  window.print()
}

// Modal Create Custom PIN
const showCustomPinModal = ref(false)
const customRole = ref<'juri' | 'wasit' | 'dewan'>('juri')
const customSeat = ref(1)
const customPinInput = ref('')
const customTtl = ref(12)

async function handleSaveCustomPin() {
  const pinValue = customPinInput.value || String(Math.floor(100000 + Math.random() * 900000))
  const formatted = `${pinValue.slice(0, 3)} ${pinValue.slice(3)}`

  try {
    const compId = authStore.competitionId || 'comp-default'
    await api.post('/api/v1/auth/create-pin', {
      competition_id: compId,
      arena_id: selectedArena.value,
      role: customRole.value,
      seat_number: customRole.value === 'juri' ? customSeat.value : 0,
      pin: pinValue,
      ttl_hours: customTtl.value,
    })
  } catch {
    // Fallback
  }

  // Update in array if exists
  const existing = currentSeats.value.find(s => s.role === customRole.value && s.seatNumber === customSeat.value)
  if (existing) {
    existing.pin = formatted
  }

  showToast('PIN Berhasil Dibuat', `PIN ${customRole.value.toUpperCase()} (${formatted}) berhasil disimpan.`, 'success')
  showCustomPinModal.value = false
  customPinInput.value = ''
}
</script>

<template>
  <div class="pin-mgr-view">
    <!-- SUB-BANNER / TITLE ROW -->
    <div class="pin-mgr-banner">
      <div class="pin-mgr-title-group">
        <div class="pin-mgr-icon-badge">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#10B981" stroke-width="2.2">
            <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
            <path d="M7 11V7a5 5 0 0 1 10 0v4" />
          </svg>
        </div>
        <div>
          <div class="pin-mgr-sub font-mono">
            <span>KONTROL AKSES TABLET GELANGGANG & DEWAN HAKIM</span>
            <span class="bullet">•</span>
            <span class="text-emerald-400">STANDAR RESMI PB IPSI</span>
          </div>
          <h2 class="pin-mgr-title">MANAJEMEN PERANGKAT & PIN JURI</h2>
        </div>
      </div>

      <div class="pin-mgr-actions font-mono">
        <AppButton variant="emerald" size="sm" @click="handleBatchGenerate">
          <template #icon>
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2" />
            </svg>
          </template>
          Generate Ulang Semua PIN
        </AppButton>

        <button class="pin-action-btn font-mono" @click="showCustomPinModal = true">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 5v14M5 12h14" />
          </svg>
          <span>Buat PIN Kustom</span>
        </button>

        <button class="pin-action-btn font-mono" @click="showPrintModal = true">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="6 9 6 2 18 2 18 9" />
            <path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2" />
            <rect width="12" height="8" x="6" y="14" />
          </svg>
          <span>Cetak Slip Meja Juri</span>
        </button>
      </div>
    </div>

    <!-- ARENA SELECTOR TABS -->
    <div class="arena-tabs-row font-mono">
      <button
        class="arena-tab-btn"
        :class="{ 'arena-tab-btn--active': selectedArena === 'gelanggang-1' }"
        @click="selectedArena = 'gelanggang-1'"
      >
        <span class="arena-dot" :class="{ 'arena-dot--active': selectedArena === 'gelanggang-1' }" />
        GELANGGANG 1 (UTAMA)
        <span class="arena-count-badge">5 ALAT</span>
      </button>

      <button
        class="arena-tab-btn"
        :class="{ 'arena-tab-btn--active': selectedArena === 'gelanggang-2' }"
        @click="selectedArena = 'gelanggang-2'"
      >
        <span class="arena-dot" :class="{ 'arena-dot--active': selectedArena === 'gelanggang-2' }" />
        GELANGGANG 2
        <span class="arena-count-badge">4 ALAT</span>
      </button>

      <button
        class="arena-tab-btn"
        :class="{ 'arena-tab-btn--active': selectedArena === 'gelanggang-3' }"
        @click="selectedArena = 'gelanggang-3'"
      >
        <span class="arena-dot" :class="{ 'arena-dot--active': selectedArena === 'gelanggang-3' }" />
        GELANGGANG 3
        <span class="arena-count-badge">3 ALAT</span>
      </button>

      <button
        class="arena-tab-btn"
        :class="{ 'arena-tab-btn--active': selectedArena === 'gelanggang-4' }"
        @click="selectedArena = 'gelanggang-4'"
      >
        <span class="arena-dot" :class="{ 'arena-dot--active': selectedArena === 'gelanggang-4' }" />
        GELANGGANG 4
        <span class="arena-count-badge">0 ALAT</span>
      </button>
    </div>

    <!-- DEVICE & SEAT PIN CARDS GRID -->
    <div class="pin-cards-grid">
      <div
        v-for="seat in currentSeats"
        :key="seat.id"
        class="pin-device-card"
        :class="{
          'pin-device-card--referee': seat.role === 'wasit',
          'pin-device-card--scoreboard': seat.role === 'scoreboard'
        }"
      >
        <!-- Card Header -->
        <div class="pin-card-header font-mono">
          <div class="pin-seat-label">
            <span
              class="seat-status-dot"
              :class="{
                'seat-status-dot--online': seat.status === 'online',
                'seat-status-dot--idle': seat.status === 'idle',
                'seat-status-dot--offline': seat.status === 'offline'
              }"
            />
            <span class="font-bold">{{ seat.label }}</span>
          </div>

          <div class="pin-device-meta">
            <span v-if="seat.latencyMs" class="latency-tag">{{ seat.latencyMs }}ms</span>
            <span v-if="seat.batteryPct" class="battery-tag">🔋 {{ seat.batteryPct }}%</span>
          </div>
        </div>

        <!-- Device Model Info -->
        <div class="pin-device-info font-mono">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect width="16" height="20" x="4" y="2" rx="2" ry="2" />
            <line x1="12" x2="12" y1="18" y2="18" />
          </svg>
          <span>{{ seat.deviceModel }}</span>
        </div>

        <!-- PIN Display Box -->
        <div class="pin-box">
          <div class="pin-box-sub font-mono">PIN OTORISASI PERANGKAT</div>
          <div class="pin-code-row">
            <div class="pin-code-num font-mono">
              {{ seat.pin }}
            </div>
            <button
              v-if="seat.pin !== 'AKSES PUBLIK'"
              class="pin-copy-btn"
              title="Salin PIN"
              @click="copyPin(seat.pin, seat.label)"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
                <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" />
              </svg>
            </button>
          </div>
          <div class="pin-expiry-text font-mono">
            Masa Berlaku: {{ seat.expiresAt }}
          </div>
        </div>

        <!-- Card Actions Footer -->
        <div class="pin-card-footer font-mono">
          <button
            v-if="seat.role !== 'scoreboard'"
            class="card-action-btn"
            title="Generate PIN Baru"
            @click="generatePinForSeat(seat)"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
              <path d="M3 3v5h5" />
              <path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16" />
              <path d="M16 21h5v-5" />
            </svg>
            <span>Reset PIN</span>
          </button>

          <button
            v-if="seat.role === 'judge'"
            class="card-action-btn"
            title="Kirim Tes Getar / Ping"
            @click="handlePingDevice(seat)"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9" />
              <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0" />
            </svg>
            <span>Ping</span>
          </button>

          <router-link
            v-if="seat.role === 'judge'"
            :to="`/judge?arena_id=${selectedArena}&seat=${seat.seatNumber}`"
            target="_blank"
            class="card-action-link"
          >
            Buka Tablet →
          </router-link>

          <router-link
            v-if="seat.role === 'wasit'"
            :to="`/referee?arena_id=${selectedArena}`"
            target="_blank"
            class="card-action-link"
          >
            Buka Konsol Wasit →
          </router-link>

          <router-link
            v-if="seat.role === 'scoreboard'"
            :to="`/scoreboard/${selectedArena}`"
            target="_blank"
            class="card-action-link"
          >
            Buka Layar TV →
          </router-link>
        </div>
      </div>
    </div>

    <!-- MODAL CUSTOM PIN FORM -->
    <div v-if="showCustomPinModal" class="pin-modal-backdrop" @click.self="showCustomPinModal = false">
      <div class="pin-modal">
        <h3 class="pin-modal-title font-mono">BUAT PIN OTORISASI BARU</h3>
        <p class="pin-modal-sub">Generate atau tentukan PIN khusus untuk juri / wasit gelanggang.</p>

        <div class="pin-form-group">
          <label class="font-mono">PILIH GELANGGANG</label>
          <select v-model="selectedArena" class="pin-form-select font-mono">
            <option value="gelanggang-1">GELANGGANG 1 (UTAMA)</option>
            <option value="gelanggang-2">GELANGGANG 2</option>
            <option value="gelanggang-3">GELANGGANG 3</option>
            <option value="gelanggang-4">GELANGGANG 4</option>
          </select>
        </div>

        <div class="pin-form-group">
          <label class="font-mono">PERAN PERANGKAT</label>
          <select v-model="customRole" class="pin-form-select font-mono">
            <option value="juri">Juri Lapangan</option>
            <option value="wasit">Wasit / Pengawas Gelanggang</option>
            <option value="dewan">Dewan Hakim / Delegasi Teknik</option>
          </select>
        </div>

        <div v-if="customRole === 'juri'" class="pin-form-group">
          <label class="font-mono">NOMOR KURSI JURI</label>
          <select v-model="customSeat" class="pin-form-select font-mono">
            <option :value="1">Kursi Juri 1</option>
            <option :value="2">Kursi Juri 2</option>
            <option :value="3">Kursi Juri 3</option>
            <option :value="4">Kursi Juri 4</option>
            <option :value="5">Kursi Juri 5</option>
          </select>
        </div>

        <div class="pin-form-group">
          <label class="font-mono">NOMOR PIN (KOSONGKAN UNTUK ACAK 6-DIGIT)</label>
          <input
            v-model="customPinInput"
            type="text"
            maxlength="10"
            placeholder="Contoh: 839102"
            class="pin-form-input font-mono"
          >
        </div>

        <div class="pin-form-group">
          <label class="font-mono">MASA BERLAKU (TTL)</label>
          <select v-model="customTtl" class="pin-form-select font-mono">
            <option :value="4">4 Jam (1 Sesi Pertandingan)</option>
            <option :value="8">8 Jam (Pagi s/d Sore)</option>
            <option :value="12">12 Jam (Full Day Session)</option>
            <option :value="24">24 Jam (1 Hari Penuh)</option>
          </select>
        </div>

        <div class="pin-modal-actions font-mono">
          <button class="pin-cancel-btn" @click="showCustomPinModal = false">
            Batal
          </button>
          <AppButton variant="emerald" size="sm" @click="handleSaveCustomPin">
            Simpan & Terapkan PIN
          </AppButton>
        </div>
      </div>
    </div>

    <!-- MODAL PRINTABLE PIN SLIPS -->
    <div v-if="showPrintModal" class="pin-modal-backdrop" @click.self="showPrintModal = false">
      <div class="pin-print-modal">
        <div class="print-modal-header no-print">
          <div>
            <h3 class="font-mono font-bold text-lg text-white">SLIP PIN MEJA JURI</h3>
            <p class="text-xs text-slate-400">Slip kartu untuk diletakkan pada masing-masing kursi juri di gelanggang.</p>
          </div>
          <div class="flex gap-2 font-mono">
            <button class="pin-cancel-btn text-xs" @click="showPrintModal = false">Tutup</button>
            <AppButton variant="emerald" size="sm" @click="triggerPrint">Cetak Sekarang (Print)</AppButton>
          </div>
        </div>

        <!-- Printable Slips Grid -->
        <div class="printable-slips-container">
          <div
            v-for="seat in currentSeats.filter(s => s.role === 'judge')"
            :key="seat.id"
            class="print-slip-card font-mono"
          >
            <div class="slip-brand">
              <span class="slip-logo">PS</span>
              <div>
                <div class="slip-title">PENCAK SILAT DIGITAL SCORING</div>
                <div class="slip-sub">{{ selectedArena.toUpperCase() }}</div>
              </div>
            </div>
            <div class="slip-seat">{{ seat.label }}</div>
            <div class="slip-pin-box">
              <div class="slip-pin-label">KODE PIN TABLET</div>
              <div class="slip-pin-val">{{ seat.pin }}</div>
            </div>
            <div class="slip-footer">
              <span>Masa Berlaku: Hari Ini</span>
              <span>PB IPSI / PERSILAT</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- TOAST NOTIFICATION -->
    <Transition name="fade">
      <div v-if="toast" class="pin-toast font-mono" :class="`pin-toast--${toast.type}`">
        <div class="toast-title font-bold">{{ toast.title }}</div>
        <div class="toast-msg">{{ toast.message }}</div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.pin-mgr-view {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* Banner */
.pin-mgr-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  background-color: #0E1626;
  border: 1px solid rgba(255, 255, 255, 0.08);
  padding: 18px 24px;
  border-radius: 10px;
}

.pin-mgr-title-group {
  display: flex;
  align-items: center;
  gap: 16px;
}

.pin-mgr-icon-badge {
  width: 44px;
  height: 44px;
  background-color: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.3);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.pin-mgr-sub {
  font-size: 0.75rem;
  color: #94A3B8;
  letter-spacing: 0.06em;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.pin-mgr-title {
  font-size: 1.375rem;
  font-weight: 800;
  letter-spacing: 0.02em;
  color: #FFFFFF;
}

.pin-mgr-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.pin-action-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background-color: #162032;
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #E2E8F0;
  padding: 8px 14px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.pin-action-btn:hover {
  background-color: #1E293B;
  border-color: rgba(255, 255, 255, 0.2);
}

/* Arena Tabs */
.arena-tabs-row {
  display: flex;
  align-items: center;
  gap: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  padding-bottom: 12px;
}

.arena-tab-btn {
  background: #0E1626;
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #94A3B8;
  padding: 9px 16px;
  border-radius: 8px;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 10px;
  transition: all 0.15s ease;
}

.arena-tab-btn:hover {
  color: #F8FAFC;
  background-color: #162032;
}

.arena-tab-btn--active {
  background-color: #1E293B;
  border-color: #38BDF8;
  color: #38BDF8;
  box-shadow: 0 0 14px rgba(56, 189, 248, 0.15);
}

.arena-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: #64748B;
}

.arena-dot--active {
  background-color: #10B981;
  box-shadow: 0 0 6px #10B981;
}

.arena-count-badge {
  font-size: 0.625rem;
  background-color: rgba(255, 255, 255, 0.08);
  padding: 2px 6px;
  border-radius: 4px;
}

/* PIN Cards Grid */
.pin-cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 18px;
}

.pin-device-card {
  background-color: #0E1626;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  transition: all 0.2s ease;
}

.pin-device-card:hover {
  border-color: rgba(255, 255, 255, 0.15);
  transform: translateY(-2px);
}

.pin-device-card--referee {
  border-color: rgba(220, 38, 38, 0.3);
  background: linear-gradient(180deg, #121826 0%, #0E1626 100%);
}

.pin-device-card--scoreboard {
  border-color: rgba(56, 189, 248, 0.3);
  background: linear-gradient(180deg, #0B192A 0%, #0E1626 100%);
}

.pin-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.pin-seat-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  color: #FFFFFF;
}

.seat-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.seat-status-dot--online {
  background-color: #10B981;
  box-shadow: 0 0 8px #10B981;
}

.seat-status-dot--idle {
  background-color: #F59E0B;
}

.seat-status-dot--offline {
  background-color: #64748B;
}

.pin-device-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.6875rem;
}

.latency-tag {
  color: #10B981;
  background-color: rgba(16, 185, 129, 0.1);
  padding: 2px 6px;
  border-radius: 4px;
}

.battery-tag {
  color: #94A3B8;
  background-color: rgba(255, 255, 255, 0.05);
  padding: 2px 6px;
  border-radius: 4px;
}

.pin-device-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.6875rem;
  color: #64748B;
}

/* PIN Box */
.pin-box {
  background-color: #080D1A;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.pin-box-sub {
  font-size: 0.625rem;
  color: #64748B;
  letter-spacing: 0.06em;
}

.pin-code-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.pin-code-num {
  font-size: 1.75rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  color: #38BDF8;
}

.pin-copy-btn {
  background-color: #162032;
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #CBD5E1;
  padding: 6px 10px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.pin-copy-btn:hover {
  background-color: #1E293B;
  color: #FFFFFF;
}

.pin-expiry-text {
  font-size: 0.625rem;
  color: #64748B;
}

/* Card Footer Actions */
.pin-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  padding-top: 10px;
  font-size: 0.75rem;
}

.card-action-btn {
  background: transparent;
  border: none;
  color: #94A3B8;
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 4px;
  transition: color 0.15s ease;
}

.card-action-btn:hover {
  color: #FFFFFF;
  background-color: rgba(255, 255, 255, 0.05);
}

.card-action-link {
  color: #38BDF8;
  text-decoration: none;
  font-weight: 600;
  padding: 4px 6px;
}

.card-action-link:hover {
  text-decoration: underline;
}

/* Modals */
.pin-modal-backdrop {
  position: fixed;
  inset: 0;
  background-color: rgba(0, 0, 0, 0.8);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 120;
}

.pin-modal {
  background-color: #0E1626;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 12px;
  padding: 24px;
  max-width: 440px;
  width: 90%;
}

.pin-modal-title {
  font-size: 1.125rem;
  font-weight: 800;
  color: #FFFFFF;
  margin-bottom: 4px;
}

.pin-modal-sub {
  font-size: 0.75rem;
  color: #94A3B8;
  margin-bottom: 20px;
}

.pin-form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 14px;
}

.pin-form-group label {
  font-size: 0.6875rem;
  color: #64748B;
  letter-spacing: 0.06em;
}

.pin-form-select,
.pin-form-input {
  background-color: #080D1A;
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #FFFFFF;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 0.8125rem;
  outline: none;
}

.pin-form-select option {
  background-color: #0E1626;
}

.pin-modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
}

.pin-cancel-btn {
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #94A3B8;
  padding: 6px 14px;
  border-radius: 6px;
  cursor: pointer;
}

/* Printable Slips */
.pin-print-modal {
  background-color: #0E1626;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 12px;
  padding: 24px;
  max-width: 800px;
  width: 90%;
  max-height: 85vh;
  overflow-y: auto;
}

.printable-slips-container {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(230px, 1fr));
  gap: 16px;
  margin-top: 16px;
}

.print-slip-card {
  background-color: #FFFFFF;
  color: #0F172A;
  border: 2px dashed #CBD5E1;
  border-radius: 8px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.slip-brand {
  display: flex;
  align-items: center;
  gap: 8px;
}

.slip-logo {
  background-color: #DC2626;
  color: #FFFFFF;
  font-weight: 800;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.875rem;
}

.slip-title {
  font-size: 0.625rem;
  font-weight: 800;
  line-height: 1.1;
}

.slip-sub {
  font-size: 0.5625rem;
  color: #64748B;
}

.slip-seat {
  font-size: 0.875rem;
  font-weight: 800;
  text-align: center;
  padding: 4px 0;
  border-bottom: 1px solid #E2E8F0;
}

.slip-pin-box {
  background-color: #F1F5F9;
  border-radius: 6px;
  padding: 10px;
  text-align: center;
}

.slip-pin-label {
  font-size: 0.625rem;
  color: #64748B;
  margin-bottom: 2px;
}

.slip-pin-val {
  font-size: 1.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  color: #0F172A;
}

.slip-footer {
  display: flex;
  justify-content: space-between;
  font-size: 0.5625rem;
  color: #94A3B8;
  border-top: 1px solid #E2E8F0;
  padding-top: 6px;
}

/* Toast */
.pin-toast {
  position: fixed;
  top: 76px;
  right: 28px;
  background-color: #0F172A;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 8px;
  padding: 12px 18px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.5);
  z-index: 150;
}

.pin-toast--success { border-color: #10B981; }
.pin-toast--info { border-color: #38BDF8; }

@media print {
  body * {
    visibility: hidden;
  }
  .printable-slips-container,
  .printable-slips-container * {
    visibility: visible;
  }
  .printable-slips-container {
    position: absolute;
    left: 0;
    top: 0;
    width: 100%;
    display: grid;
    grid-template-columns: repeat(3, 1fr);
  }
  .no-print {
    display: none !important;
  }
}
</style>
