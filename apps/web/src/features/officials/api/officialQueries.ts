import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { officialService } from './officialService'

export const officialKeys = {
  all: ['officials'] as const,
  list: () => [...officialKeys.all, 'list'] as const,
}

export function useOfficialsQuery() {
  return useQuery({
    queryKey: officialKeys.list(),
    queryFn: () => officialService.list(),
    staleTime: 1000 * 15,
  })
}

export function useCreateOfficialMutation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: officialService.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: officialKeys.all }),
  })
}

export function useUpdateOfficialMutation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { id: string; name: string; role: string }) =>
      officialService.update(v.id, { name: v.name, role: v.role }),
    onSuccess: () => qc.invalidateQueries({ queryKey: officialKeys.all }),
  })
}

export function useDeleteOfficialMutation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => officialService.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: officialKeys.all }),
  })
}

export function useResetPasswordMutation() {
  return useMutation({
    mutationFn: (v: { id: string; password: string }) =>
      officialService.resetPassword(v.id, v.password),
  })
}

export function useResetContingentPasswordMutation() {
  return useMutation({
    mutationFn: (v: { id: string; password: string }) =>
      officialService.resetContingentPassword(v.id, v.password),
  })
}

export function useChangeOwnPasswordMutation() {
  return useMutation({
    mutationFn: (v: { current: string; next: string }) =>
      officialService.changeOwnPassword(v.current, v.next),
  })
}
