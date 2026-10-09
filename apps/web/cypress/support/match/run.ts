/**
 * Menjalankan fungsi async di dalam tes. Di Cypress dibungkus `cy.then` (menunggu promise, dengan timeout
 * sendiri) sehingga urutan perintah tetap benar; di runner Node headless dipanggil langsung.
 * Pakai SATU kali per hook/tes dan letakkan seluruh alur (await berurutan) di dalam fungsinya.
 */
export function runAsync(fn: () => Promise<unknown>, timeoutMs = 90_000): unknown {
  const c = (
    globalThis as {
      cy?: { then: (o: { timeout: number }, f: () => Promise<unknown>) => unknown }
    }
  ).cy
  return c ? c.then({ timeout: timeoutMs }, () => fn()) : fn()
}
