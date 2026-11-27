import { useQuery } from '@tanstack/react-query'
import { request } from './api'

export type PlayerAccount = {
  email: string
  identity_id?: number
  display_name?: string
  nickname?: string
  claim_status?: 'pending' | 'approved' | 'rejected'
}

export type ClaimProfile = {
  identity_id: number
  name: string
  display_name?: string
  competition: string
}

export const playerAccessKey = ['player-access']

export function usePlayerAccess() {
  return useQuery({
    queryKey: playerAccessKey,
    queryFn: ({ signal }) => request<{ enabled: boolean; account?: PlayerAccount }>('/api/player-access', { signal, cache: 'no-store' }),
    retry: false,
    gcTime: 0,
  })
}
