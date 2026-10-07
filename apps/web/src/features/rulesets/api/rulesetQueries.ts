import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { rulesetService } from './rulesetService'
import type { RulesetPayload } from '../types'

export const rulesetKeys = {
  all: ['rulesets'] as const,
  list: () => [...rulesetKeys.all, 'list'] as const,
}

export function useRulesetsQuery() {
  return useQuery({
    queryKey: rulesetKeys.list(),
    queryFn: () => rulesetService.list(),
    staleTime: 1000 * 15,
  })
}

export function useSaveRulesetMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, payload }: { id?: string; payload: RulesetPayload }) =>
      id ? rulesetService.update(id, payload) : rulesetService.create(payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: rulesetKeys.all }),
  })
}

export function useDeleteRulesetMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => rulesetService.remove(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: rulesetKeys.all }),
  })
}
