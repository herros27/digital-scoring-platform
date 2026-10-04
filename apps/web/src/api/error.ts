/**
 * Normalized API Error interface according to FRONTEND_SPECIFICATION.md
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
