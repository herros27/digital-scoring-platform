import { sleep, wsOrigin } from './http.ts';
export class HandshakeRejected extends Error {
    constructor(label) {
        super(`WebSocket ${label} ditolak saat handshake`);
        this.name = 'HandshakeRejected';
    }
}
/**
 * Satu koneksi WebSocket ke gelanggang, merekam semua pesan agar bisa di-assert berurutan.
 * Memakai `WebSocket` global (browser/Cypress dan Node 22) lewat proxy Vite `/ws`, seperti klien nyata.
 */
export class ArenaSocket {
    messages = [];
    label;
    ws;
    waiters = [];
    closedFlag = false;
    closeCode = null;
    constructor(label, ws) {
        this.label = label;
        this.ws = ws;
        ws.onmessage = (ev) => {
            try {
                const m = JSON.parse(String(ev.data));
                this.messages.push({
                    type: m.type,
                    payload: m.payload,
                    from: m.from,
                    seq: this.messages.length,
                    at: Date.now(),
                });
                this.waiters.forEach((w) => w());
            }
            catch {
                // pesan non-JSON diabaikan
            }
        };
        ws.onclose = (ev) => {
            this.closedFlag = true;
            this.closeCode = ev.code;
            this.waiters.forEach((w) => w());
        };
    }
    /** Membuka koneksi; melempar {@link HandshakeRejected} bila server menolak handshake (401/403/404). */
    static open(arenaId, token, label, timeoutMs = 8000) {
        return new Promise((resolve, reject) => {
            const query = token ? `?token=${encodeURIComponent(token)}` : '';
            const ws = new WebSocket(`${wsOrigin()}/ws/arena/${arenaId}${query}`);
            let settled = false;
            const timer = setTimeout(() => {
                if (settled)
                    return;
                settled = true;
                try {
                    ws.close();
                }
                catch {
                    // sudah tertutup
                }
                reject(new Error(`WebSocket ${label}: timeout membuka koneksi`));
            }, timeoutMs);
            ws.onopen = () => {
                if (settled)
                    return;
                settled = true;
                clearTimeout(timer);
                resolve(new ArenaSocket(label, ws));
            };
            const refused = () => {
                if (settled)
                    return;
                settled = true;
                clearTimeout(timer);
                reject(new HandshakeRejected(label));
            };
            ws.onerror = refused;
            ws.onclose = refused;
        });
    }
    get isOpen() {
        return !this.closedFlag && this.ws.readyState === WebSocket.OPEN;
    }
    send(type, payload = {}) {
        this.ws.send(JSON.stringify({ type, payload }));
    }
    /** Mengirim teks mentah (mis. bukan JSON) untuk menguji ketahanan server terhadap pesan rusak. */
    sendRaw(text) {
        this.ws.send(text);
    }
    /** Tandai posisi saat ini; pesan sebelum penanda diabaikan oleh `waitFor`/`of` bila `since` dipakai. */
    mark() {
        return this.messages.length;
    }
    of(type, since = 0) {
        return this.messages.slice(since).filter((m) => m.type === type);
    }
    /** Menunggu pesan yang cocok (tipe atau predikat); gagal dengan ringkasan pesan bila waktu habis. */
    async waitFor(match, opts = {}) {
        const test = typeof match === 'string' ? (m) => m.type === match : match;
        const since = opts.since ?? 0;
        const deadline = Date.now() + (opts.timeout ?? 6000);
        for (;;) {
            const found = this.messages.slice(since).find(test);
            if (found)
                return found;
            const left = deadline - Date.now();
            if (left <= 0 || this.closedFlag) {
                const seen = this.messages
                    .slice(since)
                    .map((m) => m.type)
                    .join(', ') || '(tidak ada pesan)';
                throw new Error(`[${this.label}] menunggu ${opts.what ?? (typeof match === 'string' ? match : 'pesan')} gagal; ` +
                    `${this.closedFlag ? 'koneksi tertutup' : 'waktu habis'}. Pesan diterima: ${seen}`);
            }
            await new Promise((resolve) => {
                const t = setTimeout(resolve, Math.min(left, 250));
                this.waiters.push(() => {
                    clearTimeout(t);
                    resolve();
                });
            });
            this.waiters = [];
        }
    }
    /** Memastikan TIDAK ada pesan bertipe tertentu selama `ms` (bukti negatif, mis. tidak ada siaran). */
    async expectNone(type, ms, since = this.mark()) {
        await sleep(ms);
        const test = typeof type === 'string' ? (m) => m.type === type : type;
        const hit = this.messages.slice(since).find(test);
        if (hit)
            throw new Error(`[${this.label}] tidak boleh menerima ${hit.type}: ${JSON.stringify(hit.payload)}`);
    }
    async close() {
        if (!this.closedFlag) {
            try {
                this.ws.close();
            }
            catch {
                // sudah tertutup
            }
        }
        const deadline = Date.now() + 1500;
        while (!this.closedFlag && Date.now() < deadline)
            await sleep(25);
    }
    /** Memutus koneksi secara paksa dari sisi klien (mensimulasikan jaringan putus). */
    async drop() {
        await this.close();
    }
}
