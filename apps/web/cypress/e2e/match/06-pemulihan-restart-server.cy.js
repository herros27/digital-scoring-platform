import { adminToken } from '../../support/match/auth.ts';
import { arrange, release, snapshot, waitState } from '../../support/match/helpers.ts';
import { must, sleep } from '../../support/match/http.ts';
import { buildKit } from '../../support/match/kit.ts';
import { restartConfigured, restartServer } from '../../support/match/restart.ts';
import { runAsync, runSeq } from '../../support/match/run.ts';
let n = 0;
const key = () => `rs-${Date.now().toString(36)}-${(n += 1)}`;
/**
 * Pemulihan setelah server restart di tengah partai. Server benar-benar dimatikan dan dinyalakan ulang lewat
 * E2E_RESTART_CMD (lihat support/match/restart.ts); tanpa itu spec ini dilewati. Yang diuji:
 * skor dan babak bertahan, jam TIDAK jalan sendiri (dipulihkan dalam keadaan dijeda), perangkat dapat
 * tersambung ulang dengan token lama, skor juri ditolak sampai wasit melanjutkan, lalu partai bisa selesai.
 */
const suite = restartConfigured() ? describe : describe.skip;
suite('Pemulihan setelah restart server', () => {
    let kit;
    let a;
    let before1;
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
            label: 'Restart',
            roundSeconds: 60,
            breakSeconds: 3,
            totalRounds: 2,
        });
    }));
    after(() => runAsync(async () => kit?.teardown()));
    afterEach(() => runAsync(async () => release(kit, a)));
    it('partai berjalan dengan skor 2-0: setelah restart tetap di babak 1 dengan skor sama, dijeda, dan dapat dilanjutkan sampai selesai', () => runSeq(async () => {
        a = await arrange(kit);
        const { match, devices } = a;
        await must('start', kit.control(match, 'start'));
        await waitState(devices.wasit, 'round_1');
        const m = devices.wasit.mark();
        tap(devices.judges[0], 'red', 'knockdown');
        tap(devices.judges[1], 'red', 'knockdown');
        await devices.wasit.waitFor('score_confirmed', { since: m, what: 'score_confirmed' });
        before1 = await snapshot(devices.wasit);
        expect(before1.red_total).to.eq(2);
        expect(before1.timer_running).to.eq(true);
    }, () => restartServer(), async () => {
        const { match } = a;
        await a.devices.closeAll(); // soket lama sudah mati bersama server
        // server baru bisa belum siap menerima soket sesaat setelah perintah restart kembali
        let devices;
        for (let attempt = 0;; attempt += 1) {
            try {
                devices = await kit.openDevices();
                break;
            }
            catch (e) {
                if (attempt >= 10)
                    throw e;
                await sleep(500);
            }
        }
        a = { match, devices };
        const snap = await snapshot(devices.wasit);
        expect(snap.match_id).to.eq(match.id);
        expect(snap.state).to.eq('round_1');
        expect(snap.red_total).to.eq(2);
        expect(snap.blue_total).to.eq(0);
        expect(snap.timer_running, 'jam tidak boleh berjalan sendiri setelah restart').to.eq(false);
        // sisa waktu dipulihkan dari simpanan terakhir (sedikit lebih besar bila ada detik sejak simpan), bukan dari jam dinding
        expect(snap.timer_remaining_ms).to.be.within(before1.timer_remaining_ms - 5000, kit.roundMs);
        // skor juri ditolak selama partai masih dijeda hasil pemulihan
        const j = devices.judges[0];
        const since = j.mark();
        tap(j, 'blue', 'technique');
        const err = await j.waitFor('error', { since, what: 'SCORING_NOT_ALLOWED setelah restart' });
        expect(err.payload.code).to.eq('SCORING_NOT_ALLOWED');
        // wasit melanjutkan → jam berjalan lagi dan skor juri diterima
        await must('resume', kit.control(match, 'resume'));
        await devices.wasit.waitFor((x) => x.type === 'state_change' && x.payload.timer_running === true, { what: 'jam berjalan setelah resume' });
        const m = devices.wasit.mark();
        tap(devices.judges[0], 'blue', 'technique');
        tap(devices.judges[1], 'blue', 'technique');
        await devices.wasit.waitFor('score_confirmed', { since: m, what: 'score_confirmed baru' });
        const after = await snapshot(devices.wasit);
        expect([after.red_total, after.blue_total]).to.deep.eq([2, 1]);
        await must('finish', kit.control(match, 'finish'));
        await waitState(devices.wasit, 'finished');
        const result = (await snapshot(devices.wasit)).result;
        expect(result.corner).to.eq('red');
    }));
});
