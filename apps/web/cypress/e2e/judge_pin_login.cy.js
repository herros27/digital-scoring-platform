// PERHATIAN:
// - Server mengunci IP 15 menit setelah 5 PIN salah beruntun (HTTP 429) -> hanya SATU tes PIN salah.
// - Endpoint login dibatasi 10 request/menit per IP -> data PIN diambil SEKALI di before().
// - Retry dimatikan di spec ini agar percobaan ulang tidak menghabiskan jatah login.
describe('Login tablet dengan PIN (backend nyata)', { retries: 0 }, () => {
    let arena;
    before(() => {
        cy.fetchArenaPins().then((data) => {
            arena = data;
        });
    });
    beforeEach(() => {
        cy.intercept('POST', '**/api/v1/auth/judge-login').as('pinLogin');
    });
    it('juri kursi 1 masuk dengan PIN dan masuk ke layar penilaian', () => {
        cy.loginWithPin(arena.find('judge', 1));
        cy.wait('@pinLogin').its('response.statusCode').should('eq', 200);
        cy.location('pathname').should('eq', '/judge');
        cy.location('search').should('contain', 'seat=1');
        cy.location('search').should('contain', arena.arenaId);
    });
    it('wasit masuk dengan PIN dan masuk ke konsol wasit', () => {
        cy.loginWithPin(arena.find('wasit', 0));
        cy.wait('@pinLogin').its('response.statusCode').should('eq', 200);
        cy.location('pathname').should('eq', '/referee');
    });
    it('PIN yang terlalu pendek ditolak di sisi klien tanpa memanggil server', () => {
        cy.visit('/judge/login');
        cy.contains('button.keypad-btn', /^\s*1\s*$/).click();
        cy.contains('button.keypad-btn', /^\s*2\s*$/).click();
        cy.contains('button', /^\s*Masuk\s*$/).click();
        cy.get('[role="alert"]').should('contain.text', 'PIN harus 6 digit');
        cy.get('@pinLogin.all').should('have.length', 0);
    });
    it('PIN salah ditolak dan tidak membuat sesi', () => {
        cy.loginWithPin('999999');
        cy.wait('@pinLogin').its('response.statusCode').should('be.oneOf', [400, 401, 403, 404]);
        cy.get('[role="alert"]').should('be.visible');
        cy.location('pathname').should('eq', '/judge/login');
        cy.window().its('sessionStorage').invoke('getItem', 'ds_token').should('be.null');
    });
    it('juri tanpa login tidak bisa membuka layar penilaian', () => {
        cy.visit('/judge');
        cy.location('pathname').should('eq', '/login');
    });
});
export {};
