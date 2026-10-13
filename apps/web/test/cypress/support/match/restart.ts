/**
 * Hook restart server untuk uji pemulihan. Aplikasi uji tidak boleh mematikan backend sendiri, jadi perintah
 * restart disediakan lingkungan lewat E2E_RESTART_CMD (perintah shell yang mematikan, menjalankan ulang, dan
 * baru kembali setelah server siap). Tanpa variabel itu spec pemulihan dilewati, bukan gagal.
 *
 * - Cypress: `cy.task('server:restart')` (didaftarkan di cypress.config.ts, menjalankan perintah di proses Node).
 * - Runner Node: `globalThis.__e2eRestart` dipasang oleh scripts/run-match-flow.ts.
 */
type CypressGlobal = {
  expose: (k: string) => unknown
}
type G = {
  Cypress?: CypressGlobal
  cy?: { task: (name: string, arg: unknown, o: { timeout: number }) => unknown }
  __e2eRestart?: () => Promise<void>
  process?: { env?: Record<string, string | undefined> }
}
const g = globalThis as unknown as G

export function restartConfigured(): boolean {
  return Boolean(g.Cypress?.expose('E2E_RESTART') ?? g.__e2eRestart)
}

/** Satu langkah dalam rantai tes: di Cypress berupa perintah antrean, di Node berupa promise. */
export function restartServer(timeoutMs = 120_000): unknown {
  if (g.cy) return g.cy.task('server:restart', null, { timeout: timeoutMs })
  return g.__e2eRestart?.()
}
