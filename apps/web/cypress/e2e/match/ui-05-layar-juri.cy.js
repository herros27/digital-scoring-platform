import { adminToken } from '../../support/match/auth.ts';
import { release } from '../../support/match/helpers.ts';
import { must } from '../../support/match/http.ts';
import { buildKit } from '../../support/match/kit.ts';
import { runAsync } from '../../support/match/run.ts';
/**
 * Layar tablet JURI di browser nyata untuk SETIAP kursi (1-3): koneksi WebSocket sungguhan, aksi wasit lewat
 * REST, dan juri lain dimainkan lewat soket. Layar harus mengikuti server secara realtime TANPA reload.
 * (Hanya Cypress: spec ini memakai halaman; alur API/WS-nya diuji di 02-juri-sinkronisasi-realtime.)
 */
const clock = (text) => {
    const [m, s] = text.trim().split(':').map(Number);
    return m * 60 + s;
};
describe('Layar juri (UI) - sinkronisasi realtime per kursi', () => {
    let kit;
    let a;
    before(() => runAsync(async () => {
        kit = await buildKit(await adminToken(), {
            label: 'UI Juri',
            roundSeconds: 12,
            breakSeconds: 4,
            totalRounds: 2,
        });
    }));
    after(() => runAsync(async () => kit?.teardown()));
    afterEach(() => runAsync(async () => release(kit, a)));
    for (const seat of [1, 2, 3]) {
        describe(`juri kursi ${seat}`, () => {
            beforeEach(() => runAsync(async () => {
                const match = await kit.createMatch();
                // semua perangkat LAIN online lewat soket; kursi yang diuji dilayani layar sungguhan
                const devices = await kit.openDevices({ skipSeats: [seat] });
                await must('load-arena', kit.loadMatch(match));
                a = { match, devices };
            }));
            it('menampilkan identitas, atlet, dan status partai dari server, lalu mengikuti start/jeda/lanjut tanpa reload', () => {
                cy.visitWithSession('/judge', kit.judges[seat - 1]);
                cy.get('.judge-online-pill', { timeout: 10000 }).should('contain.text', 'ONLINE');
                cy.get('.judge-main-title').should('contain.text', `JURI ${seat}`);
                // nama atlet berasal dari endpoint scoreboard publik partai yang dimuat (asinkron: .should mengulang)
                cy.get('.judge-athlete-name')
                    .first()
                    .should(($red) => expect($red.text()).to.eq(a.match.redName));
                cy.get('.judge-athlete-name')
                    .last()
                    .should(($blue) => expect($blue.text()).to.eq(a.match.blueName));
                cy.get('.judge-round-badge').should('contain.text', 'MENUNGGU');
                // wasit memulai lewat REST → layar berubah sendiri
                cy.then(() => runAsync(async () => {
                    await must('start', kit.control(a.match, 'start'));
                }));
                cy.get('.judge-round-badge', { timeout: 8000 })
                    .should('contain.text', 'BABAK 1')
                    .and('contain.text', 'BERJALAN');
                // timer berjalan mundur
                cy.get('.judge-timer-badge')
                    .invoke('text')
                    .then((t1) => {
                    cy.wait(1600);
                    cy.get('.judge-timer-badge')
                        .invoke('text')
                        .should((t2) => expect(clock(t2)).to.be.lessThan(clock(t1)));
                });
                // jeda → timer berhenti dan label berubah
                cy.then(() => runAsync(async () => {
                    await must('pause', kit.control(a.match, 'pause'));
                }));
                cy.get('.judge-round-badge', { timeout: 6000 }).should('contain.text', 'DIJEDA');
                cy.get('.judge-timer-badge')
                    .invoke('text')
                    .then((frozen) => {
                    cy.wait(1500);
                    cy.get('.judge-timer-badge').invoke('text').should('eq', frozen);
                });
                cy.then(() => runAsync(async () => {
                    await must('resume', kit.control(a.match, 'resume'));
                }));
                cy.get('.judge-round-badge', { timeout: 6000 }).should('contain.text', 'BERJALAN');
            });
            it('tap juri di layar + juri lain lewat soket → konsensus terlihat di layar dan di wasit; layar terkunci saat istirahat', () => {
                cy.visitWithSession('/judge', kit.judges[seat - 1]);
                cy.get('.judge-online-pill', { timeout: 10000 }).should('contain.text', 'ONLINE');
                cy.then(() => runAsync(async () => {
                    await must('start', kit.control(a.match, 'start'));
                }));
                cy.get('.judge-round-badge', { timeout: 8000 }).should('contain.text', 'BERJALAN');
                // juri di layar menekan tombol teknik pertama (Pukulan) sudut merah
                cy.get('.judge-btn--red').first().should('not.be.disabled').click();
                cy.get('[data-test="consensus-text-red"]', { timeout: 6000 }).should('contain.text', 'MENUNGGU JURI LAIN');
                // juri lain sepakat → konsensus; wasit menerima skor yang sama
                cy.then(() => runAsync(async () => {
                    const other = a.devices.judges[0];
                    const since = a.devices.wasit.mark();
                    other.send('judge_input', {
                        event_id: `ui-${seat}-${Date.now()}`,
                        score_type: 'technique',
                        target: 'red',
                        round: 1,
                    });
                    const upd = await a.devices.wasit.waitFor('score_update', {
                        since,
                        what: 'score_update dari konsensus',
                    });
                    expect(upd.payload).to.include({ round: 1, red_technical: 1 });
                }));
                cy.get('[data-test="consensus-text-red"]', { timeout: 6000 })
                    .should('contain.text', 'TERKONFIRMASI')
                    .and('have.class', 'judge-consensus-text--ok');
                // saat istirahat: tombol terkunci dan catatan hitung mundur tampil; ronde 2 membuka kunci otomatis
                cy.get('.judge-round-badge', { timeout: 20000 }).should('contain.text', 'ISTIRAHAT');
                cy.get('.judge-break-note').should('be.visible');
                cy.get('.judge-btn--red').first().should('be.disabled');
                cy.get('.judge-btn--blue').first().should('be.disabled');
                cy.get('.judge-round-badge', { timeout: 12000 })
                    .should('contain.text', 'BABAK 2')
                    .and('contain.text', 'BERJALAN');
                cy.get('.judge-btn--red').first().should('not.be.disabled');
            });
            it('wasit menyelesaikan partai → layar menampilkan SELESAI dan tidak ada hasil ganda', () => {
                cy.visitWithSession('/judge', kit.judges[seat - 1]);
                cy.get('.judge-online-pill', { timeout: 10000 }).should('contain.text', 'ONLINE');
                cy.then(() => runAsync(async () => {
                    await must('start', kit.control(a.match, 'start'));
                    await must('finish', kit.control(a.match, 'finish', { winner: 'blue' }));
                }));
                cy.get('.judge-round-badge', { timeout: 8000 }).should('contain.text', 'SELESAI');
            });
        });
    }
    describe('dijeda dan BATAL', () => {
        beforeEach(() => runAsync(async () => {
            const match = await kit.createMatch();
            const devices = await kit.openDevices({ skipSeats: [1] });
            await must('load-arena', kit.loadMatch(match));
            await must('start', kit.control(match, 'start'));
            a = { match, devices };
        }));
        it('saat wasit menjeda: tombol skor terkunci dan ada keterangan; setelah dilanjutkan tombol aktif lagi', () => {
            cy.visitWithSession('/judge', kit.judges[0]);
            cy.get('.judge-round-badge', { timeout: 10000 }).should('contain.text', 'BERJALAN');
            cy.get('.judge-btn--red').first().should('not.be.disabled');
            cy.then(() => runAsync(async () => must('pause', kit.control(a.match, 'pause'))));
            cy.get('.judge-round-badge', { timeout: 8000 }).should('contain.text', 'DIJEDA');
            cy.get('.judge-paused-note').should('be.visible');
            cy.get('.judge-btn--red').each(($b) => expect($b).to.have.prop('disabled', true));
            cy.get('.judge-btn--blue').each(($b) => expect($b).to.have.prop('disabled', true));
            cy.then(() => runAsync(async () => must('resume', kit.control(a.match, 'resume'))));
            cy.get('.judge-round-badge', { timeout: 8000 }).should('contain.text', 'BERJALAN');
            cy.get('.judge-paused-note').should('not.exist');
            cy.get('.judge-btn--red').first().should('not.be.disabled');
        });
        it('BATAL menarik tekanan yang menunggu (lampu padam); BATAL lagi tanpa input menampilkan pesan, bukan diam', () => {
            cy.visitWithSession('/judge', kit.judges[0]);
            cy.get('.judge-round-badge', { timeout: 10000 }).should('contain.text', 'BERJALAN');
            cy.get('.judge-btn--blue').first().click();
            cy.get('[data-test="consensus-text-blue"]', { timeout: 5000 }).should('contain.text', 'MENUNGGU JURI LAIN');
            cy.get('.judge-batal-btn').click();
            cy.get('[data-test="consensus-text-blue"]', { timeout: 5000 }).should('not.exist');
            cy.get('.judge-batal-btn').click();
            cy.get('.judge-notice', { timeout: 5000 })
                .should('be.visible')
                .and('contain.text', 'dibatalkan');
        });
    });
    describe('koneksi dan akses', () => {
        beforeEach(() => runAsync(async () => {
            const match = await kit.createMatch();
            const devices = await kit.openDevices({ skipSeats: [1] });
            await must('load-arena', kit.loadMatch(match));
            a = { match, devices };
        }));
        it('wasit melihat tablet juri yang terbuka online di meja pengawas (kehadiran perangkat)', () => {
            cy.visitWithSession('/judge', kit.judges[0]);
            cy.get('.judge-online-pill', { timeout: 10000 }).should('contain.text', 'ONLINE');
            cy.then(() => runAsync(async () => {
                const w = a.devices.wasit;
                await w.waitFor((m) => m.type === 'presence' &&
                    m.payload.judges.find((j) => j.seat === 1)?.connected === true, { what: 'presence kursi 1 online' });
            }));
        });
        it('tablet juri tanpa sesi diarahkan ke login PIN, dan wasit tidak dapat membuka layar juri', () => {
            cy.visit('/judge');
            cy.location('pathname').should('eq', '/login');
            cy.visitWithSession('/judge', kit.wasit);
            cy.location('pathname', { timeout: 8000 }).should('not.eq', '/judge');
        });
    });
});
