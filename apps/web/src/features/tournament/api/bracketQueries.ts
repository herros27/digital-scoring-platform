import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { bracketService, type GenerateBracketPayload } from './bracketService'

export const bracketKeys = {
  all: ['brackets'] as const,
  trees: () => [...bracketKeys.all, 'tree'] as const,
  tree: (compId: string, weightClass: string, gender: string) =>
    [...bracketKeys.trees(), compId, weightClass, gender] as const,
  categoryTree: (compId: string, categoryId: string) =>
    [...bracketKeys.trees(), 'category', compId, categoryId] as const,
  participants: (compId: string) => [...bracketKeys.all, 'participants', compId] as const,
}

/**
 * Fetch bracket tree slots for a competition & category
 */
export function useBracketTreeQuery(
  compId: Ref<string>,
  weightClass: Ref<string>,
  gender: Ref<string>,
) {
  return useQuery({
    queryKey: computed(() => bracketKeys.tree(compId.value, weightClass.value, gender.value)),
    queryFn: () => bracketService.getBracketTree(compId.value, weightClass.value, gender.value),
    enabled: computed(() => !!compId.value && !!weightClass.value && !!gender.value),
    staleTime: 1000 * 15,
  })
}

/**
 * Fetch bracket tree isolated per category (golongan umur + kelas bobot)
 */
export function useCategoryBracketTreeQuery(compId: Ref<string>, categoryId: Ref<string>) {
  return useQuery({
    queryKey: computed(() => bracketKeys.categoryTree(compId.value, categoryId.value)),
    queryFn: () => bracketService.getBracketTreeByCategory(compId.value, categoryId.value),
    enabled: computed(() => !!compId.value && !!categoryId.value),
    staleTime: 1000 * 15,
  })
}

/**
 * Fetch verified participants for bracket generation
 */
export function useBracketParticipantsQuery(compId: Ref<string>) {
  return useQuery({
    queryKey: computed(() => bracketKeys.participants(compId.value)),
    queryFn: () => bracketService.getParticipants(compId.value),
    enabled: computed(() => !!compId.value),
    staleTime: 1000 * 30,
  })
}

/**
 * Generate bracket mutation
 */
export function useGenerateBracketMutation(compId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: GenerateBracketPayload) =>
      bracketService.generateBracket(compId.value, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bracketKeys.trees() })
      queryClient.invalidateQueries({ queryKey: ['matches'] })
    },
  })
}

/**
 * Reset bracket mutation: hapus bagan satu kategori (peserta tetap terdaftar)
 */
export function useResetBracketMutation(compId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (categoryId: string) => bracketService.resetBracket(compId.value, categoryId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bracketKeys.trees() })
      queryClient.invalidateQueries({ queryKey: ['matches'] })
    },
  })
}
