import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { arenaService } from './arenaService'

export const arenaKeys = {
  all: ['arenas'] as const,
  lists: () => [...arenaKeys.all, 'list'] as const,
  list: (compId: string) => [...arenaKeys.lists(), compId] as const,
}

/**
 * Fetch arenas for a competition
 */
export function useArenasQuery(compId: Ref<string>) {
  return useQuery({
    queryKey: computed(() => arenaKeys.list(compId.value)),
    queryFn: () => arenaService.getArenas(compId.value),
    enabled: computed(() => !!compId.value),
    staleTime: 1000 * 15,
  })
}

export function useCreateArenaMutation(defaultCompId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: string | { name: string; compId?: string }) => {
      const name = typeof payload === 'string' ? payload : payload.name
      const target =
        typeof payload === 'object' && payload.compId ? payload.compId : defaultCompId.value
      return arenaService.createArena(target, name)
    },
    onSuccess: (_data, payload) => {
      const target =
        typeof payload === 'object' && payload.compId ? payload.compId : defaultCompId.value
      queryClient.invalidateQueries({ queryKey: arenaKeys.list(target) })
    },
  })
}

/**
 * Update Arena mutation
 */
export function useUpdateArenaMutation(compId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      arenaService.updateArena(compId.value, id, name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: arenaKeys.list(compId.value) })
    },
  })
}

/**
 * Delete Arena mutation
 */
export function useDeleteArenaMutation(compId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => arenaService.deleteArena(compId.value, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: arenaKeys.list(compId.value) })
    },
  })
}

/**
 * Seed Standard Arenas mutation
 */
export function useSeedArenasMutation(compId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => arenaService.seedStandardArenas(compId.value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: arenaKeys.list(compId.value) })
    },
  })
}
