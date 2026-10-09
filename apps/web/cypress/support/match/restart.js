const g = globalThis;
export function restartConfigured() {
    return Boolean(g.Cypress?.env('E2E_RESTART') ?? g.__e2eRestart);
}
/** Satu langkah dalam rantai tes: di Cypress berupa perintah antrean, di Node berupa promise. */
export function restartServer(timeoutMs = 120_000) {
    if (g.cy)
        return g.cy.task('server:restart', null, { timeout: timeoutMs });
    return g.__e2eRestart?.();
}
