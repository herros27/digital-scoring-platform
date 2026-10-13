import { useQuery } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { publicBracketService } from './publicBracketService'

/** Bagan berubah saat pertandingan selesai; penonton cukup menerima pembaruan tiap 15 dtk. */
const REFRESH_MS = 15_000

export const publicBracketKeys = {
  all: ['public-brackets'] as const,
  competitions: () => [...publicBracketKeys.all, 'competitions'] as const,
  list: (compId: string) => [...publicBracketKeys.all, 'list', compId] as const,
  detail: (compId: string, categoryId: string) =>
    [...publicBracketKeys.all, 'detail', compId, categoryId] as const,
}

export function usePublicCompetitionsQuery() {
  return useQuery({
    queryKey: publicBracketKeys.competitions(),
    queryFn: () => publicBracketService.getCompetitions(),
    refetchInterval: REFRESH_MS,
    retry: 1,
  })
}

export function usePublicBracketListQuery(compId: Ref<string>) {
  return useQuery({
    queryKey: computed(() => publicBracketKeys.list(compId.value)),
    queryFn: () => publicBracketService.getBrackets(compId.value),
    enabled: computed(() => !!compId.value),
    refetchInterval: REFRESH_MS,
    retry: 1,
  })
}

export function usePublicBracketQuery(compId: Ref<string>, categoryId: Ref<string>) {
  return useQuery({
    queryKey: computed(() => publicBracketKeys.detail(compId.value, categoryId.value)),
    queryFn: () => publicBracketService.getBracket(compId.value, categoryId.value),
    enabled: computed(() => !!compId.value && !!categoryId.value),
    refetchInterval: REFRESH_MS,
    retry: false, // 404 (belum aktif / tidak ada) tidak perlu diulang
  })
}
