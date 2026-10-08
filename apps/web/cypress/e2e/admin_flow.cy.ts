describe('Admin Flow - Competition & Tournament Operations', () => {
  beforeEach(() => {
    // 1. Fallback default interceptor untuk semua API
    cy.intercept('GET', '**/api/v1/**', {
      statusCode: 200,
      body: { success: true, data: [] },
    })

    // 2. Mock Auth Login
    cy.intercept('POST', '**/api/v1/auth/login', {
      statusCode: 200,
      body: {
        success: true,
        data: {
          token: 'valid-test-jwt-token',
          user: {
            id: 'admin-1',
            name: 'Administrator Utama',
            email: 'admin@silat.id',
            role: 'admin',
          },
        },
      },
    }).as('loginReq')

    // 3. Mock Rulesets List
    cy.intercept('GET', /\/api\/v1\/rulesets(\?.*)?$/, {
      statusCode: 200,
      body: {
        success: true,
        data: [
          {
            id: 'ruleset-ipsi-2024',
            name: 'IPSI 2024 v2024.1',
            version: '2024.1',
            status: 'active',
            rounds: 3,
            round_duration_seconds: 120,
          },
        ],
      },
    }).as('getRulesets')

    // 4. Mock Competitions List
    cy.intercept('GET', /\/api\/v1\/competitions(\?.*)?$/, {
      statusCode: 200,
      body: {
        success: true,
        data: [
          {
            id: 'b26d7145-d341-4795-ac58-bff86d13cbf8',
            name: 'Kejuaraan Nasional Pencak Silat Menpora 2026',
            location: 'GOR POPKI Cibubur, Jakarta Timur',
            start_date: '2026-10-10',
            end_date: '2026-10-15',
            ruleset_name: 'IPSI 2024',
            status: 'active',
          },
        ],
        pagination: {
          page: 1,
          limit: 20,
          total: 1,
        },
      },
    }).as('getCompetitions')

    // 5. Mock Arenas
    cy.intercept('GET', /\/api\/v1\/competitions\/[^/]+\/arenas(\?.*)?$/, {
      statusCode: 200,
      body: {
        success: true,
        data: [
          { id: 'arena-a', name: 'Gelanggang A' },
          { id: 'arena-b', name: 'Gelanggang B' },
        ],
      },
    }).as('getArenas')

    // 6. Mock Categories
    cy.intercept('GET', /\/api\/v1\/competitions\/[^/]+\/categories(\?.*)?$/, {
      statusCode: 200,
      body: {
        success: true,
        data: [
          {
            id: 'cat-1',
            name: 'Tanding Dewasa Putra - Kelas A (45 - 50 Kg)',
            gender: 'male',
            weight_class: 'A',
            age_group: 'dewasa',
          },
        ],
      },
    }).as('getCategories')

    // 7. Mock PINs
    cy.intercept('GET', /\/api\/v1\/auth\/pins(\?.*)?$/, {
      statusCode: 200,
      body: {
        success: true,
        data: [
          { id: 'pin-1', seat_number: 1, role: 'judge', pin: '123456' },
          { id: 'pin-2', seat_number: 2, role: 'judge', pin: '234567' },
          { id: 'pin-w', seat_number: 0, role: 'wasit', pin: '345678' },
        ],
      },
    }).as('getPins')

    // 8. Mock Create Competition
    cy.intercept('POST', /\/api\/v1\/competitions$/, {
      statusCode: 201,
      body: {
        success: true,
        data: {
          id: 'comp-12345678',
          name: 'KEJURNAS PENCAK SILAT DEWASA PIALA PRESIDEN 2026',
          location: 'Padepokan Pencak Silat TMII, Jakarta Timur',
          start_date: '2026-10-08',
          end_date: '2026-10-15',
          ruleset_name: 'IPSI 2024 v2024.1',
          ruleset_id: 'ruleset-ipsi-2024',
          status: 'draft',
        },
      },
    }).as('createComp')

    // Login via UI
    cy.visit('/login')
    cy.get('button.demo-chip').contains(/Admin Pertandingan/i).click()
    cy.get('button[type="submit"]').contains(/Masuk ke Akun/i).click()
    cy.wait('@loginReq')
    cy.url().should('include', '/ops')
  })

  it('bisa membuka panel operations dan berpindah ke tab Kategori Tanding', () => {
    cy.get('aside button.ops-nav__item').contains('Kategori Tanding').click()

    // Verifikasi Header berganti ke KATEGORI TANDING
    cy.contains('h1, h2', /KATEGORI TANDING/i).should('be.visible')

    // Verifikasi elemen aksi manajemen kategori tersedia
    cy.get('button').contains(/Tambah Kategori/i).should('be.visible')
    cy.get('button').contains(/Terapkan Standar PB IPSI/i).should('be.visible')
  })

  it('bisa berpindah ke tab Bagan Pertandingan dan melihat bagan turnamen', () => {
    cy.get('aside button.ops-nav__item').contains('Bagan Pertandingan').click()

    // Verifikasi Header berganti ke BAGAN PERTANDINGAN
    cy.contains('h1, h2', /BAGAN PERTANDINGAN/i).should('be.visible')

    // Verifikasi filter kategori & kontrol bagan aktif
    cy.contains(/KATEGORI & KELAS TANDING/i).should('be.visible')
    cy.get('select.filter-select').should('be.visible')
  })

  it('bisa membuka menu Perangkat & PIN dan melihat manajemen PIN juri', () => {
    cy.get('aside button.ops-nav__item').contains('Perangkat & PIN').click()

    // Verifikasi Header dan Banner Manajemen PIN aktif
    cy.contains('h1, h2', /PERANGKAT & PIN JURI|MANAJEMEN PERANGKAT/i).should('be.visible')
    cy.contains(/KONTROL AKSES TABLET GELANGGANG/i).should('be.visible')

    // Verifikasi Tombol Aksi Utama
    cy.get('button').contains(/Generate Ulang Semua PIN/i).should('be.visible')
    cy.get('button').contains(/Buat PIN Kustom/i).should('be.visible')

    // Verifikasi Kontrol Format Juri
    cy.get('button.format-btn').contains(/5 JURI/i).should('be.visible')
    cy.get('button.format-btn').contains(/3 JURI/i).should('be.visible')
  })

  it('bisa berganti format juri pada manajemen PIN', () => {
    cy.get('aside button.ops-nav__item').contains('Perangkat & PIN').click()

    // Ganti format juri ke 3 Juri
    cy.get('button.format-btn').contains(/3 JURI/i).click()
    cy.get('button.format-btn').contains(/3 JURI/i).should('have.class', 'format-btn--active')

    // Kembalikan ke format 5 Juri
    cy.get('button.format-btn').contains(/5 JURI/i).click()
    cy.get('button.format-btn').contains(/5 JURI/i).should('have.class', 'format-btn--active')
  })
})