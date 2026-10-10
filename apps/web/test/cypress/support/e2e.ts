import './commands'
import './slow-mo'

// Semua tes berjalan terhadap backend NYATA (lewat proxy Vite /api -> :8080).
// Pastikan backend + seed sudah jalan sebelum menjalankan Cypress.
before(() => {
  cy.request({ url: '/api/v1/health', failOnStatusCode: false }).then((res) => {
    expect(
      res.status,
      'Backend harus hidup (go run ./cmd/server) dan Vite dev server di :5173. Seed: go run ./cmd/seed/main.go',
    ).to.eq(200)
  })
})

// Build produksi (nginx/Docker) mendaftarkan service worker PWA. Setelah kunjungan pertama, halaman berikutnya disajikan
// dari cache service worker sehingga tidak lewat proksi Cypress, dan tes UI setelah yang pertama gagal. Tes Cypress
// tidak menguji PWA (itu tugas Playwright, `npm run test:pw:pwa`), jadi pendaftarannya dimatikan di sini.
Cypress.on('window:before:load', (win) => {
  const proto = win.Navigator.prototype as unknown as Record<string, unknown>
  if ('serviceWorker' in proto) delete proto.serviceWorker
})
