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
