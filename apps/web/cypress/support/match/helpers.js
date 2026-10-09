import { api, must } from './http.ts';
/** Meminta snapshot terkini lewat `sync_request` dan mengembalikan payload `sync_response`. */
export async function snapshot(sock) {
    const since = sock.mark();
    sock.send('sync_request');
    const m = await sock.waitFor('sync_response', { since, what: 'sync_response' });
    return m.payload;
}
/** Menunggu `state_change` dengan state tertentu (mis. "round_1") pada soket. */
export function waitState(sock, state, since = 0, timeout = 6000) {
    return sock.waitFor((m) => m.type === 'state_change' && m.payload?.state === state, {
        since,
        timeout,
        what: `state_change → ${state}`,
    });
}
/** Menunggu `state_change` yang cocok predikat pada payload. */
export function waitStateWhere(sock, pred, what, since = 0, timeout = 6000) {
    return sock.waitFor((m) => m.type === 'state_change' && pred(m.payload), { since, timeout, what });
}
/** Menjalankan `fn` untuk setiap soket secara paralel; hasil berurutan sesuai soket. */
export const eachSocket = (socks, fn) => Promise.all(socks.map((s, i) => fn(s, i)));
/** Bidang `state_change` yang harus identik di semua perangkat (tanpa jam server yang berbeda per pesan). */
export const stateView = (p) => ({
    state: p.state,
    round: p.round,
    timer_running: p.timer_running,
    timer_ends_at_ms: p.timer_ends_at_ms,
    tie_pending: p.tie_pending,
});
/**
 * Snapshot tanpa field yang memang berbeda per pesan/perangkat (jam server dan kehadiran), supaya snapshot dari
 * perangkat berbeda dapat dibandingkan sama persis.
 */
export function stable(snap) {
    const copy = { ...snap };
    delete copy.server_time;
    delete copy.server_time_ms;
    delete copy.presence;
    return copy;
}
/** Selisih mutlak ≤ toleransi. */
export const within = (actual, expected, tolerance) => Math.abs(actual - expected) <= tolerance;
export async function arrange(kit) {
    const match = await kit.createMatch();
    const devices = await kit.openDevices();
    await must('load-arena', kit.loadMatch(match));
    // semua perangkat menerima snapshot partai baru (siaran saat dimuat)
    await eachSocket(devices.all, (s) => s.waitFor((m) => m.type === 'sync_response' && m.payload?.match_id === match.id, {
        what: 'snapshot partai baru',
    }));
    return { match, devices };
}
/** Pembersihan per tes: tutup soket dan batalkan partai agar gelanggang bebas untuk tes berikutnya. */
export async function release(kit, a) {
    if (!a)
        return;
    await a.devices.closeAll();
    await kit.control(a.match, 'cancel');
}
export async function matchRecord(kit, id) {
    return must('ambil partai', api('GET', `/competitions/${kit.competitionId}/matches/${id}`, kit.adminToken));
}
/** Dua juri menekan teknik yang sama pada sudut yang sama → konsensus tercapai (poin sah). */
export async function confirmScore(devices, target, scoreType = 'technique', seats = [0, 1]) {
    const tag = `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`;
    const [a, b] = seats.map((i) => devices.judges[i]);
    a.send('judge_input', { event_id: `${tag}-a`, score_type: scoreType, target, round: 1 });
    b.send('judge_input', { event_id: `${tag}-b`, score_type: scoreType, target, round: 1 });
}
