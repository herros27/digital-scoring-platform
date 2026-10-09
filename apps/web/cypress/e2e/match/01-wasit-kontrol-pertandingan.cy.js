import { adminToken } from '../../support/match/auth.ts';
import { eachSocket, matchRecord, snapshot, stable, stateView, waitState, waitStateWhere, within, arrange, release, confirmScore, } from '../../support/match/helpers.ts';
import { api, must, sleep } from '../../support/match/http.ts';
import { buildKit } from '../../support/match/kit.ts';
import { runAsync } from '../../support/match/run.ts';
/**
 * Flow WASIT saat pertandingan dimulai (integrasi nyata REST + WebSocket + state machine + timer).
 * Wasit mengendalikan partai lewat REST; semua perangkat di gelanggang (wasit + 3 juri) menerima siaran
 * WebSocket yang sama. Ruleset uji: ronde 8 dtk, istirahat 3 dtk, 2 babak.
 */
describe('Wasit - kontrol pertandingan', () => {
    let kit;
    let a;
    before(() => runAsync(async () => {
        kit = await buildKit(await adminToken(), {
            label: 'Wasit',
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
    describe('memuat partai', () => {
        it('semua perangkat menerima snapshot idle yang konsisten dengan ruleset', () => runAsync(async () => {
            const { match, devices } = a;
            const snaps = await eachSocket(devices.all, (s) => snapshot(s));
            for (const p of snaps) {
                expect(p).to.include({
                    match_id: match.id,
                    arena_id: kit.arenaId,
                    state: 'idle',
                    round: 0,
                    timer_running: false,
                });
                expect(p.round_duration_ms).to.eq(8000);
                expect(p.break_duration_ms).to.eq(3000);
                expect(p.total_rounds).to.eq(2);
                expect(p.consensus).to.deep.include({ judges: 3, majority: 2, window_ms: 3000 });
                expect(p.techniques.map((t) => t.type)).to.deep.eq([
                    'technique',
                    'knockdown',
                    'fall',
                ]);
            }
            // semua perangkat melihat data yang identik (kecuali jam server per pesan dan kehadiran)
            snaps.slice(1).forEach((p) => expect(stable(p)).to.deep.eq(stable(snaps[0])));
        }));
        it('partai tersimpan idle di database sebelum dimulai', () => runAsync(async () => {
            expect((await matchRecord(kit, a.match.id)).status).to.eq('idle');
        }));
    });
    describe('memulai pertandingan', () => {
        it('wasit memulai: REST 200 dan semua perangkat menerima state_change round_1 dengan tenggat yang sama', () => runAsync(async () => {
            const { match, devices } = a;
            const since = devices.all.map((s) => s.mark());
            const res = await kit.control(match, 'start');
            expect(res.status).to.eq(200);
            expect(res.data).to.deep.eq({ status: 'running' });
            const changes = await eachSocket(devices.all, (s, i) => waitState(s, 'round_1', since[i]));
            const views = changes.map((m) => stateView(m.payload));
            views.forEach((v) => {
                expect(v).to.include({ state: 'round_1', round: 1, timer_running: true });
                expect(v.timer_ends_at_ms).to.be.greaterThan(Date.now() - 1000);
            });
            // tenggat absolut identik → semua layar menghitung mundur ke waktu yang sama
            views.slice(1).forEach((v) => expect(v).to.deep.eq(views[0]));
            expect(within(changes[0].payload.timer_remaining_ms, 8000, 600)).to.eq(true);
        }));
        it('status partai di database menjadi berjalan setelah dimulai', () => runAsync(async () => {
            await must('start', kit.control(a.match, 'start'));
            await waitState(a.devices.judges[0], 'round_1');
            expect(await matchRecord(kit, a.match.id))
                .to.have.property('status')
                .that.is.oneOf(['running', 'round_1']);
        }));
        it('memulai untuk kedua kali ditolak (INVALID_STATE_TRANSITION) tanpa siaran tambahan', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const before = devices.judges.map((s) => s.mark());
            const again = await kit.control(match, 'start');
            expect(again.status).to.eq(400);
            expect(again.error?.code).to.eq('INVALID_STATE_TRANSITION');
            await eachSocket(devices.judges, (s, i) => s.expectNone('state_change', 600, before[i]));
            expect((await snapshot(devices.wasit)).state).to.eq('round_1');
        }));
        it('memulai partai yang belum dimuat ke gelanggang ditolak', () => runAsync(async () => {
            const loose = await kit.createMatch();
            const res = await kit.control(loose, 'start');
            expect(res.status).to.eq(400);
            expect(res.error?.code).to.eq('INVALID_STATE_TRANSITION');
            expect(res.error?.message).to.match(/not loaded/i);
        }));
    });
    describe('validasi perubahan status sebelum partai dimulai', () => {
        for (const action of ['pause', 'resume']) {
            it(`${action} saat idle ditolak dan state tetap idle`, () => runAsync(async () => {
                const res = await kit.control(a.match, action);
                expect(res.status).to.eq(400);
                expect(res.error?.code).to.eq('INVALID_STATE_TRANSITION');
                expect((await snapshot(a.devices.wasit)).state).to.eq('idle');
            }));
        }
        it('finish saat idle ditolak: partai yang belum dimulai tidak boleh diberi pemenang', () => runAsync(async () => {
            const { match, devices } = a;
            const before = devices.all.map((s) => s.mark());
            for (const body of [undefined, { winner: 'red' }]) {
                const res = await kit.control(match, 'finish', body);
                expect(res.status).to.eq(400);
                expect(res.error?.code).to.eq('INVALID_STATE_TRANSITION');
            }
            await eachSocket(devices.all, (s, i) => s.expectNone('match_result', 500, before[i]));
            expect((await snapshot(devices.wasit)).state).to.eq('idle');
            expect((await matchRecord(kit, match.id)).winner_id).to.be.oneOf(['', null, undefined]);
        }));
    });
    describe('timer', () => {
        it('berjalan mundur, beku saat dijeda, dan berlanjut saat dilanjutkan', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const r1 = (await snapshot(devices.judges[0])).timer_remaining_ms;
            await sleep(900);
            const r2 = (await snapshot(devices.judges[0])).timer_remaining_ms;
            expect(r2).to.be.lessThan(r1 - 600);
            const since = devices.all.map((s) => s.mark());
            expect((await kit.control(match, 'pause')).status).to.eq(200);
            const paused = await eachSocket(devices.all, (s, i) => waitStateWhere(s, (p) => p.timer_running === false, 'jeda', since[i]));
            paused.forEach((m) => expect(m.payload).to.include({ state: 'round_1', timer_running: false }));
            const frozen = paused[0].payload.timer_remaining_ms;
            await sleep(900);
            expect((await snapshot(devices.judges[1])).timer_remaining_ms).to.eq(frozen);
            expect((await snapshot(devices.judges[1])).timer_running).to.eq(false);
            // daftar partai (UI panitia) membaca status dari database: jeda harus tercatat "paused"
            expect((await matchRecord(kit, match.id)).status).to.eq('paused');
            const since2 = devices.all.map((s) => s.mark());
            expect((await kit.control(match, 'resume')).status).to.eq(200);
            const resumed = await eachSocket(devices.all, (s, i) => waitStateWhere(s, (p) => p.timer_running === true, 'lanjut', since2[i]));
            const p = resumed[0].payload;
            expect(within(p.timer_ends_at_ms - p.server_time_ms, frozen, 400)).to.eq(true);
            expect((await matchRecord(kit, match.id)).status).to.eq('running');
            resumed
                .slice(1)
                .forEach((m) => expect(m.payload.timer_ends_at_ms).to.eq(p.timer_ends_at_ms));
        }));
        it('wasit mengoreksi waktu (-2 detik): semua perangkat menerima sisa waktu baru', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const before = (await snapshot(devices.wasit)).timer_remaining_ms;
            const since = devices.all.map((s) => s.mark());
            devices.wasit.send('state_change', { action: 'adjust_time', delta_ms: -2000 });
            const changes = await eachSocket(devices.all, (s, i) => waitState(s, 'round_1', since[i]));
            const ends = changes.map((m) => m.payload.timer_ends_at_ms);
            expect(new Set(ends).size).to.eq(1);
            expect(within(changes[0].payload.timer_remaining_ms, before - 2000, 500)).to.eq(true);
        }));
        it('server menyiarkan timer_sync berkala ke semua perangkat selama timer berjalan', () => runAsync(async () => {
            const { match, devices } = a;
            const since = devices.all.map((s) => s.mark());
            await must('start', kit.control(match, 'start'));
            const syncs = await eachSocket(devices.all, (s, i) => s.waitFor('timer_sync', { since: since[i], timeout: 7500, what: 'timer_sync' }));
            syncs.forEach((m) => {
                expect(m.payload.timer_running).to.eq(true);
                expect(m.payload.timer_ends_at_ms).to.be.greaterThan(Date.now());
            });
            expect(new Set(syncs.map((m) => m.payload.timer_ends_at_ms)).size).to.eq(1);
        }));
    });
    describe('pergantian ronde otomatis', () => {
        it('ronde 1 habis → istirahat → ronde 2, dengan urutan state identik di semua perangkat', () => runAsync(async () => {
            const { match, devices } = a;
            const since = devices.all.map((s) => s.mark());
            await must('start', kit.control(match, 'start'));
            await eachSocket(devices.all, (s, i) => waitState(s, 'round_2', since[i], 16000));
            const orders = devices.all.map((s, i) => s
                .of('state_change', since[i])
                .map((m) => m.payload.state)
                .filter((st, idx, arr) => st !== arr[idx - 1]));
            orders.forEach((o) => expect(o).to.deep.eq(['round_1', 'break_1', 'round_2']));
            const brk = devices.wasit
                .of('state_change', since[0])
                .find((m) => m.payload.state === 'break_1');
            expect(within(brk.payload.timer_remaining_ms, 3000, 600)).to.eq(true);
        }, 120_000));
        it('babak terakhir habis dengan skor unggul → partai selesai otomatis dan hasil disiarkan ke semua', () => runAsync(async () => {
            const { match, devices } = a;
            const since = devices.all.map((s) => s.mark());
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            await confirmScore(devices, 'red', 'technique');
            await devices.wasit.waitFor('score_update', { since: since[0], what: 'score_update' });
            const results = await eachSocket(devices.all, (s, i) => s.waitFor('match_result', { since: since[i], timeout: 30000, what: 'match_result' }));
            results.forEach((m) => expect(m.payload).to.include({ corner: 'red', result_type: 'point_win' }));
            const final = await snapshot(devices.judges[2]);
            expect(final.state).to.eq('finished');
            expect(final.red_total).to.eq(1);
            expect((await matchRecord(kit, match.id)).status).to.eq('finished');
        }, 150_000));
    });
    describe('mengakhiri pertandingan', () => {
        it('skor seri tanpa pemenang ditolak (TIE_NEEDS_DECISION), wasit lalu memutuskan pemenang', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const tie = await kit.control(match, 'finish');
            expect(tie.status).to.eq(400);
            expect(tie.error?.code).to.eq('TIE_NEEDS_DECISION');
            expect((await snapshot(devices.wasit)).state).to.eq('round_1');
            const since = devices.all.map((s) => s.mark());
            const res = await kit.control(match, 'finish', { winner: 'blue' });
            expect(res.status).to.eq(200);
            const results = await eachSocket(devices.all, (s, i) => s.waitFor('match_result', { since: since[i], what: 'match_result' }));
            results.forEach((m) => expect(m.payload.corner).to.eq('blue'));
            const rec = await matchRecord(kit, match.id);
            expect(rec.status).to.eq('finished');
            expect(rec.winner_id).to.eq(match.blueId);
        }));
        it('skor unggul tanpa pemenang eksplisit: hasil mengikuti poin sah juri', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.wasit.mark();
            await confirmScore(devices, 'blue', 'knockdown', [1, 2]);
            await devices.wasit.waitFor('score_update', { since, what: 'score_update' });
            await must('finish', kit.control(match, 'finish'));
            const result = await devices.wasit.waitFor('match_result', { since, what: 'match_result' });
            expect(result.payload).to.include({ corner: 'blue', result_type: 'point_win' });
            expect((await matchRecord(kit, match.id)).winner_id).to.eq(match.blueId);
        }));
        it('finish kedua kali ditolak dan tidak menyiarkan hasil kedua', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await must('finish', kit.control(match, 'finish', { winner: 'red' }));
            await devices.wasit.waitFor('match_result');
            const before = devices.all.map((s) => s.mark());
            const again = await kit.control(match, 'finish', { winner: 'blue' });
            expect(again.status).to.eq(400);
            await eachSocket(devices.all, (s, i) => s.expectNone('match_result', 500, before[i]));
            expect((await matchRecord(kit, match.id)).winner_id).to.eq(match.redId);
        }));
        it('setelah selesai: start/pause/resume/cancel ditolak dan gelanggang bebas untuk partai berikutnya', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await must('finish', kit.control(match, 'finish', { winner: 'red' }));
            await devices.wasit.waitFor('match_result');
            for (const action of ['start', 'pause', 'resume', 'cancel']) {
                const res = await kit.control(match, action);
                expect(res.status, action).to.eq(400);
                expect(res.error?.code, action).to.eq('INVALID_STATE_TRANSITION');
            }
            const next = await kit.createMatch();
            expect((await kit.loadMatch(next)).status).to.eq(200);
            await kit.control(next, 'cancel');
        }));
        it('wasit membatalkan partai berjalan: semua perangkat menerima state cancelled dan start sesudahnya ditolak', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.all.map((s) => s.mark());
            expect((await kit.control(match, 'cancel')).status).to.eq(200);
            const cancelled = await eachSocket(devices.all, (s, i) => waitState(s, 'cancelled', since[i]));
            cancelled.forEach((m) => expect(m.payload.timer_running).to.eq(false));
            expect((await matchRecord(kit, match.id)).status).to.eq('cancelled');
            expect((await kit.control(match, 'start')).status).to.eq(400);
        }));
    });
    describe('hasil khusus & sanksi (WebSocket wasit)', () => {
        it('walkover sebelum mulai menyelesaikan partai dan hasilnya disiarkan ke semua perangkat', () => runAsync(async () => {
            const { match, devices } = a;
            const since = devices.all.map((s) => s.mark());
            const res = await api('POST', `/competitions/${kit.competitionId}/matches/${match.id}/outcome`, kit.wasit.token, { outcome: 'walkover', winner: 'red' });
            expect(res.status).to.eq(200);
            const results = await eachSocket(devices.all, (s, i) => s.waitFor('match_result', { since: since[i], what: 'match_result' }));
            results.forEach((m) => expect(m.payload).to.include({ corner: 'red', result_type: 'walkover' }));
        }));
        it('KO sebelum partai dimulai ditolak (hanya valid dalam pertandingan)', () => runAsync(async () => {
            const res = await api('POST', `/competitions/${kit.competitionId}/matches/${a.match.id}/outcome`, kit.wasit.token, { outcome: 'ko', winner: 'red' });
            expect(res.status).to.eq(400);
            expect(res.error?.code).to.eq('INVALID_STATE_TRANSITION');
        }));
        it('sanksi dari wasit mengubah skor dan daftar sanksi di semua perangkat; sanksi ganda ditolak', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.all.map((s) => s.mark());
            devices.wasit.send('penalty', { target: 'red', type: 'teguran_1' });
            const updates = await eachSocket(devices.all, (s, i) => s.waitFor('score_update', { since: since[i], what: 'score_update' }));
            updates.forEach((m) => expect(m.payload).to.include({ round: 1, red_penalties: 1 }));
            const lists = await eachSocket(devices.all, (s, i) => s.waitFor('penalties', { since: since[i], what: 'penalties' }));
            lists.forEach((m) => expect(m.payload.penalties).to.have.length(1));
            const mark = devices.wasit.mark();
            devices.wasit.send('penalty', { target: 'red', type: 'teguran_1' });
            const err = await devices.wasit.waitFor('error', { since: mark, what: 'error' });
            expect(err.payload.code).to.eq('PENALTY_ALREADY_APPLIED');
            await devices.judges[0].expectNone('error', 300);
        }));
        it('sinyal gong dan medic dari wasit disiarkan ke seluruh perangkat', () => runAsync(async () => {
            const { devices } = a;
            const since = devices.all.map((s) => s.mark());
            devices.wasit.send('arena_signal', { signal: 'gong' });
            devices.wasit.send('arena_signal', { signal: 'medic', active: true });
            const signals = await eachSocket(devices.all, async (s, i) => {
                await s.waitFor((m) => m.type === 'arena_signal' && m.payload.signal === 'medic', {
                    since: since[i],
                    what: 'arena_signal medic',
                });
                return s
                    .of('arena_signal', since[i])
                    .map((m) => `${m.payload.signal}:${m.payload.active}`);
            });
            signals.forEach((list) => expect(list).to.deep.eq(['gong:true', 'medic:true']));
        }));
    });
    describe('log kejadian (audit)', () => {
        it('GET events memuat match_started dan match_paused berurutan', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await must('pause', kit.control(match, 'pause'));
            await devices.wasit.waitFor((m) => m.type === 'state_change' && m.payload.timer_running === false);
            const events = await must('events', api('GET', `/competitions/${kit.competitionId}/matches/${match.id}/events`, kit.wasit.token));
            const types = events.map((e) => e.event_type);
            expect(types.indexOf('match_started')).to.be.greaterThan(-1);
            expect(types.indexOf('match_paused')).to.be.greaterThan(types.indexOf('match_started'));
        }));
    });
});
