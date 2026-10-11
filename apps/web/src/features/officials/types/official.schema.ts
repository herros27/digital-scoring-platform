import { z } from 'zod'

export const MIN_PASSWORD_LEN = 8

/** Peran yang dapat dipilih di UI. (`juri` ada di backend tetapi juri masuk lewat PIN, bukan akun email.) */
export const OFFICIAL_ROLES = ['admin', 'dewan', 'wasit', 'operator'] as const
export type OfficialRole = (typeof OFFICIAL_ROLES)[number]

export const ROLE_LABELS: Record<string, string> = {
  admin: 'Admin',
  dewan: 'Pengawas Pertandingan',
  wasit: 'Operator Meja',
  operator: 'Staf Administrasi',
  juri: 'Juri',
}

export interface Official {
  id: string
  name: string
  email: string
  role: string
  created_at: string
}

const passwordField = z
  .string()
  .min(MIN_PASSWORD_LEN, `Password minimal ${MIN_PASSWORD_LEN} karakter`)
  .max(72, 'Password maksimal 72 karakter')

export const createOfficialSchema = z.object({
  name: z.string().trim().min(2, 'Nama minimal 2 karakter').max(100, 'Nama maksimal 100 karakter'),
  email: z.string().trim().email('Format email tidak valid'),
  password: passwordField,
  role: z.enum(OFFICIAL_ROLES, { message: 'Pilih peran' }),
})

export const updateOfficialSchema = z.object({
  name: z.string().trim().min(2, 'Nama minimal 2 karakter').max(100, 'Nama maksimal 100 karakter'),
  role: z.enum(OFFICIAL_ROLES, { message: 'Pilih peran' }),
})

export const resetPasswordSchema = z.object({ password: passwordField })

export const changeOwnPasswordSchema = z
  .object({
    current_password: z.string().min(1, 'Password lama wajib diisi'),
    new_password: passwordField,
    confirm: z.string(),
  })
  .refine((v) => v.new_password === v.confirm, {
    message: 'Konfirmasi password tidak sama',
    path: ['confirm'],
  })
  .refine((v) => v.new_password !== v.current_password, {
    message: 'Password baru harus berbeda dari yang lama',
    path: ['new_password'],
  })

export type CreateOfficialInput = z.infer<typeof createOfficialSchema>
export type UpdateOfficialInput = z.infer<typeof updateOfficialSchema>
