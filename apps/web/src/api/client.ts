import axios, { type AxiosInstance, type InternalAxiosRequestConfig, type AxiosResponse } from 'axios'
import { generateUUID } from '@/lib/uuid'
import { AppApiError, type NormalizedApiError } from './error'

const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

/**
 * Enterprise Axios Client for REST API services
 */
export const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE,
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request Interceptor: Inject JWT and distributed tracing X-Request-Id
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = sessionStorage.getItem('ds_token')
    if (token && !config.headers['Authorization']) {
      config.headers['Authorization'] = `Bearer ${token}`
    }
    if (!config.headers['X-Request-Id']) {
      config.headers['X-Request-Id'] = generateUUID()
    }
    return config
  },
  (error) => Promise.reject(error)
)

// Response Interceptor: Normalize responses and catch 401/403 errors
apiClient.interceptors.response.use(
  (response: AxiosResponse) => {
    // Unpack standard backend response envelope { success: true, data: ... }
    const body = response.data
    return body
  },
  (error) => {
    const status = error.response?.status || 500
    const data = error.response?.data

    const normalized: NormalizedApiError = {
      statusCode: status,
      code: data?.error?.code || data?.code || 'UNKNOWN_ERROR',
      message: data?.error?.message || data?.message || error.message || 'Terjadi kesalahan pada server',
      fieldErrors: data?.error?.details || data?.details,
    }

    // Auto-redirect if unauthorized and not on public auth views
    if (status === 401 && !window.location.pathname.startsWith('/login') && !window.location.pathname.startsWith('/judge/login')) {
      sessionStorage.removeItem('ds_token')
      sessionStorage.removeItem('ds_auth_session')
    }

    return Promise.reject(new AppApiError(normalized))
  }
)
