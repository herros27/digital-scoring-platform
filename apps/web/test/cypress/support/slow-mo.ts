/**
 * Mode "slow motion" supaya jalannya tes mudah diikuti mata.
 * Aktif hanya bila SLOW_MO (milidetik) diisi lewat flag CLI, mis. `cypress open --expose SLOW_MO=800`. Default: mati (tes tetap cepat).
 * Jeda disisipkan SEBELUM click, type, clear, visit, dan reload.
 *
 * Catatan teknis: jeda dibuat dengan promise biasa, BUKAN cy.wait(). Command bawaan Cypress
 * mengembalikan promise, dan Cypress melarang mencampurnya dengan perintah cy.* di dalam override.
 */
const delay = Number(Cypress.expose('SLOW_MO') ?? 0)

if (Number.isFinite(delay) && delay > 0) {
  // Tipe overload bawaan Cypress sulit dipenuhi untuk overwrite generik, jadi di-cast.
  type Fn = (...args: unknown[]) => unknown

  const sleep = () => new Cypress.Promise<void>((resolve) => setTimeout(resolve, delay))

  const slowed =
    (transform?: (args: unknown[]) => unknown[]) =>
    (original: Fn, ...args: unknown[]) =>
      sleep().then(() => original(...(transform ? transform(args) : args)))

  // args untuk perintah dengan subject: [subject, ...opsi]
  Cypress.Commands.overwrite('click', slowed() as never)
  Cypress.Commands.overwrite('clear', slowed() as never)
  Cypress.Commands.overwrite('visit', slowed() as never)
  Cypress.Commands.overwrite('reload', slowed() as never)

  // ketikan per huruf ikut diperlambat sedikit (maks 120 ms/huruf)
  Cypress.Commands.overwrite(
    'type',
    slowed(([subject, text, options]) => [
      subject,
      text,
      { delay: Math.min(120, Math.round(delay / 6)), ...(options as object | undefined) },
    ]) as never,
  )
}
