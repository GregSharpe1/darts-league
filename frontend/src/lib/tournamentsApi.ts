import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError, request } from './api'

export type TournamentMatchFormat = {
  game_variant: string
  legs_to_win: number
}

export type Tournament = TournamentMatchFormat & {
  slug: string
  name: string
  starts_on: string
  registration_open: boolean
  bracket?: BracketMatch[][]
}

export type TournamentSummary = Pick<Tournament, 'slug' | 'name' | 'starts_on' | 'registration_open'> & {
  started: boolean
}

export type BracketSlot = { player?: string; winner_of?: number; bye?: true }
export type BracketMatch = { number: number; slots: [BracketSlot, BracketSlot] }

export type TournamentEntry = {
  id: number
  display_name: string
  nickname?: string
  registered_at: string
}

const publicListPath = '/api/tournaments'
const publicPath = (slug: string) => `${publicListPath}/${encodeURIComponent(slug)}`
const adminPath = '/api/admin/tournaments'
const adminKey = ['admin', 'tournaments']
const publicListKey = ['tournaments']
const entryKey = (slug: string) => [...adminKey, 'entries', slug]
const eventKey = (slug: string) => ['tournament', slug]

export function useTournaments() {
  return useQuery({
    queryKey: publicListKey,
    queryFn: () => request<TournamentSummary[]>(publicListPath),
    retry: false,
  })
}

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
      if (error instanceof ApiError && (error.code === 'registration_closed' || error.code === 'tournament_not_found')) {
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
    mutationFn: (event: Pick<Tournament, 'slug' | 'name' | 'starts_on'> & TournamentMatchFormat) =>
      request<Tournament>(adminPath, { method: 'POST', body: JSON.stringify(event) }),
    onSuccess: async (event) => {
      client.setQueryData<Tournament[]>(adminKey, (events) => [...(events ?? []), event])
      await Promise.all([
        client.invalidateQueries({ queryKey: adminKey, exact: true }),
        client.invalidateQueries({ queryKey: publicListKey }),
      ])
    },
  })
}

export function useUpdateTournamentFormat(slug: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (format: TournamentMatchFormat) =>
      request<Tournament>(`${adminPath}/${encodeURIComponent(slug)}/format`, {
        method: 'PUT',
        body: JSON.stringify(format),
      }),
    onSuccess: (event) => {
      client.setQueryData(eventKey(slug), event)
      client.setQueryData<Tournament[]>(adminKey, (events) => events?.map((item) => item.slug === slug ? event : item))
    },
    onSettled: () => client.invalidateQueries({ queryKey: adminKey, exact: true }),
  })
}

export function useStartTournament(slug: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: () => request<Tournament>(`${adminPath}/${encodeURIComponent(slug)}/start`, { method: 'POST' }),
    onSuccess: async (event) => {
      client.setQueryData(eventKey(slug), event)
      client.setQueryData<Tournament[]>(adminKey, (events) => events?.map((item) => item.slug === slug ? event : item))
      await client.invalidateQueries({ queryKey: publicListKey })
    },
    onSettled: () => client.invalidateQueries({ queryKey: adminKey, exact: true }),
  })
}

export function useDeleteTournament(slug: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: () => request<void>(`${adminPath}/${encodeURIComponent(slug)}`, { method: 'DELETE' }),
    onSuccess: async () => {
      client.setQueryData<Tournament[]>(adminKey, (events) => events?.filter((event) => event.slug !== slug))
      client.setQueryData<TournamentSummary[]>(publicListKey, (events) => events?.filter((event) => event.slug !== slug))
      client.removeQueries({ queryKey: entryKey(slug) })
      await Promise.all([
        client.invalidateQueries({ queryKey: publicListKey }),
        client.invalidateQueries({ queryKey: eventKey(slug) }),
      ])
    },
    onSettled: () => client.invalidateQueries({ queryKey: adminKey, exact: true }),
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
    onSuccess: async (event) => {
      client.setQueryData(eventKey(slug), event)
      client.setQueryData<Tournament[]>(adminKey, (events) => events?.map((item) => item.slug === slug ? event : item))
      await client.invalidateQueries({ queryKey: publicListKey })
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
