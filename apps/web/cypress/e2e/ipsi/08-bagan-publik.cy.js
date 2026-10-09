import { activateCompetition, adminToken, apiCall, cancelCompetition, createCompetition, pauseCompetition, seedBracket, } from '../../support/api';
/**
 * Bagan publik untuk penonton (tanpa login): hanya muncul setelah kejuaraan AKTIF, dan hanya memuat nama +
 * kontingen atlet. Semua tes berjalan tanpa token; token admin hanya dipakai menyiapkan data.
 */
// Semua field yang boleh muncul di respons publik (rekursif). Menambah field = harus sengaja menambah di sini.
const ALLOWED_FIELDS = new Set([
    'success',
    'data',
    'id',
    'name',
    'location',
    'start_date',
    'end_date',
    'competition',
    'category',
    'category_id',
    'gender',
    'age_group',
    'weight_class',
    'participants',
    'rounds',
    'completed',
    'slots',
    'round',
    'slot_index',
    'is_bye',
    'next_slot_id',
    'is_winner',
    'participant',
    'team',
]);
function collectKeys(value, into = new Set()) {
    if (Array.isArray(value))
        value.forEach((v) => collectKeys(v, into));
    else if (value && typeof value === 'object') {
        Object.entries(value).forEach(([k, v]) => {
            into.add(k);
            collectKeys(v, into);
        });
    }
    return into;
}
describe('Bagan publik', () => {
    let admin;
    let comp;
    let putra;
    let putri;
    const list = () => apiCall('GET', `/public/competitions/${comp.id}/brackets`, null);
    const detail = (categoryId) => apiCall('GET', `/public/competitions/${comp.id}/brackets/${categoryId}`, null);
    before(() => {
        adminToken().then((t) => {
            admin = t;
            createCompetition(t, 'Bagan Publik').then((c) => {
                comp = c;
                seedBracket(t, c.id, { name: 'Dewasa Putra Publik', prefix: 'Putra' }).then((s) => (putra = s));
                seedBracket(t, c.id, {
                    name: 'Pra Remaja Putri Publik',
                    gender: 'female',
                    ageGroup: 'pra_remaja',
                    age: 13,
                    prefix: 'Putri',
                    athletes: 2,
                }).then((s) => (putri = s));
            });
        });
    });
    after(() => {
        if (comp)
            cancelCompetition(admin, comp.id);
    });
    describe('sebelum kejuaraan diaktifkan (draft)', () => {
        it('bagan disamarkan 404 dan kejuaraan tidak muncul di daftar publik', () => {
            list().its('status').should('eq', 404);
            detail(putra.category.id).its('status').should('eq', 404);
            apiCall('GET', '/public/competitions', null).then((res) => {
                expect(res.status).to.eq(200);
                expect(res.body.data.map((c) => c.id)).to.not.include(comp.id);
            });
        });
        it('halaman /bagan tidak menampilkan kejuaraan draft', () => {
            cy.visit('/bagan');
            cy.get('main').should('be.visible');
            cy.contains(comp.name).should('not.exist');
        });
    });
    describe('setelah kejuaraan aktif', () => {
        before(() => {
            activateCompetition(admin, comp.id).its('status').should('eq', 200);
        });
        it('kejuaraan muncul di daftar publik tanpa login', () => {
            apiCall('GET', '/public/competitions', null).then((res) => {
                const found = res.body.data.find((c) => c.id === comp.id);
                expect(found, 'kejuaraan aktif').to.not.equal(undefined);
                expect(found.start_date).to.match(/^\d{4}-\d{2}-\d{2}$/);
            });
        });
        it('daftar bagan memuat satu entri per kategori yang sudah dibuat', () => {
            list().then((res) => {
                expect(res.status).to.eq(200);
                const names = res.body.data.map((b) => b.name);
                expect(names).to.include.members(['Dewasa Putra Publik', 'Pra Remaja Putri Publik']);
            });
        });
        it('kategori tanpa bagan tidak ikut terdaftar', () => {
            // kategori kedua dibuat khusus untuk tes ini, tanpa atlet dan tanpa bagan
            cy.then(() => apiCall('POST', `/competitions/${comp.id}/categories`, admin, {
                name: 'Kategori Tanpa Bagan',
                gender: 'male',
                weight_class: `K${Date.now().toString(36)}`,
            }));
            list().then((res) => {
                expect(res.body.data.map((b) => b.name)).to.not.include('Kategori Tanpa Bagan');
            });
        });
        it('detail bagan menampilkan nama dan kontingen atlet', () => {
            detail(putra.category.id).then((res) => {
                expect(res.status).to.eq(200);
                expect(res.body.data.competition.id).to.eq(comp.id);
                const shown = res.body.data.slots
                    .filter((s) => s.round === 1 && s.participant)
                    .map((s) => s.participant.name);
                expect(shown).to.have.members(putra.athleteNames);
                res.body.data.slots
                    .filter((s) => s.participant)
                    .forEach((s) => expect(s.participant.team).to.match(/^Klub Putra \d$/));
            });
        });
        it('respons publik HANYA memuat field yang diizinkan dan tidak membocorkan data pribadi', () => {
            const responses = [
                apiCall('GET', '/public/competitions', null),
                list(),
                detail(putra.category.id),
                detail(putri.category.id),
            ];
            responses.forEach((chain) => chain.then((res) => {
                expect(res.status).to.eq(200);
                const keys = collectKeys(res.body);
                const unexpected = [...keys].filter((k) => !ALLOWED_FIELDS.has(k));
                expect(unexpected, 'field di luar whitelist publik').to.deep.eq([]);
                const raw = JSON.stringify(res.body);
                const secrets = [...putra.secrets, ...putri.secrets];
                secrets.forEach((s) => expect(raw, `data sensitif ${s}`).to.not.include(s));
                [
                    '"nik"',
                    'birth_date',
                    'actual_weight',
                    'docs_verified',
                    'athlete_id',
                    'participant_id',
                    'contingent_id',
                ].forEach((w) => expect(raw, `kunci ${w}`).to.not.include(w));
            }));
        });
        it('bagan kategori lain, kategori yang tidak ada, dan id salah memberi 404', () => {
            detail('00000000-0000-4000-8000-000000000000').its('status').should('eq', 404);
            detail('bukan-uuid').its('status').should('eq', 404);
            // kategori milik kejuaraan lain
            createCompetition(admin, 'Bagan Lain').then((other) => {
                activateCompetition(admin, other.id);
                apiCall('GET', `/public/competitions/${other.id}/brackets/${putra.category.id}`, null)
                    .its('status')
                    .should('eq', 404);
                cancelCompetition(admin, other.id);
            });
        });
        describe('halaman /bagan (UI)', () => {
            const openList = () => {
                cy.visit('/bagan');
                // bila ada lebih dari satu kejuaraan aktif, pilih kejuaraan uji
                cy.get('main').should('be.visible');
                cy.get('body').then(($body) => {
                    if ($body.find('#bgl-comp').length)
                        cy.get('#bgl-comp').select(comp.id);
                });
            };
            it('dapat dibuka tanpa login dan menampilkan daftar bagan', () => {
                openList();
                cy.location('pathname').should('eq', '/bagan');
                cy.contains('h1', 'BAGAN PERTANDINGAN').should('be.visible');
                cy.contains('[data-test="bracket-card"]', 'Dewasa Putra Publik').should('be.visible');
                cy.contains('[data-test="bracket-card"]', 'Pra Remaja Putri Publik').should('be.visible');
            });
            it('dikelompokkan per golongan umur', () => {
                openList();
                cy.contains('h2', 'PRA-REMAJA').should('be.visible');
                cy.contains('h2', 'DEWASA').should('be.visible');
            });
            it('filter jenis kelamin dan pencarian menyaring daftar', () => {
                openList();
                cy.contains('button', 'PUTRI').click();
                cy.contains('[data-test="bracket-card"]', 'Pra Remaja Putri Publik').should('be.visible');
                cy.contains('[data-test="bracket-card"]', 'Dewasa Putra Publik').should('not.exist');
                cy.contains('button', 'SEMUA').click();
                cy.get('input[type="search"]').type('dewasa putra');
                cy.contains('[data-test="bracket-card"]', 'Dewasa Putra Publik').should('be.visible');
                cy.contains('[data-test="bracket-card"]', 'Pra Remaja Putri Publik').should('not.exist');
                cy.get('input[type="search"]').clear().type('zzz-tidak-ada');
                cy.get('[data-test="no-match"]').should('be.visible');
            });
            it('klik salah satu bagan membuka bagan lengkap dengan nama atlet', () => {
                openList();
                cy.contains('[data-test="bracket-card"]', 'Dewasa Putra Publik').click();
                cy.location('pathname').should('eq', `/bagan/${comp.id}/${putra.category.id}`);
                cy.get('[data-test="category-name"]').should('have.text', 'Dewasa Putra Publik');
                cy.contains('BABAK FINAL').should('be.visible');
                putra.athleteNames.forEach((n) => cy.contains(n).should('be.visible'));
                cy.contains('Klub Putra 1').should('be.visible');
                // data pribadi tidak tampil di halaman
                putra.secrets.forEach((s) => cy.contains(s).should('not.exist'));
            });
            it('tombol kembali membawa ke daftar bagan', () => {
                cy.visit(`/bagan/${comp.id}/${putra.category.id}`);
                cy.get('[data-test="back-to-list"]').click();
                cy.location('pathname').should('eq', '/bagan');
            });
            it('bagan yang tidak ada menampilkan pesan, bukan layar kosong', () => {
                cy.visit(`/bagan/${comp.id}/00000000-0000-4000-8000-000000000000`);
                cy.get('[data-test="not-found"]').should('be.visible');
            });
            it('scoreboard publik menyediakan tautan ke daftar bagan', () => {
                cy.visit('/scoreboard');
                cy.get('[data-test="link-bagan"]').click();
                cy.location('pathname').should('eq', '/bagan');
            });
        });
    });
    describe('setelah kejuaraan dijeda', () => {
        it('bagan kembali disembunyikan dari publik segera (cache dibersihkan)', () => {
            pauseCompetition(admin, comp.id).its('status').should('eq', 200);
            list().its('status').should('eq', 404);
            detail(putra.category.id).its('status').should('eq', 404);
            apiCall('GET', '/public/competitions', null).then((res) => {
                expect(res.body.data.map((c) => c.id)).to.not.include(comp.id);
            });
        });
    });
});
