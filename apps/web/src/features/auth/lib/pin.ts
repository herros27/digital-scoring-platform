import { AppApiError } from '@/api/error'
import { getErrorMessage } from '@/api/error'

/** PIN juri 6 digit; PIN wasit 8 digit (sama dengan auth.PINLength di backend). */
export const JUDGE_PIN_LENGTH = 6
export const REFEREE_PIN_LENGTH = 8
export const MAX_PIN_LENGTH = REFEREE_PIN_LENGTH

/** Panjang PIN yang disyaratkan untuk sebuah role pada form admin. */
export function pinLengthForRole(role: string): number {
  return role === 'wasit' || role === 'referee' ? REFEREE_PIN_LENGTH : JUDGE_PIN_LENGTH
}

/**
 * Login tablet tidak tahu role pemilik PIN, jadi PIN dianggap lengkap bila panjangnya
 * 6 (juri) atau 8 (wasit); server yang memutuskan PIN itu valid atau tidak.
 */
export function isCompletePin(pin: string): boolean {
  return /^\d+$/.test(pin) && (pin.length === JUDGE_PIN_LENGTH || pin.length === REFEREE_PIN_LENGTH)
}

/** Validasi PIN kustom pada form admin; mengembalikan pesan error atau null bila valid. */
export function validateCustomPin(role: string, pin: string): string | null {
  const want = pinLengthForRole(role)
  if (!/^\d+$/.test(pin)) return 'PIN hanya boleh berisi angka.'
  if (pin.length !== want) return `PIN untuk peran ini harus ${want} digit.`
  return null
}

/** Tampilkan PIN dengan spasi agar mudah dibaca: "123 456" atau "1111 0001". */
export function formatPin(raw: string): string {
  const clean = (raw || '').replace(/\s+/g, '')
  if (clean.length === JUDGE_PIN_LENGTH) return `${clean.slice(0, 3)} ${clean.slice(3)}`
  if (clean.length === REFEREE_PIN_LENGTH) return `${clean.slice(0, 4)} ${clean.slice(4)}`
  return clean || '------'
}

/** Pesan untuk pengguna saat login PIN gagal. */
export function pinLoginErrorMessage(error: unknown): string {
  if (error instanceof AppApiError) {
    if (error.code === 'INVALID_PIN') return 'PIN salah atau sudah kedaluwarsa.'
    // pesan server membedakan "perangkat ini" (tablet ini saja) dan "jaringan ini" (batas IP/global)
    if (error.code === 'TOO_MANY_ATTEMPTS') {
      const msg = error.message.trim().replace(/[.,\s]+$/, '')
      const head = msg.length > 8 ? msg[0].toUpperCase() + msg.slice(1) : 'Terlalu banyak PIN salah'
      return `${head}. Minta panitia memeriksa PIN bila terus gagal.`
    }
  }
  return getErrorMessage(error, 'Tidak dapat terhubung ke server. Periksa jaringan lalu coba lagi.')
}
