import { z } from 'zod'

export const createCategorySchema = z.object({
  name: z.string().min(2, 'Nama kategori minimal 2 karakter').max(100, 'Maksimal 100 karakter'),
  gender: z.enum(['male', 'female', 'putra', 'putri']),
  weight_class: z.string().min(1, 'Kelas tanding wajib dipilih'),
  ruleset_id: z.string().optional(),
})

export const categoryFilterSchema = z.object({
  gender: z.enum(['all', 'male', 'female', 'putra', 'putri']).default('all'),
  search: z.string().optional(),
})

export type CreateCategoryFormValues = z.infer<typeof createCategorySchema>
export type CategoryFilterValues = z.infer<typeof categoryFilterSchema>

export interface Category {
  id: string
  competition_id: string
  name: string
  gender: 'male' | 'female' | 'putra' | 'putri'
  weight_class: string
  ruleset_id?: string
  created_at: string
  // UI extended metadata
  quota?: number
  participant_count?: number
  status?: 'ready' | 'draft' | 'in_progress' | 'completed'
}

export interface StandardClassPreset {
  code: string
  label: string
  minWeight: number
  maxWeight: number
  gender: 'male' | 'female'
  categoryName: string
}

export const IPSI_STANDARD_PRESETS: StandardClassPreset[] = [
  // Dewasa Putra
  { code: 'A', label: 'Kelas A (45–50 Kg)', minWeight: 45, maxWeight: 50, gender: 'male', categoryName: 'Tanding Dewasa Putra - Kelas A (45–50 Kg)' },
  { code: 'B', label: 'Kelas B (50–55 Kg)', minWeight: 50, maxWeight: 55, gender: 'male', categoryName: 'Tanding Dewasa Putra - Kelas B (50–55 Kg)' },
  { code: 'C', label: 'Kelas C (55–60 Kg)', minWeight: 55, maxWeight: 60, gender: 'male', categoryName: 'Tanding Dewasa Putra - Kelas C (55–60 Kg)' },
  { code: 'D', label: 'Kelas D (60–65 Kg)', minWeight: 60, maxWeight: 65, gender: 'male', categoryName: 'Tanding Dewasa Putra - Kelas D (60–65 Kg)' },
  { code: 'E', label: 'Kelas E (65–70 Kg)', minWeight: 65, maxWeight: 70, gender: 'male', categoryName: 'Tanding Dewasa Putra - Kelas E (65–70 Kg)' },
  { code: 'F', label: 'Kelas F (70–75 Kg)', minWeight: 70, maxWeight: 75, gender: 'male', categoryName: 'Tanding Dewasa Putra - Kelas F (70–75 Kg)' },
  { code: 'G', label: 'Kelas G (75–80 Kg)', minWeight: 75, maxWeight: 80, gender: 'male', categoryName: 'Tanding Dewasa Putra - Kelas G (75–80 Kg)' },
  { code: 'H', label: 'Kelas H (80–85 Kg)', minWeight: 80, maxWeight: 85, gender: 'male', categoryName: 'Tanding Dewasa Putra - Kelas H (80–85 Kg)' },
  { code: 'I', label: 'Kelas I (85–90 Kg)', minWeight: 85, maxWeight: 90, gender: 'male', categoryName: 'Tanding Dewasa Putra - Kelas I (85–90 Kg)' },
  { code: 'J', label: 'Kelas J (90–95 Kg)', minWeight: 90, maxWeight: 95, gender: 'male', categoryName: 'Tanding Dewasa Putra - Kelas J (90–95 Kg)' },
  { code: 'OPEN', label: 'Kelas Bebas (>95 Kg)', minWeight: 95, maxWeight: 130, gender: 'male', categoryName: 'Tanding Dewasa Putra - Kelas Bebas (>95 Kg)' },

  // Dewasa Putri
  { code: 'A', label: 'Kelas A (45–50 Kg)', minWeight: 45, maxWeight: 50, gender: 'female', categoryName: 'Tanding Dewasa Putri - Kelas A (45–50 Kg)' },
  { code: 'B', label: 'Kelas B (50–55 Kg)', minWeight: 50, maxWeight: 55, gender: 'female', categoryName: 'Tanding Dewasa Putri - Kelas B (50–55 Kg)' },
  { code: 'C', label: 'Kelas C (55–60 Kg)', minWeight: 55, maxWeight: 60, gender: 'female', categoryName: 'Tanding Dewasa Putri - Kelas C (55–60 Kg)' },
  { code: 'D', label: 'Kelas D (60–65 Kg)', minWeight: 60, maxWeight: 65, gender: 'female', categoryName: 'Tanding Dewasa Putri - Kelas D (60–65 Kg)' },
  { code: 'E', label: 'Kelas E (65–70 Kg)', minWeight: 65, maxWeight: 70, gender: 'female', categoryName: 'Tanding Dewasa Putri - Kelas E (65–70 Kg)' },
  { code: 'F', label: 'Kelas F (70–75 Kg)', minWeight: 70, maxWeight: 75, gender: 'female', categoryName: 'Tanding Dewasa Putri - Kelas F (70–75 Kg)' },
  { code: 'OPEN', label: 'Kelas Bebas (>75 Kg)', minWeight: 75, maxWeight: 110, gender: 'female', categoryName: 'Tanding Dewasa Putri - Kelas Bebas (>75 Kg)' },
]
