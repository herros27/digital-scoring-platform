// ponytail: auth request/response types mirroring Go backend endpoints in internal/api/handler/auth.go, official.go, contingent.go
export interface JudgeLoginRequest {
  competition_id: string
  arena_id: string
  pin: string
}

export interface JudgeLoginResponse {
  token: string
  role: string
  arena_id: string
  competition_id: string
  seat_number?: number
}

export interface OfficialLoginRequest {
  email: string
  password: string
}

export interface OfficialUser {
  id: string
  name: string
  email: string
  role: string
}

export interface OfficialLoginResponse {
  token: string
  user: OfficialUser
}

export interface ContingentRegisterRequest {
  name: string
  pic_name: string
  email: string
  phone: string
  password: string
}

export interface ContingentInfo {
  id: string
  name: string
  pic_name: string
  email: string
  phone: string
  created_at?: string
}

export type ContingentRegisterResponse = ContingentInfo

export interface ContingentLoginRequest {
  email: string
  password: string
}

export interface ContingentLoginResponse {
  token: string
  contingent: ContingentInfo
}

export interface CreatePinRequest {
  competition_id: string
  arena_id: string
  role: string
  seat_number?: number
  pin?: string
  ttl_hours?: number
}

export interface CreatePinResponse {
  id: string
  competition_id: string
  arena_id: string
  role: string
  seat_number?: number
  pin: string
  expires_at: string
}
