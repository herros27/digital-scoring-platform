import { adminToken } from '../../support/match/auth.ts';
import { arrange, eachSocket, release, snapshot, stable, stateView, waitState, waitStateWhere, } from '../../support/match/helpers.ts';
import { api, must, sleep } from '../../support/match/http.ts';
import { buildKit } from '../../support/match/kit.ts';
import { runAsync } from '../../support/match/run.ts';
const nextKey = (() => {
    let n = 0;
    return (prefix) => `${prefix}-${Date.now().toString(36)}-${(n += 1)}`;
})();
/**
 * Flow SELURUH JURI (kursi 1-3) dan integrasi lintas role: aksi wasit tersinkron ke tiap juri, input juri
 * menjadi konsensus yang dilihat semua perangkat, dan juri tidak dapat melihat/mengubah apa yang bukan haknya.
 * Semua lewat WebSocket + REST nyata.
 */
describe('Juri - sinkronisasi realtime lintas role', () => {
    let kit;
    let a;
    before(() => runAsync(async () => {
        kit = await buildKit(await adminToken(), {
            label: 'Juri',
            roundSeconds: 8,
            breakSeconds: 3,
            totalRounds: 2,
        });
    }));
    after(() => runAsync(async () => kit?.teardown()));
    beforeEach(() => runAsync(async () => {
        a = await arrange(kit);
    }));
    afterEach(() => runAsync(async () => release(kit, a)));
    describe('informasi yang diterima setiap juri', () => {
        for (const seat of [1, 2, 3]) {
            it(`juri kursi ${seat} login PIN: token terikat ke gelanggang, role judge, dan kursi benar`, () => runAsync(async () => {
                const s = kit.judges[seat - 1];
                expect(s).to.include({
                    role: 'judge',
                    seat,
                    arenaId: kit.arenaId,
                    competitionId: kit.competitionId,
                });
                expect(s.token.split('.')).to.have.length(3);
            }));
        }
        it('ketiga juri menerima snapshot yang identik (ruleset, teknik, konsensus, partai) dan wasit melihat ketiganya online', () => runAsync(async () => {
            const { match, devices } = a;
            const snaps = await eachSocket(devices.judges, (s) => snapshot(s));
            snaps.forEach((p) => {
                expect(p.match_id).to.eq(match.id);
                expect(p.ruleset_name).to.be.a('string').and.have.length.greaterThan(0);
                expect(p.consensus).to.deep.include({ judges: 3, majority: 2 });
                expect(stable(p)).to.deep.eq(stable(snaps[0]));
            });
            const wasitView = await snapshot(devices.wasit);
            const online = wasitView.presence.judges
                .filter((j) => j.connected)
                .map((j) => j.seat);
            expect(online).to.include.members([1, 2, 3]);
        }));
    });
    describe('aksi wasit tersinkron ke seluruh juri', () => {
        it('start → pause → resume: ketiga juri menerima urutan dan nilai state_change yang sama persis', () => runAsync(async () => {
            const { match, devices } = a;
            const since = devices.all.map((s) => s.mark());
            await must('start', kit.control(match, 'start'));
            await eachSocket(devices.all, (s, i) => waitState(s, 'round_1', since[i]));
            await must('pause', kit.control(match, 'pause'));
            await eachSocket(devices.all, (s, i) => waitStateWhere(s, (p) => p.timer_running === false, 'jeda', since[i]));
            await must('resume', kit.control(match, 'resume'));
            await eachSocket(devices.all, (s, i) => s.waitFor((m) => m.type === 'state_change' &&
                m.payload.timer_running === true &&
                s.of('state_change', since[i]).length >= 3, { since: since[i], what: 'lanjut' }));
            const timelines = devices.judges.map((s, i) => s.of('state_change', since[i + 1]).map((m) => stateView(m.payload)));
            expect(timelines[0].map((v) => `${v.state}/${v.timer_running}`)).to.deep.eq([
                'round_1/true',
                'round_1/false',
                'round_1/true',
            ]);
            timelines.slice(1).forEach((t) => expect(t).to.deep.eq(timelines[0]));
            // wasit (pengirim aksi) juga menerima siaran yang sama: tidak ada perangkat yang tertinggal
            const w = devices.wasit.of('state_change', since[0]).map((m) => stateView(m.payload));
            expect(w).to.deep.eq(timelines[0]);
        }));
        it('jam hitung mundur tiap juri konsisten: tenggat absolut sama dan sisa waktu ≈ selisih terhadap jam server', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const snaps = await eachSocket(devices.judges, (s) => snapshot(s));
            expect(new Set(snaps.map((p) => p.timer_ends_at_ms)).size).to.eq(1);
            snaps.forEach((p) => {
                expect(Math.abs(p.timer_ends_at_ms - p.server_time_ms - p.timer_remaining_ms)).to.be.lessThan(60);
            });
        }));
        it('perubahan ronde otomatis (istirahat) sampai ke tiap juri dengan hitung mundur istirahat yang berjalan', () => runAsync(async () => {
            const { match, devices } = a;
            const since = devices.judges.map((s) => s.mark());
            await must('start', kit.control(match, 'start'));
            const breaks = await eachSocket(devices.judges, (s, i) => waitState(s, 'break_1', since[i], 14000));
            breaks.forEach((m) => {
                expect(m.payload).to.include({ round: 1, timer_running: true });
                expect(m.payload.timer_remaining_ms).to.be.within(2000, 3000);
                expect(m.payload.timer_ends_at_ms).to.be.greaterThan(m.payload.server_time_ms);
            });
            expect(new Set(breaks.map((m) => m.payload.timer_ends_at_ms)).size).to.eq(1);
        }, 120_000));
        it('partai selesai: tiap juri menerima match_result yang sama dan snapshot berstatus finished', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            const since = devices.judges.map((s) => s.mark());
            await must('finish', kit.control(match, 'finish', { winner: 'blue' }));
            const results = await eachSocket(devices.judges, (s, i) => s.waitFor('match_result', { since: since[i], what: 'match_result' }));
            results.slice(1).forEach((m) => expect(m.payload).to.deep.eq(results[0].payload));
            expect(results[0].payload.corner).to.eq('blue');
            // layar juri hanya mengikuti state_change: harus ada state finished (timer berhenti) sebelum hasil
            for (const [i, s] of devices.judges.entries()) {
                const seen = s.messages
                    .slice(since[i])
                    .filter((m) => m.type === 'state_change' || m.type === 'match_result');
                const finishedAt = seen.findIndex((m) => m.type === 'state_change' && m.payload.state === 'finished');
                expect(finishedAt, `juri ${i + 1} menerima state_change finished`).to.be.greaterThan(-1);
                expect(seen[finishedAt].payload.timer_running).to.eq(false);
                expect(finishedAt, 'finished mendahului match_result').to.be.lessThan(seen.findIndex((m) => m.type === 'match_result'));
            }
            for (const p of await eachSocket(devices.judges, (s) => snapshot(s)))
                expect(p.state).to.eq('finished');
        }));
    });
    describe('konsensus juri (2 dari 3 dalam 3 detik)', () => {
        const tap = (sock, target, type = 'technique', key = nextKey('e')) => sock.send('judge_input', {
            event_id: key,
            idempotency_key: key,
            score_type: type,
            target,
            round: 1,
        });
        it('satu juri saja belum menghasilkan poin: juri lain melihat kemajuan anonim (1 dari 2), wasit tidak', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.all.map((s) => s.mark());
            tap(devices.judges[0], 'red');
            const progress = await eachSocket(devices.judges, (s, i) => s.waitFor('consensus_progress', { since: since[i + 1], what: 'consensus_progress' }));
            progress.forEach((m) => {
                expect(m.payload).to.include({
                    target: 'red',
                    score_type: 'technique',
                    count: 1,
                    needed: 2,
                    round: 1,
                });
                expect(m.payload).to.not.have.any.keys('judge_id', 'seat', 'seat_number', 'from');
            });
            await devices.wasit.expectNone((m) => m.type === 'consensus_progress' ||
                m.type === 'score_confirmed' ||
                m.type === 'score_update', 500, since[0]);
        }));
        for (const [first, second] of [
            [0, 1],
            [1, 2],
            [0, 2],
        ]) {
            it(`juri ${first + 1} + juri ${second + 1} sepakat → poin sah dan semua perangkat menerima score_confirmed + score_update`, () => runAsync(async () => {
                const { match, devices } = a;
                await must('start', kit.control(match, 'start'));
                await waitState(devices.wasit, 'round_1');
                const since = devices.all.map((s) => s.mark());
                tap(devices.judges[first], 'red', 'knockdown');
                tap(devices.judges[second], 'red', 'knockdown');
                const confirmed = await eachSocket(devices.all, (s, i) => s.waitFor('score_confirmed', { since: since[i], what: 'score_confirmed' }));
                confirmed.forEach((m) => expect(m.payload).to.include({
                    score_type: 'knockdown',
                    target: 'red',
                    points: 2,
                    count: 2,
                    round: 1,
                }));
                const updates = await eachSocket(devices.all, (s, i) => s.waitFor('score_update', { since: since[i], what: 'score_update' }));
                updates.forEach((m) => expect(m.payload).to.include({ round: 1, red_technical: 2, blue_technical: 0 }));
                // juri yang tidak ikut menekan tetap melihat skor yang sama
                expect((await snapshot(devices.judges[3 - first - second])).red_total).to.eq(2);
            }));
        }
        it('poin berbeda per teknik: pukulan 1, tendangan 2, jatuhan 3 terakumulasi pada sudut yang benar', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            for (const [type, target] of [
                ['technique', 'red'],
                ['knockdown', 'blue'],
                ['fall', 'red'],
            ]) {
                const m = devices.wasit.mark();
                tap(devices.judges[0], target, type);
                tap(devices.judges[1], target, type);
                await devices.wasit.waitFor('score_update', { since: m, what: `score_update ${type}` });
            }
            const s = await snapshot(devices.judges[2]);
            expect(s.red_total).to.eq(4);
            expect(s.blue_total).to.eq(2);
        }));
        it('juri tidak sepakat (merah vs biru) → tidak ada poin dan tidak ada siaran skor', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.all.map((s) => s.mark());
            tap(devices.judges[0], 'red');
            tap(devices.judges[1], 'blue');
            await eachSocket(devices.all, (s, i) => s.expectNone((m) => m.type === 'score_confirmed' || m.type === 'score_update', 700, since[i]));
            const s = await snapshot(devices.judges[2]);
            expect(s.red_total + s.blue_total).to.eq(0);
        }));
        it('dua juri menekan di luar jendela 3 detik → tidak menjadi konsensus', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.wasit.mark();
            tap(devices.judges[0], 'red');
            await sleep(3300);
            tap(devices.judges[1], 'red');
            await devices.judges[2].waitFor((m) => m.type === 'consensus_progress' && m.payload.count === 1, { since: 0, what: 'progress ke-2 hanya 1' });
            await devices.wasit.expectNone('score_update', 500, since);
            expect((await snapshot(devices.wasit)).red_total).to.eq(0);
        }, 60_000));
        it('juri yang sama menekan dua kali (event berbeda atau event_id sama) hanya dihitung satu suara', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.wasit.mark();
            tap(devices.judges[0], 'red', 'technique', 'dup-1');
            tap(devices.judges[0], 'red', 'technique', 'dup-1');
            tap(devices.judges[0], 'red', 'technique', 'dup-2');
            await devices.wasit.expectNone('score_update', 700, since);
            expect((await snapshot(devices.wasit)).red_total).to.eq(0);
        }));
        it('jenis skor yang tidak ada di ruleset ditolak hanya ke juri pengirim (BAD_PAYLOAD)', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.all.map((s) => s.mark());
            tap(devices.judges[0], 'red', 'super_kick');
            const err = await devices.judges[0].waitFor('error', { since: since[1], what: 'error' });
            expect(err.payload.code).to.eq('BAD_PAYLOAD');
            await devices.judges[1].expectNone('error', 400, since[2]);
            await devices.wasit.expectNone('error', 100, since[0]);
        }));
    });
    describe('batas waktu input juri', () => {
        const tap = (sock, key) => sock.send('judge_input', { event_id: key, score_type: 'technique', target: 'red', round: 1 });
        it('sebelum partai dimulai: input ditolak (SCORING_NOT_ALLOWED) hanya ke pengirim', () => runAsync(async () => {
            const { devices } = a;
            const since = devices.all.map((s) => s.mark());
            tap(devices.judges[0], nextKey('pre'));
            const err = await devices.judges[0].waitFor('error', { since: since[1], what: 'error' });
            expect(err.payload.code).to.eq('SCORING_NOT_ALLOWED');
            expect(err.payload.message).to.match(/idle/);
            await devices.judges[1].expectNone('error', 300, since[2]);
        }));
        it('saat istirahat: input ditolak (SCORING_NOT_ALLOWED) dan skor tidak berubah', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.judges[0], 'break_1', 0, 14000);
            const since = devices.judges[0].mark();
            tap(devices.judges[0], nextKey('brk'));
            const err = await devices.judges[0].waitFor('error', { since, what: 'error' });
            expect(err.payload.code).to.eq('SCORING_NOT_ALLOWED');
            expect(err.payload.message).to.match(/break_1/);
            expect((await snapshot(devices.wasit)).red_total).to.eq(0);
        }, 60_000));
        it('setelah partai selesai: input ditolak dan tidak mengubah hasil akhir', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await must('finish', kit.control(match, 'finish', { winner: 'red' }));
            await devices.judges[0].waitFor('match_result');
            const since = devices.judges[0].mark();
            tap(devices.judges[0], nextKey('post'));
            const err = await devices.judges[0].waitFor('error', { since, what: 'error' });
            expect(err.payload).to.include({ code: 'SCORING_NOT_ALLOWED' });
            expect((await snapshot(devices.wasit)).state).to.eq('finished');
        }));
    });
    describe('hak akses juri di WebSocket', () => {
        for (const action of [
            'start',
            'pause',
            'resume',
            'finish',
            'cancel',
            'adjust_time',
            'reset_round',
        ]) {
            it(`juri tidak boleh mengirim state_change "${action}": FORBIDDEN ke pengirim, state tidak berubah`, () => runAsync(async () => {
                const { match, devices } = a;
                await must('start', kit.control(match, 'start'));
                await waitState(devices.wasit, 'round_1');
                const since = devices.all.map((s) => s.mark());
                devices.judges[1].send('state_change', { action, winner: 'red', delta_ms: -1000 });
                const err = await devices.judges[1].waitFor('error', { since: since[2], what: 'error' });
                expect(err.payload.code).to.eq('FORBIDDEN');
                await devices.wasit.expectNone((m) => m.type === 'state_change' || m.type === 'match_result', 500, since[0]);
                const s = await snapshot(devices.wasit);
                expect(s.state).to.eq('round_1');
                expect(s.timer_running).to.eq(true);
            }));
        }
        it('juri tidak boleh menjatuhkan sanksi atau mengirim sinyal arena', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.all.map((s) => s.mark());
            devices.judges[0].send('penalty', { target: 'red', type: 'teguran_1' });
            devices.judges[0].send('arena_signal', { signal: 'gong' });
            await devices.judges[0].waitFor((m) => m.type === 'error', { since: since[1] });
            await sleep(300);
            expect(devices.judges[0].of('error', since[1]).map((m) => m.payload.code)).to.deep.eq([
                'FORBIDDEN',
                'FORBIDDEN',
            ]);
            await devices.wasit.expectNone((m) => m.type === 'penalties' || m.type === 'arena_signal' || m.type === 'score_update', 300, since[0]);
        }));
    });
    describe('audit input juri', () => {
        it('setiap input juri tercatat sebagai kejadian dengan kursi dan jenis teknik', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const m = devices.wasit.mark();
            devices.judges[2].send('judge_input', {
                event_id: nextKey('aud'),
                score_type: 'fall',
                target: 'blue',
                round: 1,
            });
            await devices.wasit.waitFor((x) => x.type === 'match_event' && x.payload.event_type === 'judge_input', { since: m, what: 'match_event judge_input' });
            const events = await must('events', api('GET', `/competitions/${kit.competitionId}/matches/${match.id}/events`, kit.wasit.token));
            const found = events.find((e) => e.event_type === 'judge_input');
            // payload kejadian disimpan sebagai JSON (string pada respons REST)
            const payload = typeof found.payload === 'string' ? JSON.parse(found.payload) : found.payload;
            expect(payload).to.include({ seat_number: 3, score_type: 'fall', target: 'blue', round: 1 });
        }));
    });
    describe('skor saat dijeda dan pembatalan (BATAL)', () => {
        const press = (sock, target, type, key = nextKey('p')) => sock.send('judge_input', {
            event_id: key,
            idempotency_key: key,
            score_type: type,
            target,
            round: 1,
        });
        it('partai dijeda: input juri ditolak SCORING_NOT_ALLOWED dan tidak masuk ke skor, lalu diterima lagi setelah dilanjutkan', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            await must('pause', kit.control(match, 'pause'));
            await waitStateWhere(devices.judges[0], (p) => p.timer_running === false, 'dijeda');
            const since = devices.judges.map((s) => s.mark());
            press(devices.judges[0], 'red', 'technique');
            press(devices.judges[1], 'red', 'technique');
            const errors = await eachSocket(devices.judges.slice(0, 2), (s, i) => s.waitFor('error', { since: since[i], what: 'error SCORING_NOT_ALLOWED' }));
            errors.forEach((m) => expect(m.payload.code).to.eq('SCORING_NOT_ALLOWED'));
            await devices.wasit.expectNone('score_confirmed', 400);
            expect((await snapshot(devices.wasit)).red_total).to.eq(0);
            await must('resume', kit.control(match, 'resume'));
            await waitStateWhere(devices.judges[0], (p) => p.timer_running === true, 'lanjut', 0);
            const after = devices.all.map((s) => s.mark());
            press(devices.judges[0], 'red', 'technique');
            press(devices.judges[1], 'red', 'technique');
            await eachSocket(devices.all, (s, i) => s.waitFor('score_confirmed', { since: after[i], what: 'score_confirmed setelah lanjut' }));
            expect((await snapshot(devices.wasit)).red_total).to.eq(1);
        }));
        it('REST skor juri juga ditolak (409) saat dijeda', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            await must('pause', kit.control(match, 'pause'));
            const res = await api('POST', `/competitions/${kit.competitionId}/matches/${match.id}/scores`, kit.judges[0].token, { score_type: 'technique', target: 'red', idempotency_key: nextKey('rest') });
            expect(res.status).to.be.oneOf([400, 409, 422]);
        }));
        it('BATAL menarik input yang menunggu: lampu padam (count 0) dan juri lain sendirian tidak cukup untuk poin', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.judges.map((s) => s.mark());
            press(devices.judges[0], 'red', 'technique');
            await devices.judges[2].waitFor((m) => m.type === 'consensus_progress' && m.payload.count === 1, { since: since[2], what: 'progres 1' });
            press(devices.judges[0], 'red', 'undo');
            await devices.judges[2].waitFor((m) => m.type === 'consensus_progress' && m.payload.count === 0, { since: since[2], what: 'progres 0 setelah BATAL' });
            press(devices.judges[1], 'red', 'technique');
            await devices.wasit.expectNone('score_confirmed', 500);
            expect((await snapshot(devices.wasit)).red_total).to.eq(0);
        }));
        it('BATAL tanpa input menunggu → UNDO_NOT_POSSIBLE (bukan diam); target tidak valid → BAD_PAYLOAD', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const j = devices.judges[0];
            let m = j.mark();
            press(j, 'blue', 'undo');
            expect((await j.waitFor('error', { since: m, what: 'UNDO_NOT_POSSIBLE' })).payload.code).to.eq('UNDO_NOT_POSSIBLE');
            m = j.mark();
            press(j, 'ungu', 'undo');
            expect((await j.waitFor('error', { since: m, what: 'BAD_PAYLOAD' })).payload.code).to.eq('BAD_PAYLOAD');
        }));
        it('poin yang sudah terkonfirmasi tidak dapat dibatalkan juri (UNDO_NOT_POSSIBLE) dan skor tetap', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const m0 = devices.wasit.mark();
            press(devices.judges[0], 'red', 'knockdown');
            press(devices.judges[1], 'red', 'knockdown');
            await devices.wasit.waitFor('score_confirmed', { since: m0, what: 'score_confirmed' });
            const m = devices.judges[0].mark();
            press(devices.judges[0], 'red', 'undo');
            expect((await devices.judges[0].waitFor('error', { since: m, what: 'UNDO_NOT_POSSIBLE' }))
                .payload.code).to.eq('UNDO_NOT_POSSIBLE');
            expect((await snapshot(devices.wasit)).red_total).to.eq(2);
        }));
    });
});
