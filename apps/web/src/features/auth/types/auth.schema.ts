import { z } from 'zod'

export const loginSchema = z.object({
  email: z.string().email('Format email tidak valid'),
  password: z.string().min(1, 'Kata sandi wajib diisi'),
})

export const judgeLoginSchema = z.object({
  competition_id: z.string().min(1, 'ID Kompetisi tidak valid'),
  arena_id: z.string().min(1, 'ID Arena tidak valid'),
  pin: z.string().min(4, 'PIN minimal 4 digit').max(6, 'PIN maksimal 6 digit'),
})

export const contingentRegisterSchema = z.object({
  name: z.string().min(3, 'Nama kontingen minimal 3 karakter'),
  pic_name: z.string().min(2, 'Nama penanggung jawab wajib diisi').optional(),
  pic: z.string().min(2, 'Nama penanggung jawab wajib diisi').optional(),
  email: z.string().email('Format email tidak valid'),
  phone: z.string().min(8, 'Nomor telepon tidak valid'),
  password: z.string().min(6, 'Kata sandi minimal 6 karakter'),
})

export const createPinSchema = z.object({
  competition_id: z.string().min(1, 'ID Kompetisi wajib diisi'),
  arena_id: z.string().min(1, 'ID Arena wajib diisi'),
  role: z.string().default('judge'),
  seat_number: z.number().int().min(0).max(5).optional(),
  pin: z.string().optional(),
  ttl_hours: z.number().int().min(1).max(72).optional(),
})

export type LoginFormValues = z.infer<typeof loginSchema>
export type JudgeLoginFormValues = z.infer<typeof judgeLoginSchema>
export type ContingentRegisterFormValues = z.infer<typeof contingentRegisterSchema>
export type CreatePinFormValues = z.infer<typeof createPinSchema>
