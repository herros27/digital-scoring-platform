import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { matchService } from './matchService'

export const matchKeys = {
  all: ['matches'] as const,
  lists: () => [...matchKeys.all, 'list'] as const,
  list: (compId: string, status?: string) =>
    [...matchKeys.lists(), compId, status || 'all'] as const,
  detail: (compId: string, id: string) =>
    [...matchKeys.all, 'detail', compId, id] as const,
}

/**
 * Fetch matches for a competition
 */
export function useMatchesQuery(compId: Ref<string>, statusFilter?: Ref<string>) {
  return useQuery({
    queryKey: computed(() => matchKeys.list(compId.value, statusFilter?.value)),
    queryFn: () => matchService.getMatches(compId.value, statusFilter?.value),
    enabled: computed(() => !!compId.value),
    staleTime: 1000 * 10,
  })
}

/**
 * Fetch single match detail
 */
export function useMatchQuery(compId: Ref<string>, matchId: Ref<string>) {
  return useQuery({
    queryKey: computed(() => matchKeys.detail(compId.value, matchId.value)),
    queryFn: () => matchService.getMatch(compId.value, matchId.value),
    enabled: computed(() => !!compId.value && !!matchId.value),
  })
}

/**
 * Create match mutation
 */
export function useCreateMatchMutation(compId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: {
      arena_id?: string
      bracket_slot_id?: string
      athlete_red_id: string
      athlete_blue_id: string
      current_round?: number
    }) => matchService.createMatch(compId.value, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: matchKeys.lists() })
    },
  })
}

/**
 * Load match to arena mutation
 */
export function useLoadArenaMutation(compId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ matchId, arenaId, rulesetPath }: { matchId: string; arenaId: string; rulesetPath?: string }) =>
      matchService.loadArena(compId.value, matchId, arenaId, rulesetPath),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: matchKeys.lists() })
    },
  })
}

/**
 * Start match mutation
 */
export function useStartMatchMutation(compId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (matchId: string) => matchService.startMatch(compId.value, matchId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: matchKeys.lists() })
    },
  })
}

/**
 * Pause match mutation
 */
export function usePauseMatchMutation(compId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (matchId: string) => matchService.pauseMatch(compId.value, matchId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: matchKeys.lists() })
    },
  })
}

/**
 * Resume match mutation
 */
export function useResumeMatchMutation(compId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (matchId: string) => matchService.resumeMatch(compId.value, matchId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: matchKeys.lists() })
    },
  })
}

/**
 * Finish match mutation
 */
export function useFinishMatchMutation(compId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (matchId: string) => matchService.finishMatch(compId.value, matchId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: matchKeys.lists() })
    },
  })
}
