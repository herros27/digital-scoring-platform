import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { athleteService } from './athleteService'
import type { CreateAthleteFormValues, RegisterParticipantPayload } from '../types/athlete.schema'

export const athleteKeys = {
  allAthletes: ['athletes'] as const,
  compAthletes: (compId: string) => ['competitions', compId, 'athletes'] as const,
  contingents: ['contingents'] as const,
  participants: (compId: string, status?: string) =>
    ['competitions', compId, 'participants', status || 'all'] as const,
}

/**
 * Fetch all registered athletes in system
 */
export function useAthletesQuery(
  params?: Ref<{ gender?: string; page?: number; limit?: number } | undefined>,
) {
  return useQuery({
    queryKey: computed(() => [
      ...athleteKeys.allAthletes,
      params?.value?.gender || 'all',
      params?.value?.page || 1,
    ]),
    queryFn: () => athleteService.getAthletes(params?.value),
    staleTime: 1000 * 30,
  })
}

/**
 * Fetch athletes registered to a specific tournament
 */
export function useCompetitionAthletesQuery(compId: Ref<string>) {
  return useQuery({
    queryKey: computed(() => athleteKeys.compAthletes(compId.value)),
    queryFn: () => athleteService.getAthletesByCompetition(compId.value),
    enabled: computed(() => !!compId.value),
    staleTime: 1000 * 30,
  })
}

/**
 * Fetch participants queued for verification in a competition
 */
export function useParticipantsQuery(compId: Ref<string>, statusFilter?: Ref<string>) {
  return useQuery({
    queryKey: computed(() => athleteKeys.participants(compId.value, statusFilter?.value)),
    queryFn: () => athleteService.getParticipants(compId.value, statusFilter?.value),
    enabled: computed(() => !!compId.value),
    staleTime: 1000 * 15,
  })
}

/**
 * Fetch list of contingents
 */
export function useContingentsQuery() {
  return useQuery({
    queryKey: athleteKeys.contingents,
    queryFn: () => athleteService.getContingents(),
    staleTime: 1000 * 60,
  })
}

/**
 * Create athlete mutation
 */
export function useCreateAthleteMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateAthleteFormValues) => athleteService.createAthlete(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: athleteKeys.allAthletes })
    },
  })
}

/**
 * Register participant to tournament mutation
 */
export function useRegisterToCompetitionMutation(compId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: RegisterParticipantPayload) =>
      athleteService.registerToCompetition(compId.value, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: athleteKeys.compAthletes(compId.value) })
      queryClient.invalidateQueries({ queryKey: ['competitions', compId.value, 'participants'] })
    },
  })
}

/**
 * Verify / Reject participant mutation
 */
export function useVerifyParticipantMutation(compId: Ref<string>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      participantId,
      status,
      actual_weight,
      docs_verified,
    }: {
      participantId: string
      status: 'verified' | 'rejected'
      actual_weight?: number
      docs_verified?: Record<string, boolean>
    }) =>
      athleteService.verifyParticipant(compId.value, participantId, status, {
        actual_weight,
        docs_verified,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['competitions', compId.value, 'participants'] })
      queryClient.invalidateQueries({ queryKey: athleteKeys.compAthletes(compId.value) })
    },
  })
}
