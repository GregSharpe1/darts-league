import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError, request } from './api'

export type Tournament = {
  slug: string
  name: string
  starts_on: string
  registration_open: boolean
}

export type TournamentEntry = {
  id: number
  display_name: string
  nickname?: string
  registered_at: string
}

const publicPath = (slug: string) => `/api/tournaments/${encodeURIComponent(slug)}`
const adminPath = '/api/admin/tournaments'
const adminKey = ['admin', 'tournaments']
const entryKey = (slug: string) => [...adminKey, 'entries', slug]
const eventKey = (slug: string) => ['tournament', slug]

export function useTournament(slug: string) {
  return useQuery({
    queryKey: eventKey(slug),
    queryFn: () => request<Tournament>(publicPath(slug)),
    retry: false,
  })
}

export function useTournamentSignup(slug: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (entry: { display_name: string; nickname: string }) =>
      request<{ preferred_name: string }>(`${publicPath(slug)}/entries`, { method: 'POST', body: JSON.stringify(entry) }),
    onError: async (error) => {
      if (error instanceof ApiError && error.code === 'registration_closed') {
        await client.invalidateQueries({ queryKey: eventKey(slug) })
      }
    },
    onSuccess: () => client.invalidateQueries({ queryKey: entryKey(slug) }),
  })
}

export function useAdminTournaments() {
  return useQuery({
    queryKey: adminKey,
    queryFn: () => request<Tournament[]>(adminPath),
    retry: false,
  })
}

export function useCreateTournament() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (event: Omit<Tournament, 'registration_open'>) =>
      request<Tournament>(adminPath, { method: 'POST', body: JSON.stringify(event) }),
    onSuccess: async (event) => {
      client.setQueryData<Tournament[]>(adminKey, (events) => [...(events ?? []), event])
      await client.invalidateQueries({ queryKey: adminKey, exact: true })
    },
  })
}

export function useSetTournamentRegistration(slug: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (open: boolean) =>
      request<Tournament>(`${adminPath}/${encodeURIComponent(slug)}/registration`, {
        method: 'PUT',
        body: JSON.stringify({ registration_open: open }),
      }),
    onSuccess: (event) => {
      client.setQueryData(eventKey(slug), event)
      client.setQueryData<Tournament[]>(adminKey, (events) => events?.map((item) => item.slug === slug ? event : item))
    },
    onSettled: () => client.invalidateQueries({ queryKey: adminKey, exact: true }),
  })
}

export function useTournamentEntries(slug: string) {
  return useQuery({
    queryKey: entryKey(slug),
    queryFn: () => request<TournamentEntry[]>(`${adminPath}/${encodeURIComponent(slug)}/entries`),
    retry: false,
  })
}

export function useRemoveTournamentEntry(slug: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (id: number) =>
      request<void>(`${adminPath}/${encodeURIComponent(slug)}/entries/${id}`, { method: 'DELETE' }),
    onSettled: () => client.invalidateQueries({ queryKey: entryKey(slug) }),
  })
}
