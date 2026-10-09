import { http } from '@/api/client'
import { ENDPOINTS } from '@/api/endpoints'
import type { Official } from '../types/official.schema'

export const officialService = {
  async list(): Promise<Official[]> {
    const res = await http.get<Official[]>(ENDPOINTS.OFFICIALS.LIST)
    return res.data || []
  },
  async create(body: {
    name: string
    email: string
    password: string
    role: string
  }): Promise<Official> {
    const res = await http.post<Official>(ENDPOINTS.OFFICIALS.CREATE, body)
    return res.data
  },
  async update(id: string, body: { name: string; role: string }): Promise<Official> {
    const res = await http.put<Official>(ENDPOINTS.OFFICIALS.UPDATE(id), body)
    return res.data
  },
  async remove(id: string): Promise<void> {
    await http.delete<unknown>(ENDPOINTS.OFFICIALS.DELETE(id))
  },
  /** Admin/dewan mengganti password akun lain tanpa password lama. */
  async resetPassword(id: string, password: string): Promise<void> {
    await http.put<unknown>(ENDPOINTS.OFFICIALS.PASSWORD(id), { password })
  },
  /** Admin/dewan mereset password manajer kontingen (lupa password) tanpa password lama. */
  async resetContingentPassword(id: string, password: string): Promise<void> {
    await http.put<unknown>(ENDPOINTS.CONTINGENTS.PASSWORD(id), { password })
  },
  /** Mengganti password sendiri; password lama wajib benar. */
  async changeOwnPassword(current_password: string, new_password: string): Promise<void> {
    await http.post<unknown>(ENDPOINTS.AUTH.CHANGE_PASSWORD, { current_password, new_password })
  },
}
