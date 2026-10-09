import { adminToken } from '../../support/match/auth.ts';
import { arrange, matchRecord, release, snapshot, waitState, waitStateWhere, } from '../../support/match/helpers.ts';
import { api, appOrigin, must, sleep } from '../../support/match/http.ts';
import { buildKit } from '../../support/match/kit.ts';
import { ArenaSocket } from '../../support/match/socket.ts';
import { runAsync } from '../../support/match/run.ts';
/**
 * Skenario negatif, aksi berulang/balapan, kegagalan koneksi dan pemulihan.
 * Server adalah sumber kebenaran: setelah koneksi putus/pulih, semua perangkat harus kembali menampilkan
 * keadaan yang sama tanpa kehilangan atau menggandakan kejadian.
 */
describe('Ketahanan & kasus tepi pertandingan', () => {
    let kit;
    let a;
    before(() => runAsync(async () => {
        kit = await buildKit(await adminToken(), {
            label: 'Ketahanan',
            roundSeconds: 20,
            breakSeconds: 3,
            totalRounds: 2,
        });
    }));
    after(() => runAsync(async () => kit?.teardown()));
    beforeEach(() => runAsync(async () => {
        a = await arrange(kit);
    }));
    afterEach(() => runAsync(async () => release(kit, a)));
    describe('gagal memulai pertandingan', () => {
        it('id partai tidak dikenal atau berformat salah → 400 tanpa merusak partai yang sedang dimuat', () => runAsync(async () => {
            const ghost = {
                id: '00000000-0000-4000-8000-000000000000',
                redId: '',
                blueId: '',
                redName: '',
                blueName: '',
            };
            expect((await kit.control(ghost, 'start')).status).to.eq(400);
            expect((await kit.control({ ...ghost, id: 'bukan-uuid' }, 'start')).status).to.eq(400);
            expect((await snapshot(a.devices.wasit)).state).to.eq('idle');
        }));
        it('ruleset tidak ada → load-arena gagal dan partai tetap tidak dapat dimulai', () => runAsync(async () => {
            const loose = await kit.createMatch();
            await kit.control(a.match, 'cancel'); // bebaskan gelanggang
            const res = await api('POST', `/competitions/${kit.competitionId}/matches/${loose.id}/load-arena`, kit.wasit.token, {
                arena_id: kit.arenaId,
                ruleset_path: 'ruleset-yang-tidak-pernah-ada',
            });
            expect(res.status).to.eq(400);
            expect((await kit.control(loose, 'start')).error?.message).to.match(/not loaded/i);
        }));
        it('arena_id wajib pada load-arena (400)', () => runAsync(async () => {
            const loose = await kit.createMatch();
            const res = await api('POST', `/competitions/${kit.competitionId}/matches/${loose.id}/load-arena`, kit.wasit.token, {});
            expect(res.status).to.eq(400);
        }));
        it('partai yang sama dimuat dua kali ditolak; partai kedua ke gelanggang yang sedang dipakai → 409 ARENA_BUSY', () => runAsync(async () => {
            const again = await kit.loadMatch(a.match);
            expect(again.status).to.eq(400);
            expect(again.error?.message).to.match(/already loaded/i);
            const second = await kit.createMatch();
            const busy = await kit.loadMatch(second);
            expect(busy.status).to.eq(409);
            expect(busy.error?.code).to.eq('ARENA_BUSY');
            // partai pertama tidak terganggu
            expect((await snapshot(a.devices.wasit)).match_id).to.eq(a.match.id);
        }));
        it('setelah partai pertama selesai, gelanggang bebas: partai kedua dimuat, dimulai, dan semua juri berpindah ke partai baru', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await must('finish', kit.control(match, 'finish', { winner: 'red' }));
            await devices.wasit.waitFor('match_result');
            const second = await kit.createMatch();
            const since = devices.all.map((s) => s.mark());
            await must('load kedua', kit.loadMatch(second));
            for (const [i, s] of devices.all.entries()) {
                const snap = await s.waitFor((m) => m.type === 'sync_response' && m.payload.match_id === second.id, { since: since[i], what: 'snapshot partai kedua' });
                expect(snap.payload.state).to.eq('idle');
                expect(snap.payload.red_total + snap.payload.blue_total).to.eq(0);
            }
            await must('start kedua', kit.control(second, 'start'));
            await waitState(devices.judges[2], 'round_1', since[3]);
            await kit.control(second, 'cancel');
        }));
        it('partai dibatalkan → dijadwalkan ulang (idle) → dimuat ulang dan dapat dimulai dari 0-0', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await must('cancel', kit.control(match, 'cancel'));
            await waitState(devices.wasit, 'cancelled');
            expect((await matchRecord(kit, match.id)).status).to.eq('cancelled');
            const re = await kit.control(match, 'reschedule');
            expect(re.status).to.eq(200);
            expect((await matchRecord(kit, match.id)).status).to.eq('idle');
            const since = devices.all.map((s) => s.mark());
            await must('load ulang', kit.loadMatch(match));
            const snap = await devices.judges[0].waitFor((m) => m.type === 'sync_response' && m.payload.match_id === match.id, { since: since[1], what: 'snapshot ulang' });
            expect(snap.payload).to.include({ state: 'idle', red_total: 0, blue_total: 0 });
            await must('start ulang', kit.control(match, 'start'));
            await waitState(devices.judges[0], 'round_1', since[1]);
        }));
    });
    describe('aksi berulang dan balapan', () => {
        it('lima permintaan start bersamaan: tepat satu berhasil dan hanya satu siaran round_1', () => runAsync(async () => {
            const { match, devices } = a;
            const since = devices.all.map((s) => s.mark());
            const results = await Promise.all(Array.from({ length: 5 }, () => kit.control(match, 'start')));
            expect(results.filter((r) => r.status === 200)).to.have.length(1);
            results
                .filter((r) => r.status !== 200)
                .forEach((r) => expect(r.error?.code).to.eq('INVALID_STATE_TRANSITION'));
            await sleep(500);
            for (const [i, s] of devices.all.entries()) {
                expect(s.of('state_change', since[i]).filter((m) => m.payload.state === 'round_1'), s.label).to.have.length(1);
            }
        }));
        it('tiga permintaan finish bersamaan: satu hasil resmi, satu match_result per perangkat', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.all.map((s) => s.mark());
            const results = await Promise.all(['red', 'blue', 'red'].map((w) => kit.control(match, 'finish', { winner: w })));
            expect(results.filter((r) => r.status === 200)).to.have.length(1);
            await sleep(500);
            for (const [i, s] of devices.all.entries())
                expect(s.of('match_result', since[i]), s.label).to.have.length(1);
            const winnerCorner = devices.wasit.of('match_result', since[0])[0].payload.corner;
            expect((await matchRecord(kit, match.id)).winner_id).to.eq(winnerCorner === 'red' ? match.redId : match.blueId);
        }));
        it('pause dua kali: tetap dijeda dan sisa waktu tidak berubah; resume dua kali: tenggat tidak bergeser', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            expect((await kit.control(match, 'pause')).status).to.eq(200);
            const frozen = (await snapshot(devices.wasit)).timer_remaining_ms;
            expect((await kit.control(match, 'pause')).status).to.eq(200);
            await sleep(700);
            const s2 = await snapshot(devices.judges[0]);
            expect(s2).to.include({ timer_running: false, timer_remaining_ms: frozen });
            expect((await kit.control(match, 'resume')).status).to.eq(200);
            const ends1 = (await snapshot(devices.wasit)).timer_ends_at_ms;
            expect((await kit.control(match, 'resume')).status).to.eq(200);
            const ends2 = (await snapshot(devices.judges[1])).timer_ends_at_ms;
            expect(ends2).to.eq(ends1);
        }));
        it('cancel dua kali: kedua kali ditolak/diabaikan tanpa mengubah state akhir cancelled', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await must('cancel', kit.control(match, 'cancel'));
            const again = await kit.control(match, 'cancel');
            expect(again.status).to.eq(400);
            await waitState(devices.wasit, 'cancelled');
            expect((await snapshot(devices.wasit)).state).to.eq('cancelled');
        }));
    });
    describe('kegagalan koneksi dan pemulihan', () => {
        it('juri terputus: wasit melihat kursinya offline; saat tersambung ulang menerima snapshot terkini dan dapat menilai lagi', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const presenceOf = async () => (await snapshot(devices.wasit)).presence.judges.find((j) => j.seat === 2).connected;
            expect(await presenceOf()).to.eq(true);
            await devices.judges[1].drop(); // jaringan tablet juri 2 putus
            await sleep(300);
            expect(await presenceOf()).to.eq(false);
            // selagi putus: wasit menjeda lalu melanjutkan, dan juri lain mencetak poin
            await must('pause', kit.control(match, 'pause'));
            await must('resume', kit.control(match, 'resume'));
            const back = await ArenaSocket.open(kit.arenaId, kit.judges[1].token, 'juri-2-pulih');
            devices.judges[1] = back;
            devices.all[2] = back;
            const sync = await snapshot(back);
            expect(sync).to.include({ match_id: match.id, state: 'round_1', timer_running: true });
            expect(await presenceOf()).to.eq(true);
            // juri yang pulih langsung ikut konsensus
            const since = devices.wasit.mark();
            devices.judges[0].send('judge_input', {
                event_id: 'rec-1',
                score_type: 'fall',
                target: 'blue',
                round: 1,
            });
            back.send('judge_input', {
                event_id: 'rec-2',
                score_type: 'fall',
                target: 'blue',
                round: 1,
            });
            const upd = await devices.wasit.waitFor('score_update', {
                since,
                what: 'score_update setelah pulih',
            });
            expect(upd.payload).to.include({ blue_technical: 3 });
        }));
        it('perubahan state saat juri offline tidak hilang: snapshot setelah sambung ulang mencerminkan keadaan terbaru', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            await devices.judges[2].drop();
            await must('pause', kit.control(match, 'pause'));
            await sleep(400);
            const back = await ArenaSocket.open(kit.arenaId, kit.judges[2].token, 'juri-3-pulih');
            devices.judges[2] = back;
            devices.all[3] = back;
            const s = await snapshot(back);
            expect(s).to.include({ state: 'round_1', timer_running: false });
            expect(s.timer_remaining_ms).to.be.within(1, 20000);
        }));
        it('semua perangkat terputus bersamaan: timer tetap berjalan di server dan semua kembali sinkron setelah tersambung ulang', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const before = (await snapshot(devices.wasit)).timer_remaining_ms;
            await Promise.all(devices.all.map((s) => s.drop()));
            await sleep(1500); // jaringan arena mati 1,5 detik
            const wasit = await ArenaSocket.open(kit.arenaId, kit.wasit.token, 'wasit-pulih');
            const judges = await Promise.all(kit.judges.map((j) => ArenaSocket.open(kit.arenaId, j.token, `juri-${j.seat}-pulih`)));
            devices.wasit = wasit;
            devices.judges = judges;
            devices.all = [wasit, ...judges];
            const snaps = await Promise.all(devices.all.map((s) => snapshot(s)));
            snaps.forEach((p) => {
                expect(p).to.include({ match_id: match.id, state: 'round_1', timer_running: true });
                expect(p.timer_remaining_ms).to.be.lessThan(before - 1200);
            });
            expect(new Set(snaps.map((p) => p.timer_ends_at_ms)).size).to.eq(1);
            expect(snaps[0].presence.judges.filter((j) => j.connected).map((j) => j.seat)).to.include.members([1, 2, 3]);
        }));
        it('wasit terputus lalu tersambung ulang: kontrol REST dan siaran ke juri tetap berfungsi', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            await devices.wasit.drop();
            await sleep(300);
            expect((await snapshot(devices.judges[0])).presence.referees).to.eq(0);
            const wasit = await ArenaSocket.open(kit.arenaId, kit.wasit.token, 'wasit-pulih');
            devices.wasit = wasit;
            devices.all[0] = wasit;
            expect((await snapshot(devices.judges[0])).presence.referees).to.eq(1);
            const since = devices.judges[0].mark();
            await must('pause', kit.control(match, 'pause'));
            await devices.judges[0].waitFor((m) => m.type === 'state_change' && m.payload.timer_running === false, { since, what: 'jeda' });
        }));
        it('dua tablet memakai PIN juri yang sama: keduanya menerima siaran dan kursi tetap online sampai keduanya putus', () => runAsync(async () => {
            const { match, devices } = a;
            const twin = await ArenaSocket.open(kit.arenaId, kit.judges[0].token, 'juri-1-kembar');
            try {
                await sleep(150);
                const since = [devices.judges[0].mark(), twin.mark()];
                await must('start', kit.control(match, 'start'));
                await waitState(devices.judges[0], 'round_1', since[0]);
                await waitState(twin, 'round_1', since[1]);
                await twin.close();
                await sleep(300);
                const seat1 = (await snapshot(devices.wasit)).presence.judges.find((j) => j.seat === 1);
                expect(seat1.connected, 'kursi 1 masih terhubung lewat tablet pertama').to.eq(true);
            }
            finally {
                await twin.close();
            }
        }));
        it('pesan rusak (bukan JSON) dan tipe tak dikenal tidak memutus koneksi maupun mengganggu partai', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const sock = devices.judges[0];
            sock.sendRaw('ini bukan json {{{');
            sock.send('tipe_tidak_dikenal', { x: 1 });
            await sleep(400);
            expect(sock.isOpen).to.eq(true);
            // masih bisa menilai dan menerima siaran
            const since = devices.wasit.mark();
            sock.send('judge_input', {
                event_id: 'ok-1',
                score_type: 'technique',
                target: 'red',
                round: 1,
            });
            devices.judges[1].send('judge_input', {
                event_id: 'ok-2',
                score_type: 'technique',
                target: 'red',
                round: 1,
            });
            await devices.wasit.waitFor('score_update', {
                since,
                what: 'score_update setelah pesan rusak',
            });
        }));
    });
    describe('konsistensi antar kanal (WebSocket vs endpoint publik)', () => {
        it('scoreboard publik (REST) menampilkan state, ronde, skor, dan nama atlet yang sama dengan WebSocket', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const since = devices.wasit.mark();
            devices.judges[0].send('judge_input', {
                event_id: 'pub-1',
                score_type: 'knockdown',
                target: 'red',
                round: 1,
            });
            devices.judges[1].send('judge_input', {
                event_id: 'pub-2',
                score_type: 'knockdown',
                target: 'red',
                round: 1,
            });
            await devices.wasit.waitFor('score_update', { since, what: 'score_update' });
            const ws = await snapshot(devices.judges[2]);
            const pub = await api('GET', `/live/scoreboard/${kit.arenaId}`, null); // tanpa login
            expect(pub.status).to.eq(200);
            expect(pub.data).to.include({
                match_id: match.id,
                state: ws.state,
                round: ws.round,
                red_total: ws.red_total,
                blue_total: ws.blue_total,
            });
            expect(pub.data.red_total).to.eq(2);
            expect(pub.data.red_athlete).to.match(/^Merah/);
            expect(pub.data.blue_athlete).to.match(/^Biru/);
        }));
        it('daftar gelanggang live memuat gelanggang saat partai berjalan dan tidak lagi setelah selesai', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitState(devices.wasit, 'round_1');
            const live = await api('GET', '/live/arenas', null);
            expect(live.data.map((x) => x.arena_id)).to.include(kit.arenaId);
            await must('finish', kit.control(match, 'finish', { winner: 'red' }));
            await devices.wasit.waitFor('match_result');
            const after = await api('GET', '/live/arenas', null);
            expect(after.data.map((x) => x.arena_id)).to.not.include(kit.arenaId);
        }));
        it('endpoint kesehatan dan origin aplikasi tersedia (prasyarat suite)', () => runAsync(async () => {
            const res = await fetch(`${appOrigin()}/api/v1/health`);
            expect(res.status).to.eq(200);
        }));
    });
    describe('daftar partai mencerminkan status berjalan', () => {
        it('GET matches?status=running memuat partai yang sedang berjalan dan tidak memuatnya setelah selesai', () => runAsync(async () => {
            const { match, devices } = a;
            await must('start', kit.control(match, 'start'));
            await waitStateWhere(devices.wasit, (p) => p.state === 'round_1', 'round_1');
            const url = `/competitions/${kit.competitionId}/matches?status=running&limit=50`;
            const running = await must('list running', api('GET', url, kit.wasit.token));
            expect(running.map((m) => m.id)).to.include(match.id);
            await must('finish', kit.control(match, 'finish', { winner: 'blue' }));
            const after = await must('list running', api('GET', url, kit.wasit.token));
            expect(after.map((m) => m.id)).to.not.include(match.id);
        }));
    });
});
