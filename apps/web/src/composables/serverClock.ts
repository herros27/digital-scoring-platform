/**
 * Jam server di sisi klien. Setiap pesan server membawa `server_time_ms`; selisihnya dengan jam
 * klien saat pesan diterima = selisih jam − tundaan jaringan. Tundaan selalu mengurangi sampel,
 * jadi sampel terbesar dalam jendela terakhir adalah yang paling mendekati selisih jam sebenarnya.
 * Dengan begitu semua layar menghitung waktu pertandingan dari tenggat yang sama pada jam yang sama.
 */
const WINDOW = 30
let samples: number[] = []

export const serverClock = {
  /** Catat satu pengamatan (`serverTimeMs` = jam server saat pesan dikirim). */
  observe(serverTimeMs: number, clientNowMs: number = Date.now()): void {
    if (!Number.isFinite(serverTimeMs)) return
    samples.push(serverTimeMs - clientNowMs)
    if (samples.length > WINDOW) samples.shift()
  },

  /** Perkiraan (jam server − jam klien) dalam ms; 0 bila belum ada pengamatan. */
  offset(): number {
    return samples.length ? Math.max(...samples) : 0
  },

  /** Waktu server saat ini (epoch ms) menurut perkiraan klien. */
  now(): number {
    return Date.now() + serverClock.offset()
  },

  reset(): void {
    samples = []
  },
}
