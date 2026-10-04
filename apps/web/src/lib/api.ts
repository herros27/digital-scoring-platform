const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

export interface ApiResponse<T = unknown> {
  success: boolean
  data?: T
  error?: {
    code: string
    message: string
  }
}

export interface PagedResponse<T = unknown> {
  success: boolean
  data: T[]
  pagination: {
    page: number
    limit: number
    total: number
  }
}

export async function requestRaw<T = any>(path: string, init?: RequestInit): Promise<T> {
  const token = sessionStorage.getItem('ds_token')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(init?.headers as Record<string, string>),
  }

  if (token && !headers['Authorization']) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
  })

  const contentType = res.headers.get('content-type')
  const isJson = contentType && contentType.includes('application/json')
  const body = isJson ? await res.json() : await res.text()

  if (!res.ok) {
    const errorMsg = body?.error?.message || body?.message || (typeof body === 'string' ? body : `HTTP ${res.status}`)
    throw new Error(errorMsg)
  }

  return body as T
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const body = await requestRaw<any>(path, init)

  // If response follows standard { success: true, data: ... }
  if (body && typeof body === 'object' && 'data' in body && 'success' in body) {
    return body.data as T
  }

  return body as T
}

export const api = {
  get: <T>(path: string, init?: RequestInit) => request<T>(path, { ...init, method: 'GET' }),
  getPaged: <T>(path: string, init?: RequestInit) =>
    requestRaw<PagedResponse<T>>(path, { ...init, method: 'GET' }),
  post: <T>(path: string, body?: unknown, init?: RequestInit) =>
    request<T>(path, {
      ...init,
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    }),
  put: <T>(path: string, body?: unknown, init?: RequestInit) =>
    request<T>(path, {
      ...init,
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    }),
  delete: <T>(path: string, init?: RequestInit) => request<T>(path, { ...init, method: 'DELETE' }),
}
