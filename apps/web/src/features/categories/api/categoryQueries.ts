import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { categoryService } from './categoryService'
import type { CreateCategoryFormValues, StandardClassPreset } from '../types/category.schema'

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
