import { api, must, type Json } from './http.ts'
import type { Devices, MatchKit, MatchRef } from './kit.ts'
import type { ArenaSocket, Msg } from './socket.ts'

/** Meminta snapshot terkini lewat `sync_request` dan mengembalikan payload `sync_response`. */
export async function snapshot(sock: ArenaSocket): Promise<Json> {
  const since = sock.mark()
  sock.send('sync_request')
  const m = await sock.waitFor('sync_response', { since, what: 'sync_response' })
  return m.payload
}

/** Menunggu `state_change` dengan state tertentu (mis. "round_1") pada soket. */
export function waitState(
  sock: ArenaSocket,
  state: string,
  since = 0,
  timeout = 6000,
): Promise<Msg> {
  return sock.waitFor((m) => m.type === 'state_change' && m.payload?.state === state, {
    since,
    timeout,
    what: `state_change → ${state}`,
  })
}

/** Menunggu `state_change` yang cocok predikat pada payload. */
export function waitStateWhere(
  sock: ArenaSocket,
  pred: (p: Json) => boolean,
  what: string,
  since = 0,
  timeout = 6000,
): Promise<Msg> {
  return sock.waitFor((m) => m.type === 'state_change' && pred(m.payload), { since, timeout, what })
}

/** Menjalankan `fn` untuk setiap soket secara paralel; hasil berurutan sesuai soket. */
export const eachSocket = <T>(
  socks: ArenaSocket[],
  fn: (s: ArenaSocket, i: number) => Promise<T>,
): Promise<T[]> => Promise.all(socks.map((s, i) => fn(s, i)))

/** Bidang `state_change` yang harus identik di semua perangkat (tanpa jam server yang berbeda per pesan). */
export const stateView = (p: Json) => ({
  state: p.state,
  round: p.round,
  timer_running: p.timer_running,
  timer_ends_at_ms: p.timer_ends_at_ms,
  tie_pending: p.tie_pending,
})

/**
 * Snapshot tanpa field yang memang berbeda per pesan/perangkat (jam server dan kehadiran), supaya snapshot dari
 * perangkat berbeda dapat dibandingkan sama persis.
 */
export function stable(snap: Json): Json {
  const copy = { ...snap }
  delete copy.server_time
  delete copy.server_time_ms
  delete copy.presence
  return copy
}

/** Selisih mutlak ≤ toleransi. */
export const within = (actual: number, expected: number, tolerance: number) =>
  Math.abs(actual - expected) <= tolerance

/** Partai baru + dimuat ke gelanggang + perangkat terhubung; siap `start`. */
export interface Arrangement {
  match: MatchRef
  devices: Devices
  /** Penanda posisi pesan juri; dipakai tes UI untuk menunggu siaran SETELAH aksi di layar. */
  judgeMarks?: number[]
}

export async function arrange(kit: MatchKit): Promise<Arrangement> {
  const match = await kit.createMatch()
  const devices = await kit.openDevices()
  await must('load-arena', kit.loadMatch(match))
  // semua perangkat menerima snapshot partai baru (siaran saat dimuat)
  await eachSocket(devices.all, (s) =>
    s.waitFor((m) => m.type === 'sync_response' && m.payload?.match_id === match.id, {
      what: 'snapshot partai baru',
    }),
  )
  return { match, devices }
}

/** Pembersihan per tes: tutup soket dan batalkan partai agar gelanggang bebas untuk tes berikutnya. */
export async function release(kit: MatchKit, a: Arrangement | undefined): Promise<void> {
  if (!a) return
  await a.devices.closeAll()
  await kit.control(a.match, 'cancel')
}

export async function matchRecord(kit: MatchKit, id: string): Promise<Json> {
  return must(
    'ambil partai',
    api('GET', `/competitions/${kit.competitionId}/matches/${id}`, kit.adminToken),
  )
}

/** Dua juri menekan teknik yang sama pada sudut yang sama → konsensus tercapai (poin sah). */
export async function confirmScore(
  devices: Devices,
  target: 'red' | 'blue',
  scoreType = 'technique',
  seats: [number, number] = [0, 1],
): Promise<void> {
  const tag = `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`
  const [a, b] = seats.map((i) => devices.judges[i])
  a.send('judge_input', { event_id: `${tag}-a`, score_type: scoreType, target, round: 1 })
  b.send('judge_input', { event_id: `${tag}-b`, score_type: scoreType, target, round: 1 })
}
