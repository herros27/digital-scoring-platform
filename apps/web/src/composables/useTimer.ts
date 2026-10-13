import { computed, getCurrentInstance, onUnmounted, ref } from 'vue'
import { serverClock } from './serverClock'

export interface UseTimerOptions {
  /** Sumber waktu server (epoch ms). Bawaan: jam server hasil penyelarasan. Dapat diganti untuk tes. */
  now?: () => number
  /** Interval penyegaran tampilan (ms). */
  tickMs?: number
}

/**
 * Hitung mundur pertandingan yang ditampilkan di semua layar.
 *
 * Sisa waktu TIDAK dikurangi per ketukan interval (itu melenceng dan diperlambat browser pada
 * tab/jendela yang tidak fokus). Sebaliknya server mengirim tenggat absolut (`endsAtMs`) dan
 * setiap tampilan menghitung `tenggat − jam server` pada tiap penyegaran, sehingga semua layar
 * menunjukkan angka yang sama dan langsung benar lagi setelah jendela kembali aktif.
 */
export function useTimer(options: UseTimerOptions = {}) {
  const now = options.now ?? serverClock.now
  const tickMs = options.tickMs ?? 100

  const remainingMs = ref(0)
  const isRunning = ref(false)
  let deadline = 0 // tenggat dalam waktu server (epoch ms)
  let ticker: ReturnType<typeof setInterval> | null = null

  function stopTicker() {
    if (ticker) {
      clearInterval(ticker)
      ticker = null
    }
  }

  function tick() {
    const left = deadline - now()
    if (left > 0) {
      remainingMs.value = left
      return
    }
    remainingMs.value = 0
    isRunning.value = false
    stopTicker()
  }

  /**
   * Selaraskan dengan server. `ms` = sisa waktu saat pesan dibuat; `endsAtMs` = tenggat absolut
   * (hanya saat berjalan). Tanpa `endsAtMs` (server lama) tenggat dihitung dari `ms` + jam sekarang.
   */
  function sync(ms: number, running: boolean, endsAtMs?: number) {
    stopTicker()
    isRunning.value = running
    if (!running) {
      remainingMs.value = Math.max(0, ms)
      return
    }
    deadline = endsAtMs && endsAtMs > 0 ? endsAtMs : now() + Math.max(0, ms)
    tick()
    if (isRunning.value) ticker = setInterval(tick, tickMs)
  }

  function pause() {
    if (isRunning.value) remainingMs.value = Math.max(0, deadline - now())
    isRunning.value = false
    stopTicker()
  }

  // Dibulatkan ke atas: 119,9 dtk tampil 02:00 (angka baru berganti setelah satu detik penuh berlalu)
  const totalSeconds = computed(() => Math.max(0, Math.ceil(remainingMs.value / 1000)))
  const minutes = computed(() => Math.floor(totalSeconds.value / 60))
  const seconds = computed(() => totalSeconds.value % 60)
  const formattedTime = computed(
    () => `${String(minutes.value).padStart(2, '0')}:${String(seconds.value).padStart(2, '0')}`,
  )

  if (getCurrentInstance()) onUnmounted(stopTicker)

  return { remainingMs, isRunning, minutes, seconds, formattedTime, sync, pause }
}
