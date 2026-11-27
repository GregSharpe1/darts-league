import { Link, useSearchParams } from 'react-router-dom'
import { StateNotice } from '../../components/StateNotice'
import { filterPlayerProfiles, usePlayerDirectory } from '../../lib/playerDirectory'
import type { PlayerProfile } from '../../lib/playerDirectory'
import './PlayerPage.css'

function profilePath(profile: PlayerProfile) {
  return 'season_id' in profile
    ? `/seasons/${profile.season_id}/players/${profile.player_id}`
    : `/players/tournaments/${profile.tournament_slug}/${profile.entry_id}`
}

export function PlayerDirectoryPage() {
  const [params, setParams] = useSearchParams()
  const search = params.get('q') ?? ''
  const query = usePlayerDirectory()
  const profiles = filterPlayerProfiles(query.data ?? [], search)

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
              ? search.trim() ? 'No players match your search.' : 'No player profiles are available yet.'
              : `${profiles.length} ${profiles.length === 1 ? 'profile' : 'profiles'}`}</p>
            <ul className="player-directory-results">{profiles.map((profile) =>
              <li key={profilePath(profile)}><Link className="text-link" to={profilePath(profile)}>
                <strong>{profile.name}</strong><span>{profile.competition}</span>
              </Link></li>)}
            </ul>
          </>}
    </section>
  </div>
}
