import { OFFICIAL_ROLES } from '../types/official.schema'

/**
 * Cermin aturan backend (service.CanManageRole): admin mengelola semua peran; dewan hanya wasit, operator, juri.
 * UI hanya menyembunyikan tombol, backend tetap yang memutuskan (membalas 403).
 */
const DEWAN_MANAGED = new Set(['wasit', 'operator', 'juri'])

export function canManageRole(actorRole: string, targetRole: string): boolean {
  const actor = actorRole.toLowerCase()
  if (actor === 'admin') return true
  if (actor === 'dewan') return DEWAN_MANAGED.has(targetRole.toLowerCase())
  return false
}

/** Peran yang boleh diberikan actor saat membuat atau mengubah akun. */
export function assignableRoles(actorRole: string): string[] {
  return OFFICIAL_ROLES.filter((r) => canManageRole(actorRole, r))
}

/** Apakah akun ini milik pengguna yang sedang masuk (menurut id, atau email bila sesi lama tanpa id). */
export function isSelf(
  me: { userId?: string; email?: string },
  official: { id: string; email: string },
): boolean {
  if (me.userId) return me.userId === official.id
  return !!me.email && me.email.toLowerCase() === official.email.toLowerCase()
}
