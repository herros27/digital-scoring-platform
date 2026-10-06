import { useQuery } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { reportService } from './reportService'

export const reportKeys = {
  all: ['reports'] as const,
  summary: (compId: string) => [...reportKeys.all, 'summary', compId] as const,
  medals: (compId: string) => [...reportKeys.all, 'medals', compId] as const,
  export: (compId: string) => [...reportKeys.all, 'export', compId] as const,
}

/**
 * Fetch competition summary report
 */
export function useCompetitionSummaryQuery(compId: Ref<string>) {
  return useQuery({
    queryKey: computed(() => reportKeys.summary(compId.value)),
    queryFn: () => reportService.getSummary(compId.value),
    enabled: computed(() => !!compId.value),
    staleTime: 1000 * 10,
    refetchInterval: 1000 * 15,
  })
}

/**
 * Fetch medal standings
 */
export function useMedalStandingsQuery(compId: Ref<string>) {
  return useQuery({
    queryKey: computed(() => reportKeys.medals(compId.value)),
    queryFn: () => reportService.getMedalStandings(compId.value),
    enabled: computed(() => !!compId.value),
    staleTime: 1000 * 10,
    refetchInterval: 1000 * 15,
  })
}

/**
 * Fetch full tournament export data
 */
export function useCompetitionExportQuery(
  compId: Ref<string>,
  enabled: Ref<boolean> = computed(() => false),
) {
  return useQuery({
    queryKey: computed(() => reportKeys.export(compId.value)),
    queryFn: () => reportService.getExport(compId.value),
    enabled: computed(() => !!compId.value && enabled.value),
    staleTime: 1000 * 60,
  })
}
