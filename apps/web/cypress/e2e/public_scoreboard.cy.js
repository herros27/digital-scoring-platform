describe('Penonton publik (tanpa login)', () => {
    let arena;
    before(() => {
        // setup data saja: ambil ID gelanggang nyata dari backend (ID berbeda di tiap database seed)
        cy.fetchArenaPins().then((data) => {
            arena = data;
        });
    });
    it('daftar scoreboard live terbuka tanpa login', () => {
        cy.visit('/scoreboard');
        cy.contains('h2', 'SEDANG BERLANGSUNG').should('be.visible');
        cy.contains('h2', 'AKSES PETUGAS').should('be.visible');
        cy.location('pathname').should('eq', '/scoreboard');
    });
    it('tautan Akses Petugas membawa pengguna ke halaman login', () => {
        cy.visit('/scoreboard');
        cy.get('a[href="/login"]').first().click();
        cy.location('pathname').should('eq', '/login');
        cy.contains('h1', 'Digital Scoring Silat').should('be.visible');
    });
    it('tautan login tablet membawa ke keypad PIN', () => {
        cy.visit('/scoreboard');
        cy.get('a[href="/judge/login"]').first().click();
        cy.location('pathname').should('eq', '/judge/login');
        cy.contains('h1', 'Login Tablet Gelanggang').should('be.visible');
    });
    it('scoreboard gelanggang bisa dibuka penonton tanpa login', () => {
        cy.request(`/api/v1/live/scoreboard/${arena.arenaId}`).its('status').should('eq', 200);
        cy.visit(`/scoreboard/${arena.arenaId}`);
        cy.location('pathname').should('eq', `/scoreboard/${arena.arenaId}`);
        cy.window().its('sessionStorage').invoke('getItem', 'ds_token').should('be.null');
    });
    it('halaman pendaftaran kontingen terbuka tanpa login', () => {
        cy.visit('/register');
        cy.contains('h1', 'Daftar Kontingen Baru').should('be.visible');
    });
});
export {};
