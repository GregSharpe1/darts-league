import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { StateNotice } from '../../components/StateNotice'
import { request } from '../../lib/api'
import './PlayerPage.css'

type Profile = {
  name: string
  competition: string
  display_name?: string
} & (
  | { season_id: number; player_id: number }
  | { tournament_slug: string; entry_id: number }
)

function profilePath(profile: Profile) {
  return 'season_id' in profile
    ? `/seasons/${profile.season_id}/players/${profile.player_id}`
    : `/players/tournaments/${profile.tournament_slug}/${profile.entry_id}`
}

function normalizeName(name: string) {
  return name.trim().replace(/\s+/g, ' ').toLocaleLowerCase('en-GB')
}

export function PlayerDirectoryPage() {
  const [params, setParams] = useSearchParams()
  const search = params.get('q') ?? ''
  const query = useQuery({
    queryKey: ['public-player-directory'],
    queryFn: ({ signal }) => request<Profile[]>('/api/players', { signal, cache: 'no-store' }),
  })
  const needle = normalizeName(search)
  const profiles = (query.data ?? []).filter(({ name, display_name }) =>
    normalizeName(name).includes(needle) || (display_name && normalizeName(display_name).includes(needle)))
    .sort((a, b) => a.name.localeCompare(b.name, 'en-GB') || a.competition.localeCompare(b.competition, 'en-GB'))

  return <div className="player-page">
    <section className="page-intro">
      <h1>Player stats</h1>
      <p>Find players in league seasons and published tournament brackets.</p>
    </section>
    <section className="content-panel" aria-label="Player directory">
      <div className="field player-search">
        <label htmlFor="player-search">Search by player name</label>
        <input id="player-search" type="search" value={search} onChange={(event) => {
          const value = event.target.value
          setParams(value ? { q: value } : {}, { replace: true })
        }} />
      </div>
      {query.isPending ? <p role="status">Loading players...</p> :
        query.isError ? <><StateNotice tone="error" message="Player directory could not be loaded." />
          <button type="button" onClick={() => { void query.refetch() }}>Try again</button></> :
          <>
            <p role="status">{profiles.length === 0
              ? needle ? 'No players match your search.' : 'No player profiles are available yet.'
              : `${profiles.length} ${profiles.length === 1 ? 'profile' : 'profiles'}`}</p>
            <ul className="player-directory-results">{profiles.map((profile) =>
              <li key={profilePath(profile)}><Link to={profilePath(profile)}>
                <strong>{profile.name}</strong><span>{profile.competition}</span>
              </Link></li>)}
            </ul>
          </>}
    </section>
  </div>
}
