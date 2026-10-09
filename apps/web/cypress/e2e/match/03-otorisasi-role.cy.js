import { adminToken, dewanToken, operatorToken } from '../../support/match/auth.ts';
import { arrange, release, snapshot, waitState } from '../../support/match/helpers.ts';
import { api, must, sleep } from '../../support/match/http.ts';
import { buildKit, loginContingent } from '../../support/match/kit.ts';
import { HandshakeRejected, ArenaSocket } from '../../support/match/socket.ts';
import { runAsync } from '../../support/match/run.ts';
/**
 * Otorisasi per role pada pertandingan: REST (kontrol partai & input skor) dan WebSocket (handshake + perintah).
 * Hak akses yang diharapkan:
 *   - kontrol partai (load/start/pause/resume/finish/cancel/outcome): wasit, dewan, admin;
 *     PIN terikat ke SATU gelanggang (wasit gelanggang lain tidak boleh).
 *   - input skor: hanya juri PIN gelanggang itu, atas nama dirinya sendiri.
 *   - buat partai: admin/dewan.  Operator, kontingen, dan tanpa login tidak boleh mengontrol/menilai.
 */
describe('Otorisasi role - pertandingan', () => {
    let kit;
    let a;
    let tokens;
    before(() => runAsync(async () => {
        const admin = await adminToken();
        kit = await buildKit(admin, {
            label: 'Otorisasi',
            roundSeconds: 20,
            breakSeconds: 3,
            totalRounds: 2,
            outsideWasit: true,
        });
        tokens = {
            admin,
            dewan: await dewanToken(),
            operator: await operatorToken(),
            contingent: await loginContingent('tapaksuci@contingent.id', 'DemoPass_2026'),
        };
    }));
    after(() => runAsync(async () => kit?.teardown()));
    beforeEach(() => runAsync(async () => {
        a = await arrange(kit);
    }));
    afterEach(() => runAsync(async () => release(kit, a)));
    const path = (id, action) => `/competitions/${kit.competitionId}/matches/${id}/${action}`;
    describe('REST: kontrol partai', () => {
        const denied = [
            ['tanpa login', () => null, 401],
            ['juri (PIN)', () => kit.judges[0].token, 403],
            ['operator (email)', () => tokens.operator, 403],
            ['manajer kontingen', () => tokens.contingent, 403],
            ['wasit gelanggang lain (PIN)', () => kit.outsideWasit.token, 403],
            ['juri gelanggang lain (PIN)', () => kit.outsideJudge.token, 403],
        ];
        for (const action of ['start', 'pause', 'resume', 'finish', 'cancel']) {
            for (const [who, token, status] of denied) {
                it(`${action} oleh ${who} → ${status} dan state partai tidak berubah`, () => runAsync(async () => {
                    const { match, devices } = a;
                    await must('start (wasit)', kit.control(match, 'start'));
                    await waitState(devices.wasit, 'round_1');
                    const since = devices.all.map((s) => s.mark());
                    const res = await api('POST', path(match.id, action), token(), action === 'finish' ? { winner: 'red' } : undefined);
                    expect(res.status, `${who}/${action}`).to.eq(status);
                    expect(res.error?.code).to.eq(status === 401 ? 'UNAUTHORIZED' : 'FORBIDDEN');
                    await Promise.all(devices.all.map((s, i) => s.expectNone((m) => m.type === 'state_change' || m.type === 'match_result', 350, since[i])));
                    const snap = await snapshot(devices.wasit);
                    expect(snap.state).to.eq('round_1');
                    expect(snap.timer_running).to.eq(true);
                }));
            }
        }
        it('load-arena oleh role tak berhak → 401/403 dan partai tidak dimuat', () => runAsync(async () => {
            const loose = await kit.createMatch();
            const attempts = [
                [null, 401],
                [kit.judges[0].token, 403],
                [tokens.operator, 403],
                [tokens.contingent, 403],
                [kit.outsideWasit.token, 403],
            ];
            for (const [token, status] of attempts) {
                const res = await api('POST', path(loose.id, 'load-arena'), token, {
                    arena_id: kit.arenaId,
                    ruleset_path: kit.rulesetId,
                });
                expect(res.status).to.eq(status);
            }
            const start = await kit.control(loose, 'start');
            expect(start.error?.message).to.match(/not loaded/i);
        }));
        it('outcome (WO/KO) oleh role tak berhak → ditolak', () => runAsync(async () => {
            const { match } = a;
            for (const token of [
                null,
                kit.judges[0].token,
                tokens.operator,
                tokens.contingent,
                kit.outsideWasit.token,
            ]) {
                const res = await api('POST', path(match.id, 'outcome'), token, {
                    outcome: 'walkover',
                    winner: 'red',
                });
                expect([401, 403]).to.include(res.status);
            }
            expect((await snapshot(a.devices.wasit)).state).to.eq('idle');
        }));
        it('membuat partai hanya untuk admin/dewan (wasit, juri, kontingen, tanpa login ditolak)', () => runAsync(async () => {
            const body = {
                arena_id: kit.arenaId,
                athlete_red_id: a.match.redId,
                athlete_blue_id: a.match.blueId,
            };
            const url = `/competitions/${kit.competitionId}/matches`;
            expect((await api('POST', url, null, body)).status).to.eq(401);
            for (const token of [
                kit.wasit.token,
                kit.judges[0].token,
                tokens.contingent,
                tokens.operator,
            ]) {
                expect((await api('POST', url, token, body)).status).to.eq(403);
            }
        }));
    });
    describe('REST: role yang berhak mengontrol', () => {
        it('wasit (PIN), dewan, dan admin masing-masing dapat memegang kendali partai yang sama', () => runAsync(async () => {
            const { match, devices } = a;
            expect((await kit.control(match, 'start', undefined, kit.wasit.token)).status).to.eq(200);
            await waitState(devices.judges[0], 'round_1');
            expect((await kit.control(match, 'pause', undefined, tokens.dewan)).status).to.eq(200);
            await devices.judges[0].waitFor((m) => m.type === 'state_change' && m.payload.timer_running === false);
            expect((await kit.control(match, 'resume', undefined, tokens.admin)).status).to.eq(200);
            await devices.judges[0].waitFor((m) => m.type === 'state_change' && m.payload.timer_running === true && m.seq > 2);
            expect((await snapshot(devices.wasit)).timer_running).to.eq(true);
        }));
    });
    describe('REST: input skor juri', () => {
        const score = (matchId, token, body = {}, headers) => api('POST', path(matchId, 'scores'), token, {
            score_type: 'technique',
            target: 'red',
            ...body,
            ...(headers ? { idempotency_key: headers } : {}),
        });
        it('hanya juri yang boleh: wasit, dewan, admin, operator, kontingen, tanpa login ditolak', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.all.map((s) => s.mark());
            const cases = [
                ['wasit', kit.wasit.token, 403],
                ['dewan', tokens.dewan, 403],
                ['admin', tokens.admin, 403],
                ['operator', tokens.operator, 403],
                ['kontingen', tokens.contingent, 403],
                ['anon', null, 401],
            ];
            for (const [who, token, status] of cases) {
                const res = await score(match.id, token, { judge_id: 'kursi-1' });
                expect(res.status, who).to.eq(status);
            }
            await Promise.all(devices.all.map((s, i) => s.expectNone((m) => m.type === 'consensus_progress' || m.type === 'score_update', 300, since[i])));
        }));
        it('juri gelanggang lain tidak boleh menilai partai ini (403)', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const res = await score(match.id, kit.outsideJudge.token);
            expect(res.status).to.eq(403);
            expect(res.error?.message).to.match(/gelanggang/i);
        }));
        it('juri sah: suara tercatat atas identitas token; judge_id palsu pada body diabaikan (tidak bisa menggandakan suara)', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.wasit.mark();
            // juri 1 berpura-pura menjadi dua juri berbeda
            const r1 = await score(match.id, kit.judges[0].token, { judge_id: 'kursi-1' }, 'imp-1');
            const r2 = await score(match.id, kit.judges[0].token, { judge_id: 'kursi-2' }, 'imp-2');
            expect(r1.status).to.eq(200);
            expect(r2.status).to.eq(200);
            await devices.wasit.expectNone('score_update', 500, since);
            expect((await snapshot(devices.wasit)).red_total).to.eq(0);
            // juri 2 yang sebenarnya menyepakati → konsensus tercapai
            const r3 = await score(match.id, kit.judges[1].token, { judge_id: 'kursi-1' }, 'imp-3');
            expect(r3.status).to.eq(200);
            await devices.wasit.waitFor('score_update', { since, what: 'score_update' });
            expect((await snapshot(devices.wasit)).red_total).to.eq(1);
        }));
        it('Idempotency-Key yang sama tidak menggandakan suara (duplicate_ignored)', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const first = await score(match.id, kit.judges[0].token, {}, 'same-key');
            const second = await score(match.id, kit.judges[0].token, {}, 'same-key');
            expect(first.data.status).to.eq('accepted');
            expect(second.data).to.include({ status: 'duplicate_ignored', duplicate: true });
        }));
        it('saat partai belum dimulai: skor REST ditolak (SCORING_ERROR)', () => runAsync(async () => {
            const res = await score(a.match.id, kit.judges[0].token);
            expect(res.status).to.eq(400);
            expect(res.error?.code).to.eq('SCORING_ERROR');
        }));
    });
    describe('WebSocket: handshake', () => {
        const rejected = async (token, arena = kit.arenaId) => {
            try {
                const s = await ArenaSocket.open(arena, token, 'uji-handshake', 4000);
                await s.close();
                return false;
            }
            catch (e) {
                if (e instanceof HandshakeRejected)
                    return true;
                throw e;
            }
        };
        it('token PIN sah (wasit dan setiap juri) diterima', () => runAsync(async () => {
            for (const s of [kit.wasit, ...kit.judges])
                expect(await rejected(s.token), `seat ${s.seat ?? 'wasit'}`).to.eq(false);
        }));
        it('token acak/rusak ditolak', () => runAsync(async () => {
            expect(await rejected('bukan-token')).to.eq(true);
            expect(await rejected(kit.judges[0].token.slice(0, -4) + 'abcd')).to.eq(true);
        }));
        it('token PIN gelanggang lain ditolak (juri dan wasit)', () => runAsync(async () => {
            expect(await rejected(kit.outsideJudge.token)).to.eq(true);
            expect(await rejected(kit.outsideWasit.token)).to.eq(true);
            // dan sebaliknya: PIN gelanggang ini tidak boleh masuk gelanggang lain
            expect(await rejected(kit.judges[0].token, kit.otherArenaId)).to.eq(true);
        }));
        it('login email tanpa arena: admin dan dewan diterima; operator dan kontingen ditolak', () => runAsync(async () => {
            expect(await rejected(tokens.admin)).to.eq(false);
            expect(await rejected(tokens.dewan)).to.eq(false);
            expect(await rejected(tokens.operator)).to.eq(true);
            expect(await rejected(tokens.contingent)).to.eq(true);
        }));
        it('penonton tanpa token diterima bila gelanggang punya partai aktif, ditolak bila tidak ada hub', () => runAsync(async () => {
            expect(await rejected(null)).to.eq(false);
            expect(await rejected(null, '00000000-0000-4000-8000-000000000000')).to.eq(true);
        }));
    });
    describe('WebSocket: perintah per role', () => {
        it('penonton: menerima siaran tetapi semua perintah ditolak (FORBIDDEN)', () => runAsync(async () => {
            const { match, devices } = a;
            const tv = await ArenaSocket.open(kit.arenaId, null, 'penonton');
            try {
                await sleep(150);
                const since = tv.mark();
                tv.send('state_change', { action: 'start' });
                tv.send('judge_input', {
                    event_id: 'tv-1',
                    score_type: 'technique',
                    target: 'red',
                    round: 1,
                });
                tv.send('penalty', { target: 'red', type: 'teguran_1' });
                tv.send('arena_signal', { signal: 'gong' });
                await sleep(500);
                expect(tv.of('error', since).map((m) => m.payload.code)).to.deep.eq([
                    'FORBIDDEN',
                    'FORBIDDEN',
                    'FORBIDDEN',
                    'FORBIDDEN',
                ]);
                expect((await snapshot(devices.wasit)).state).to.eq('idle');
                // tetapi tetap menerima siaran partai dari wasit
                const mark = tv.mark();
                await must('start', kit.control(match, 'start'));
                const change = await waitState(tv, 'round_1', mark);
                expect(change.payload.timer_running).to.eq(true);
                // penonton dapat meminta snapshot
                expect((await snapshot(tv)).match_id).to.eq(match.id);
            }
            finally {
                await tv.close();
            }
        }));
        it('wasit tidak boleh mengirim input juri (FORBIDDEN) dan skor tidak berubah', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.wasit.mark();
            devices.wasit.send('judge_input', {
                event_id: 'w-1',
                score_type: 'technique',
                target: 'red',
                round: 1,
            });
            devices.wasit.send('judge_input', {
                event_id: 'w-2',
                score_type: 'technique',
                target: 'red',
                round: 1,
            });
            const err = await devices.wasit.waitFor('error', { since, what: 'error' });
            expect(err.payload.code).to.eq('FORBIDDEN');
            await devices.judges[0].expectNone((m) => m.type === 'consensus_progress' || m.type === 'score_update', 400);
            expect((await snapshot(devices.wasit)).red_total).to.eq(0);
        }));
        it('admin (login email) dapat mengontrol lewat WebSocket, dewan juga', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            for (const token of [tokens.admin, tokens.dewan]) {
                const sock = await ArenaSocket.open(kit.arenaId, token, 'petugas');
                try {
                    const since = devices.judges[0].mark();
                    sock.send('state_change', { action: 'pause' });
                    await devices.judges[0].waitFor((m) => m.type === 'state_change' && m.payload.timer_running === false, { since, what: 'jeda oleh petugas' });
                    sock.send('state_change', { action: 'resume' });
                    await devices.judges[0].waitFor((m) => m.type === 'state_change' && m.payload.timer_running === true, { since: devices.judges[0].mark() - 1, what: 'lanjut oleh petugas' });
                }
                finally {
                    await sock.close();
                }
            }
        }));
        it('wasit mengirim aksi tak dikenal / payload rusak → error ke pengirim saja, partai tetap normal', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.all.map((s) => s.mark());
            devices.wasit.send('state_change', { action: 'terbangkan_bola' });
            devices.wasit.send('penalty', { target: 'hijau', type: 'teguran_1' });
            devices.wasit.send('arena_signal', { signal: 'sirene' });
            await sleep(500);
            expect(devices.wasit.of('error', since[0]).map((m) => m.payload.code)).to.deep.eq([
                'UNKNOWN_ACTION',
                'BAD_PAYLOAD',
                'BAD_PAYLOAD',
            ]);
            await devices.judges[2].expectNone('error', 100, since[3]);
            expect((await snapshot(devices.wasit)).state).to.eq('round_1');
        }));
    });
});
