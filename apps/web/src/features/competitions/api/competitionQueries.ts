import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { competitionService } from './competitionService'
import type {
  CreateCompetitionFormValues,
  UpdateCompetitionFormValues,
  CompetitionFilterValues,
} from '../types/competition.schema'

/**
 * Query Key Factory for Competitions domain
 */
export const competitionKeys = {
  all: ['competitions'] as const,
  lists: () => [...competitionKeys.all, 'list'] as const,
  list: (filters: CompetitionFilterValues) => [...competitionKeys.lists(), filters] as const,
  stats: () => [...competitionKeys.all, 'stats'] as const,
  details: () => [...competitionKeys.all, 'detail'] as const,
  detail: (id: string) => [...competitionKeys.details(), id] as const,
}

/**
 * Fetch competitions list with TanStack Query
 */
export function useCompetitionsQuery(filters: Ref<CompetitionFilterValues>) {
  return useQuery({
    queryKey: computed(() => competitionKeys.list(filters.value)),
    queryFn: () => competitionService.getCompetitions(filters.value),
    staleTime: 1000 * 30, // 30s
    placeholderData: keepPreviousData, // pindah halaman/filter tanpa daftar berkedip kosong
  })
}

/**
 * Hitungan kejuaraan per status (global). Ikut di-invalidate bersama daftar karena berada di bawah competitionKeys.all.
 */
export function useCompetitionStatsQuery() {
  return useQuery({
    queryKey: competitionKeys.stats(),
    queryFn: () => competitionService.getStats(),
    staleTime: 1000 * 30,
  })
}

/**
 * Fetch competition detail
 */
export function useCompetitionQuery(id: Ref<string>) {
  return useQuery({
    queryKey: computed(() => competitionKeys.detail(id.value)),
    queryFn: () => competitionService.getCompetition(id.value),
    enabled: computed(() => !!id.value),
  })
}

/**
 * Create Competition Mutation
 */
export function useCreateCompetitionMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateCompetitionFormValues) =>
      competitionService.createCompetition(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: competitionKeys.lists() })
      queryClient.invalidateQueries({ queryKey: competitionKeys.stats() })
    },
  })
}

/**
 * Update Competition Mutation
 */
export function useUpdateCompetitionMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpdateCompetitionFormValues }) =>
      competitionService.updateCompetition(id, payload),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: competitionKeys.lists() })
      queryClient.invalidateQueries({ queryKey: competitionKeys.stats() })
      queryClient.invalidateQueries({ queryKey: competitionKeys.detail(variables.id) })
    },
  })
}

/**
 * Activate Competition Mutation
 */
export function useActivateCompetitionMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => competitionService.activateCompetition(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: competitionKeys.lists() })
      queryClient.invalidateQueries({ queryKey: competitionKeys.stats() })
      queryClient.invalidateQueries({ queryKey: competitionKeys.detail(id) })
    },
  })
}

/**
 * Pause Competition Mutation
 */
export function usePauseCompetitionMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => competitionService.pauseCompetition(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: competitionKeys.lists() })
      queryClient.invalidateQueries({ queryKey: competitionKeys.stats() })
      queryClient.invalidateQueries({ queryKey: competitionKeys.detail(id) })
    },
  })
}

/**
 * Finish Competition Mutation
 */
export function useFinishCompetitionMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => competitionService.finishCompetition(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: competitionKeys.lists() })
      queryClient.invalidateQueries({ queryKey: competitionKeys.stats() })
      queryClient.invalidateQueries({ queryKey: competitionKeys.detail(id) })
    },
  })
}

/**
 * Cancel Competition Mutation
 */
export function useCancelCompetitionMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => competitionService.cancelCompetition(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: competitionKeys.lists() })
      queryClient.invalidateQueries({ queryKey: competitionKeys.stats() })
      queryClient.invalidateQueries({ queryKey: competitionKeys.detail(id) })
    },
  })
}
