import { apiClient } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'
import type {
  LoginFormValues,
  JudgeLoginFormValues,
  ContingentRegisterFormValues,
  CreatePinFormValues,
} from '../types/auth.schema'
import type {
  OfficialLoginResponse,
  JudgeLoginResponse,
  ContingentLoginResponse,
  ContingentRegisterResponse,
  CreatePinResponse,
} from '@/types/auth'

export const authService = {
  /**
   * Official Login (Admin, Dewan, Wasit, Operator)
   */
  async officialLogin(payload: LoginFormValues): Promise<OfficialLoginResponse> {
    const res = await apiClient.post<any, { success: boolean; data: OfficialLoginResponse }>(
      ENDPOINTS.AUTH.OFFICIAL_LOGIN,
      payload
    )
    return res.data
  },

  /**
   * Judge PIN Login
   */
  async judgeLogin(payload: JudgeLoginFormValues): Promise<JudgeLoginResponse> {
    const res = await apiClient.post<any, { success: boolean; data: JudgeLoginResponse }>(
      ENDPOINTS.AUTH.JUDGE_LOGIN,
      payload
    )
    return res.data
  },

  /**
   * Contingent Manager Login
   */
  async contingentLogin(payload: LoginFormValues): Promise<ContingentLoginResponse> {
    const res = await apiClient.post<any, { success: boolean; data: ContingentLoginResponse }>(
      ENDPOINTS.AUTH.CONTINGENT_LOGIN,
      payload
    )
    return res.data
  },

  /**
   * Contingent Self-Registration
   */
  async contingentRegister(payload: ContingentRegisterFormValues): Promise<ContingentRegisterResponse> {
    const res = await apiClient.post<any, { success: boolean; data: ContingentRegisterResponse }>(
      ENDPOINTS.AUTH.CONTINGENT_REGISTER,
      payload
    )
    return res.data
  },

  /**
   * Create Judge/Device PIN (Admin only)
   */
  async createPin(payload: CreatePinFormValues): Promise<CreatePinResponse> {
    const res = await apiClient.post<any, { success: boolean; data: CreatePinResponse }>(
      ENDPOINTS.AUTH.CREATE_PIN,
      payload
    )
    return res.data
  },

  /**
   * List Active Device PINs for an Arena (Admin only)
   */
  async listPins(competitionId: string, arenaId: string): Promise<any[]> {
    const url = `${ENDPOINTS.AUTH.PINS}?competition_id=${competitionId}&arena_id=${arenaId}`
    const res = await apiClient.get<any, { success: boolean; data: any[] }>(url)
    return res.data || []
  },

  /**
   * Logout user / device session
   */
  async logout(): Promise<void> {
    try {
      await apiClient.post(ENDPOINTS.AUTH.LOGOUT)
    } catch {
      // Ignore network errors or expired session on logout
    }
  },
}
