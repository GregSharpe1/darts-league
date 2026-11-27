import { useQuery } from '@tanstack/react-query'
import { request } from './api'

export type PlayerProfile = {
  identity_id: number
  name: string
  competition: string
  display_name?: string
  first_name?: string
} & (
  | { season_id: number; player_id: number }
  | { tournament_slug: string; entry_id: number }
)

export function playerProfilePath(profile: PlayerProfile) {
  return 'season_id' in profile
    ? `/seasons/${profile.season_id}/players/${profile.player_id}`
    : `/players/tournaments/${profile.tournament_slug}/${profile.entry_id}`
}

export function usePlayerDirectory() {
  return useQuery({
    queryKey: ['public-player-directory'],
    queryFn: ({ signal }) => request<PlayerProfile[]>('/api/players', { signal, cache: 'no-store' }),
  })
}

function normalizeName(name: string) {
  return name.trim().replace(/\s+/g, ' ').toLocaleLowerCase('en-GB')
}

export function filterPlayerProfiles(profiles: PlayerProfile[], search: string) {
  const needle = normalizeName(search)
  return profiles.filter(({ name, display_name, first_name }) =>
    [name, display_name, first_name].some(value => value && normalizeName(value).includes(needle)))
    .sort((a, b) => a.name.localeCompare(b.name, 'en-GB') || a.competition.localeCompare(b.competition, 'en-GB'))
}
