/**
 * Akun seed untuk pengujian E2E nyata (tanpa mock).
 * Sumber: docs/CREDENTIALS.md (dibuat oleh `go run ./cmd/seed/main.go`).
 * Hanya untuk database development/staging. JANGAN diarahkan ke produksi.
 */
export interface EmailAccount {
  email: string
  password: string
  role: 'admin' | 'dewan' | 'wasit' | 'operator' | 'contingent'
  name?: string
}

export const ADMIN: EmailAccount = {
  email: 'admin@silat.id',
  password: 'DemoPass_2026',
  role: 'admin',
  name: 'Administrator Utama',
}

export const DEWAN: EmailAccount = {
  email: 'dewan@silat.id',
  password: 'DemoPass_2026',
  role: 'dewan',
}

export const KONTINGEN: EmailAccount = {
  email: 'tapaksuci@contingent.id',
  password: 'DemoPass_2026',
  role: 'contingent',
}

/** Kontingen kedua, dipakai untuk menguji isolasi data antar kontingen. */
export const KONTINGEN_PSHT: EmailAccount = {
  email: 'psht@contingent.id',
  password: 'DemoPass_2026',
  role: 'contingent',
}
