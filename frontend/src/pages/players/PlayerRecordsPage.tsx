import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { StateNotice } from '../../components/StateNotice'
import { request } from '../../lib/api'
import { playerProfilePath } from '../../lib/playerDirectory'
import type { PlayerProfile } from '../../lib/playerDirectory'
import './PlayerPage.css'

const metrics = [
  { key: 'total_180', title: 'Most 180s', empty: 'No 180s have been recorded.' },
  { key: 'highest_finish', title: 'Highest checkout', empty: 'No checkouts have been recorded.' },
  { key: 'bullseyes', title: 'Recorded bullseyes', empty: 'No inner bulls have been recorded.' },
  { key: 'match_average', title: 'Highest match average', empty: 'No match averages have been recorded.' },
  { key: 'best_leg', title: 'Fewest darts to win a leg', empty: 'No complete winning legs have been recorded.' },
  { key: 'wins', title: 'Most match wins', empty: 'No match wins have been recorded.' },
] as const

type Metric = typeof metrics[number]['key']

type PlayerRecord = {
  profile: PlayerProfile
  rank: number
  value: number
  known_matches: number
  played: number
}

function displayValue(metric: Metric, value: number) {
  if (metric === 'match_average') return value.toFixed(2)
  if (metric === 'best_leg') return `${value} darts`
  return value.toLocaleString('en-GB')
}

export function PlayerRecordsPage() {
  const query = useQuery({
    queryKey: ['player-records'],
    queryFn: ({ signal }) => request<Record<Metric, PlayerRecord[]>>('/api/players/records', { signal, cache: 'no-store' }),
    retry: false,
  })

  return <div className="player-page">
    <header className="player-page-heading"><Link className="text-link" to="/players">Player stats</Link></header>
    <section className="page-intro">
      <h1>Overall records</h1>
      <p>Across revealed league results and published tournament brackets. Darts stats depend on the available Autodarts data. Bullseyes count inner bulls only.</p>
    </section>
    {query.isPending ? <p className="content-panel" role="status">Loading player records...</p> :
      query.isError ? <div className="content-panel">
        <StateNotice tone="error" message="Player records could not be loaded." />
        <div className="toolbar-actions"><button type="button" onClick={() => { void query.refetch() }}>Try again</button></div>
      </div> :
        <div className="player-records-grid">{metrics.map(metric => {
          const rows = query.data[metric.key]
          return <section className="content-panel" key={metric.key} aria-labelledby={`player-record-${metric.key}`}>
            <h2 id={`player-record-${metric.key}`}>{metric.title}</h2>
            {rows.length === 0 ? <p className="player-note">{metric.empty}</p> :
              <ol className="player-record-rows">{rows.map(row => <li key={row.profile.identity_id}>
                <span className="position-badge" aria-label={`Position ${row.rank}`}>{row.rank}</span>
                <div>
                  <Link className="text-link" to={playerProfilePath(row.profile)}>{row.profile.name}</Link>
                  <span className="player-note">{row.known_matches} of {row.played} matches with data</span>
                </div>
                <strong>{displayValue(metric.key, row.value)}</strong>
              </li>)}</ol>}
          </section>
        })}</div>}
  </div>
}
