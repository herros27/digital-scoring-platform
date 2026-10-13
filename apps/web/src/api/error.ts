/**
 * Normalized API Error interface
 */
export interface NormalizedApiError {
  statusCode: number
  code: string
  message: string
  fieldErrors?: Record<string, string[]>
}

export class AppApiError extends Error implements NormalizedApiError {
  statusCode: number
  code: string
  fieldErrors?: Record<string, string[]>

  constructor(error: NormalizedApiError) {
    super(error.message)
    this.name = 'AppApiError'
    this.statusCode = error.statusCode
    this.code = error.code
    this.fieldErrors = error.fieldErrors
  }
}

/**
 * Ambil pesan yang aman ditampilkan dari nilai `unknown` hasil `catch`.
 */
export function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) return error.message
  if (typeof error === 'string' && error) return error
  return fallback
}

/**
 * Deteksi error "tidak ditemukan" dari backend (HTTP 404 atau pesan berbahasa Indonesia).
 */
export function isNotFoundError(error: unknown): boolean {
  if (error instanceof AppApiError && error.statusCode === 404) return true
  return error instanceof Error && error.message.includes('tidak ditemukan')
}
