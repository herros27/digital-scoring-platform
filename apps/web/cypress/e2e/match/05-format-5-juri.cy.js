import { adminToken } from '../../support/match/auth.ts';
import { arrange, release, snapshot, waitState } from '../../support/match/helpers.ts';
import { must } from '../../support/match/http.ts';
import { buildKit } from '../../support/match/kit.ts';
import { runAsync } from '../../support/match/run.ts';
let n = 0;
const key = () => `f5-${Date.now().toString(36)}-${(n += 1)}`;
/**
 * Format 5 juri (kursi 1-5): ruleset mengharuskan 3 dari 5 juri sepakat dalam window. Memastikan angka
 * konsensus mengikuti ruleset (bukan 2-dari-3 yang tertanam), kelima juri dapat terhubung, dan lampu progres
 * juri menghitung sampai 3.
 */
describe('Format 5 juri - konsensus 3 dari 5', () => {
    let kit;
    let a;
    const tap = (sock, target, type = 'technique') => {
        const k = key();
        sock.send('judge_input', {
            event_id: k,
            idempotency_key: k,
            score_type: type,
            target,
            round: 1,
        });
    };
    before(() => runAsync(async () => {
        kit = await buildKit(await adminToken(), {
            label: 'Lima Juri',
            roundSeconds: 8,
            breakSeconds: 3,
            totalRounds: 2,
            judges: 5,
            majority: 3,
            consensusWindowMs: 3000,
        });
    }));
    after(() => runAsync(async () => kit?.teardown()));
    beforeEach(() => runAsync(async () => {
        a = await arrange(kit);
        await must('start', kit.control(a.match, 'start'));
        await waitState(a.devices.wasit, 'round_1');
    }));
    afterEach(() => runAsync(async () => release(kit, a)));
    it('kelima juri online dan menerima aturan konsensus 3 dari 5 di snapshot', () => runAsync(async () => {
        const { devices } = a;
        expect(devices.judges).to.have.length(5);
        const snaps = await Promise.all(devices.judges.map((s) => snapshot(s)));
        snaps.forEach((p) => {
            expect(p.consensus).to.deep.include({ judges: 5, majority: 3 });
            expect(p.judge_count).to.eq(5);
        });
        const online = (await snapshot(devices.wasit)).presence.judges
            .filter((j) => j.connected)
            .map((j) => j.seat);
        expect(online).to.have.members([1, 2, 3, 4, 5]);
    }));
    it('dua juri saja belum cukup: progres 1/3 lalu 2/3 tampil di tablet juri, tanpa poin', () => runAsync(async () => {
        const { devices } = a;
        const since = devices.judges.map((s) => s.mark());
        tap(devices.judges[0], 'red');
        tap(devices.judges[1], 'red');
        const last = await devices.judges[4].waitFor((m) => m.type === 'consensus_progress' && m.payload.count === 2, { since: since[4], what: 'progres 2/3' });
        expect(last.payload).to.include({ needed: 3, target: 'red' });
        await devices.wasit.expectNone('score_confirmed', 500);
        expect((await snapshot(devices.wasit)).red_total).to.eq(0);
    }));
    it('tiga juri mana pun (kursi 2, 4, 5) sepakat → satu poin sah, kelima juri dan wasit menerimanya', () => runAsync(async () => {
        const { devices } = a;
        const since = devices.all.map((s) => s.mark());
        for (const i of [1, 3, 4])
            tap(devices.judges[i], 'blue', 'knockdown');
        for (const [i, s] of devices.all.entries()) {
            const m = await s.waitFor('score_confirmed', { since: since[i], what: 'score_confirmed' });
            expect(m.payload).to.include({
                target: 'blue',
                score_type: 'knockdown',
                points: 2,
                count: 3,
            });
        }
        expect((await snapshot(devices.judges[0])).blue_total).to.eq(2);
    }));
    it('juri keempat dan kelima yang menyusul setelah konsensus tidak menambah poin kedua', () => runAsync(async () => {
        const { devices } = a;
        for (const i of [0, 1, 2])
            tap(devices.judges[i], 'red');
        await devices.wasit.waitFor('score_confirmed', { what: 'score_confirmed' });
        const before = devices.wasit.mark();
        tap(devices.judges[3], 'red');
        tap(devices.judges[4], 'red');
        // dua suara susulan baru 2/3 untuk putaran berikutnya: belum cukup
        await devices.wasit.expectNone('score_confirmed', 600, before);
        expect((await snapshot(devices.wasit)).red_total).to.eq(1);
    }));
    it('tiga juri menekan sudut berbeda tidak membentuk konsensus', () => runAsync(async () => {
        const { devices } = a;
        tap(devices.judges[0], 'red');
        tap(devices.judges[1], 'blue');
        tap(devices.judges[2], 'red', 'knockdown');
        await devices.wasit.expectNone('score_confirmed', 600);
        const s = await snapshot(devices.wasit);
        expect([s.red_total, s.blue_total]).to.deep.eq([0, 0]);
    }));
    it('suara di luar window 3 detik tidak digabung', () => runAsync(async () => {
        const { devices } = a;
        tap(devices.judges[0], 'red');
        tap(devices.judges[1], 'red');
        await new Promise((r) => setTimeout(r, 3300));
        tap(devices.judges[2], 'red');
        await devices.wasit.expectNone('score_confirmed', 500);
        expect((await snapshot(devices.wasit)).red_total).to.eq(0);
    }));
});
