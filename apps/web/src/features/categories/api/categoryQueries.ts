import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { categoryService } from './categoryService'
import type {
  CreateCategoryFormValues,
  RulesetAssignment,
  StandardClassPreset,
} from '../types/category.schema'

/**
 * Query Key Factory for Categories domain
 */
export const categoryKeys = {
  all: ['categories'] as const,
  lists: () => [...categoryKeys.all, 'list'] as const,
  list: (competitionId: string) => [...categoryKeys.lists(), competitionId] as const,
  details: () => [...categoryKeys.all, 'detail'] as const,
  detail: (competitionId: string, id: string) =>
    [...categoryKeys.details(), competitionId, id] as const,
}

/**
 * Fetch categories for a competition with TanStack Query
 */
export function useCategoriesQuery(competitionId: Ref<string>) {
  return useQuery({
    queryKey: computed(() => categoryKeys.list(competitionId.value)),
    queryFn: () => categoryService.getCategories(competitionId.value),
    enabled: computed(() => !!competitionId.value),
    staleTime: 1000 * 30, // 30s
  })
}

/**
 * Fetch single category detail
 */
export function useCategoryQuery(competitionId: Ref<string>, id: Ref<string>) {
  return useQuery({
    queryKey: computed(() => categoryKeys.detail(competitionId.value, id.value)),
    queryFn: () => categoryService.getCategory(competitionId.value, id.value),
    enabled: computed(() => !!competitionId.value && !!id.value),
  })
}

/**
 * Create Category Mutation
 */
export function useCreateCategoryMutation(competitionId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateCategoryFormValues) =>
      categoryService.createCategory(competitionId.value, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: categoryKeys.list(competitionId.value) })
    },
  })
}

/**
 * Batch Seed Presets Mutation
 */
export function useSeedPresetsMutation(competitionId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (presets?: StandardClassPreset[]) =>
      categoryService.seedPresetCategories(competitionId.value, presets),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: categoryKeys.list(competitionId.value) })
    },
  })
}

/** Ubah satu kategori; daftar kategori dan bagan dimuat ulang. */
export function useUpdateCategoryMutation(competitionId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { id: string; payload: CreateCategoryFormValues }) =>
      categoryService.updateCategory(competitionId.value, vars.id, vars.payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: categoryKeys.list(competitionId.value) })
      queryClient.invalidateQueries({ queryKey: ['brackets'] })
    },
  })
}

/** Hapus satu kategori beserta slot bagannya. */
export function useDeleteCategoryMutation(competitionId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => categoryService.deleteCategory(competitionId.value, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: categoryKeys.list(competitionId.value) })
      queryClient.invalidateQueries({ queryKey: ['brackets'] })
    },
  })
}

/**
 * Reset all categories of a competition (switch PB IPSI <-> PERSILAT 2026)
 */
export function useResetCategoriesMutation(competitionId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => categoryService.resetCategories(competitionId.value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: categoryKeys.list(competitionId.value) })
      queryClient.invalidateQueries({ queryKey: ['brackets'] })
    },
  })
}

/**
 * Assign per-category / per-age-group ruleset (e.g. Usia Dini 2 x 1.5 min)
 */
export function useAssignRulesetsMutation(competitionId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (assignments: RulesetAssignment[]) =>
      categoryService.assignRulesets(competitionId.value, assignments),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: categoryKeys.list(competitionId.value) })
    },
  })
}

/** Kunci/buka draw bagan sebuah kategori; daftar kategori dimuat ulang agar status kunci dari server tampil. */
export function useSetBracketLockMutation(competitionId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { categoryId: string; locked: boolean }) =>
      categoryService.setBracketLock(competitionId.value, vars.categoryId, vars.locked),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: categoryKeys.list(competitionId.value) }),
  })
}
