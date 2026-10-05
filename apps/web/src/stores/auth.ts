import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authService } from '@/features/auth/api/authService'

export interface AuthSession {
  token: string
  role: 'admin' | 'dewan' | 'wasit' | 'judge' | 'operator' | 'contingent' | 'public' | string
  seatNumber?: number
  arenaId?: string
  competitionId?: string
  name?: string
  email?: string
  contingentId?: string
}

const STORAGE_KEY = 'ds_auth_session'

function getSavedSession(): AuthSession | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export const useAuthStore = defineStore('auth', () => {
  const session = ref<AuthSession | null>(getSavedSession())

  const token = computed(() => session.value?.token || '')
  const role = computed(() => session.value?.role || 'public')
  const isAuthenticated = computed(() => !!session.value?.token)
  const isJudge = computed(() => session.value?.role === 'judge')
  const isReferee = computed(() => session.value?.role === 'wasit' || session.value?.role === 'admin')
  const isContingent = computed(() => session.value?.role === 'contingent')
  const isAdminOrDewan = computed(() => session.value?.role === 'admin' || session.value?.role === 'dewan')
  const seatNumber = computed(() => session.value?.seatNumber)
  const arenaId = computed(() => session.value?.arenaId || '')
  const competitionId = computed(() => session.value?.competitionId || '')
  const name = computed(() => session.value?.name || '')

  function setSession(data: AuthSession) {
    session.value = data
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(data))
    sessionStorage.setItem('ds_token', data.token)
  }

  function clearSession() {
    session.value = null
    sessionStorage.removeItem(STORAGE_KEY)
    sessionStorage.removeItem('ds_token')
    sessionStorage.removeItem('token')
  }

  function setCompetitionId(compId: string) {
    if (session.value) {
      session.value = { ...session.value, competitionId: compId }
    } else {
      session.value = { token: '', role: 'public', competitionId: compId }
    }
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(session.value))
  }

  async function logout() {
    try {
      await authService.logout()
    } catch {
      // ignore
    } finally {
      clearSession()
    }
  }

  return {
    session,
    token,
    role,
    isAuthenticated,
    isJudge,
    isReferee,
    name,
    isContingent,
    isAdminOrDewan,
    seatNumber,
    arenaId,
    competitionId,
    setSession,
    clearSession,
    setCompetitionId,
    logout,
  }
})
