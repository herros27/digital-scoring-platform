import { http } from '@/api/client'
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
  DevicePin,
} from '@/types/auth'

export const authService = {
  /**
   * Official Login (Admin, Dewan, Wasit, Operator)
   */
  async officialLogin(payload: LoginFormValues): Promise<OfficialLoginResponse> {
    const res = await http.post<OfficialLoginResponse>(
      ENDPOINTS.AUTH.OFFICIAL_LOGIN,
      payload
    )
    return res.data
  },

  /**
   * Judge PIN Login
   */
  async judgeLogin(payload: JudgeLoginFormValues): Promise<JudgeLoginResponse> {
    const res = await http.post<JudgeLoginResponse>(
      ENDPOINTS.AUTH.JUDGE_LOGIN,
      payload
    )
    return res.data
  },

  /**
   * Contingent Manager Login
   */
  async contingentLogin(payload: LoginFormValues): Promise<ContingentLoginResponse> {
    const res = await http.post<ContingentLoginResponse>(
      ENDPOINTS.AUTH.CONTINGENT_LOGIN,
      payload
    )
    return res.data
  },

  /**
   * Contingent Self-Registration
   */
  async contingentRegister(payload: ContingentRegisterFormValues): Promise<ContingentRegisterResponse> {
    const res = await http.post<ContingentRegisterResponse>(
      ENDPOINTS.AUTH.CONTINGENT_REGISTER,
      payload
    )
    return res.data
  },

  /**
   * Create Judge/Device PIN (Admin only)
   */
  async createPin(payload: CreatePinFormValues): Promise<CreatePinResponse> {
    const res = await http.post<CreatePinResponse>(
      ENDPOINTS.AUTH.CREATE_PIN,
      payload
    )
    return res.data
  },

  /**
   * List Active Device PINs for an Arena (Admin only)
   */
  async listPins(competitionId: string, arenaId: string): Promise<DevicePin[]> {
    const url = `${ENDPOINTS.AUTH.PINS}?competition_id=${competitionId}&arena_id=${arenaId}`
    const res = await http.get<DevicePin[]>(url)
    return res.data || []
  },

  /**
   * Logout user / device session
   */
  async logout(): Promise<void> {
    try {
      await http.post<unknown>(ENDPOINTS.AUTH.LOGOUT)
    } catch {
      // Ignore network errors or expired session on logout
    }
  },
}
