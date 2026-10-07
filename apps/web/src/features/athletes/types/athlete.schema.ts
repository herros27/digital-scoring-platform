import { z } from 'zod'

export const createAthleteSchema = z.object({
  name: z.string().min(2, 'Nama atlet minimal 2 karakter').max(100, 'Maksimal 100 karakter'),
  gender: z.enum(['male', 'female', 'putra', 'putri']),
  team: z.string().min(2, 'Nama kontingen / perguruan minimal 2 karakter').max(100),
  contingent_id: z.string().optional(),
})

export const registerParticipantSchema = z.object({
  athlete_id: z.string().optional(),
  name: z.string().min(2, 'Nama atlet minimal 2 karakter').optional(),
  gender: z.enum(['male', 'female', 'putra', 'putri']).optional(),
  team: z.string().min(2, 'Nama tim / perguruan minimal 2 karakter').optional(),
  weight_class: z.string().min(1, 'Kelas tanding wajib dipilih'),
})

export const verifyParticipantSchema = z.object({
  status: z.enum(['verified', 'rejected']),
})

export type CreateAthleteFormValues = z.infer<typeof createAthleteSchema>
export type RegisterParticipantPayload = z.infer<typeof registerParticipantSchema>
export type VerifyParticipantPayload = z.infer<typeof verifyParticipantSchema>

export type Gender = 'male' | 'female' | 'putra' | 'putri'

export type ParticipantStatus = 'pending' | 'verified' | 'rejected' | 'registered'

export interface Athlete {
  id: string
  contingent_id?: string
  name: string
  gender: Gender
  team: string
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
  created_at: string
}

/** Respons POST /competitions/{comp_id}/athletes */
export interface RegisterParticipantResult {
  competition_id: string
  athlete_id: string
  contingent_id: string
  weight_class: string
  status: string
}

/** Respons PUT /competitions/{comp_id}/participants/{id}/verify */
export interface VerifyParticipantResult {
  participant_id: string
  status: string
}
