import { z } from 'zod'

export const createAthleteSchema = z.object({
  name: z.string().min(2, 'Nama atlet minimal 2 karakter').max(100, 'Maksimal 100 karakter'),
  gender: z.enum(['male', 'female', 'putra', 'putri']),
  team: z.string().min(2, 'Nama kontingen / perguruan minimal 2 karakter').max(100),
  contingent_id: z.string().optional(),
  birth_date: z
    .string()
    .regex(/^\d{4}-\d{2}-\d{2}$/, 'Tanggal lahir wajib diisi (YYYY-MM-DD)')
    .optional(),
  nik: z.string().max(50, 'NIK maksimal 50 karakter').optional(),
})

export const registerParticipantSchema = z.object({
  athlete_id: z.string().optional(),
  name: z.string().min(2, 'Nama atlet minimal 2 karakter').optional(),
  gender: z.enum(['male', 'female', 'putra', 'putri']).optional(),
  team: z.string().min(2, 'Nama tim / perguruan minimal 2 karakter').optional(),
  birth_date: z.string().optional(),
  nik: z.string().optional(),
  /** Kategori resmi (golongan umur + kelas bobot). Bila diisi, weight_class diambil dari kategori. */
  category_id: z.string().optional(),
  weight_class: z.string().optional(),
})

export const verifyParticipantSchema = z.object({
  status: z.enum(['verified', 'rejected']),
  actual_weight: z.number().positive('Berat badan harus lebih dari 0').max(300).optional(),
  docs_verified: z.record(z.string(), z.boolean()).optional(),
})

export type CreateAthleteFormValues = z.infer<typeof createAthleteSchema>
export type RegisterParticipantPayload = z.infer<typeof registerParticipantSchema>
export type VerifyParticipantPayload = z.infer<typeof verifyParticipantSchema>

export type Gender = 'male' | 'female' | 'putra' | 'putri'

export type ParticipantStatus =
  'pending' | 'verified' | 'rejected' | 'registered' | 'overweight' | 'underweight'

export interface Athlete {
  id: string
  contingent_id?: string
  name: string
  gender: Gender
  team: string
  birth_date?: string
  nik?: string
  created_at: string
}

export interface Contingent {
  id: string
  name: string
  pic_name: string
  email: string
  phone: string
  created_at: string
}

export interface Participant {
  id: string
  competition_id: string
  athlete_id: string
  contingent_id?: string
  name: string
  club?: string
  weight_class: string
  gender: Gender
  registration_status: ParticipantStatus
  category_id?: string
  category_name?: string
  age_group?: string
  birth_date?: string
  nik?: string
  /**
   * Pemeriksaan umur vs golongan kategori (server): `missing_birth_date` = tanggal lahir belum diisi
   * (umumnya peserta lama), `age_mismatch` = usia tidak masuk golongan. Kosong bila sesuai.
   */
  age_check?: 'missing_birth_date' | 'age_mismatch'
  actual_weight?: number | null
  docs_verified?: Record<string, boolean>
  created_at: string
}

/** Respons POST /competitions/{comp_id}/athletes */
export interface RegisterParticipantResult {
  competition_id: string
  athlete_id: string
  contingent_id: string
  category_id?: string
  weight_class: string
  status: string
}

/** Respons PUT /competitions/{comp_id}/participants/{id}/verify */
export interface VerifyParticipantResult {
  participant_id: string
  /** Status akhir; bisa 'overweight' / 'underweight' walau diminta 'verified'. */
  status: ParticipantStatus
  reason?: string
  /** Catatan bagan otomatis setelah peserta lolos ("" = tidak ada): bagan dibentuk ulang, atau terkunci/berjalan. */
  bracket_note?: string
}
