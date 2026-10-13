import { ADMIN, DEWAN } from '../accounts.ts'
import { loginOfficial } from './kit.ts'

let cached: Promise<string> | null = null

/** Token admin untuk menyiapkan data. Satu login per spec (login dibatasi 10/menit). */
export function adminToken(): Promise<string> {
  cached ??= loginOfficial(ADMIN.email, ADMIN.password)
  return cached
}

export const dewanToken = (): Promise<string> => loginOfficial(DEWAN.email, DEWAN.password)
export const operatorToken = (): Promise<string> =>
  loginOfficial('operator@silat.id', 'DemoPass_2026')
