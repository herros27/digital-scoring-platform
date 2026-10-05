// ponytail: type-safe wrapper bridging to feature-based authService
import { authService } from '@/features/auth/api/authService'
import type {
  JudgeLoginRequest,
  JudgeLoginResponse,
  OfficialLoginRequest,
  OfficialLoginResponse,
  ContingentRegisterRequest,
  ContingentRegisterResponse,
  ContingentLoginRequest,
  ContingentLoginResponse,
  CreatePinRequest,
  CreatePinResponse,
} from '@/types/auth'

export const authApi = {
  /**
   * 1. POST /api/v1/auth/judge-login
   */
  judgeLogin: (payload: JudgeLoginRequest): Promise<JudgeLoginResponse> =>
    authService.judgeLogin(payload),

  /**
   * 2. POST /api/v1/auth/login
   */
  login: (payload: OfficialLoginRequest): Promise<OfficialLoginResponse> =>
    authService.officialLogin(payload),

  /**
   * 3. POST /api/v1/auth/contingent/register
   */
  contingentRegister: (payload: ContingentRegisterRequest): Promise<ContingentRegisterResponse> =>
    authService.contingentRegister(payload),

  /**
   * 4. POST /api/v1/auth/contingent/login
   */
  contingentLogin: (payload: ContingentLoginRequest): Promise<ContingentLoginResponse> =>
    authService.contingentLogin(payload),

  /**
   * 5. POST /api/v1/auth/create-pin
   */
  createPin: (payload: CreatePinRequest): Promise<CreatePinResponse> =>
    authService.createPin(payload),

  /**
   * 6. GET /api/v1/auth/pins
   */
  listPins: (competitionId: string, arenaId: string): Promise<any[]> =>
    authService.listPins(competitionId, arenaId),

  /**
   * 7. POST /api/v1/auth/logout
   */
  logout: (): Promise<void> => authService.logout(),
}
