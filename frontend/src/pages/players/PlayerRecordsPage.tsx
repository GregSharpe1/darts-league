import { useState } from 'react'
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

type MetricDefinition = typeof metrics[number]
type Metric = MetricDefinition['key']

const previewSize = 5

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

function RecordSection({ metric, rows }: { metric: MetricDefinition; rows: PlayerRecord[] }) {
  const [expanded, setExpanded] = useState(false)
  const headingId = `player-record-${metric.key}`
  const listId = `${headingId}-rows`
  const visibleRows = expanded ? rows : rows.slice(0, previewSize)

  return <section className="content-panel" aria-labelledby={headingId}>
    <div className="player-record-heading">
      <h2 id={headingId}>{metric.title}</h2>
      {rows.length > previewSize && (
        <div className="toolbar-actions">
          <button type="button" className="button-link secondary-button" aria-expanded={expanded} aria-controls={listId}
            onClick={() => setExpanded(!expanded)}>
            {expanded ? `Show top ${previewSize}` : `Show all ${rows.length}`}
          </button>
        </div>
      )}
    </div>
    {rows.length === 0 ? <p className="player-note">{metric.empty}</p> : (
      <ol id={listId} className="player-record-rows">{visibleRows.map(row => <li key={row.profile.identity_id}>
        <span className="position-badge" aria-label={`Position ${row.rank}`}>{row.rank}</span>
        <div>
          <Link className="text-link" to={playerProfilePath(row.profile)}>{row.profile.name}</Link>
          <span className="player-note">{row.known_matches} of {row.played} matches with data</span>
        </div>
        <strong>{displayValue(metric.key, row.value)}</strong>
      </li>)}</ol>
    )}
  </section>
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
        <div className="player-records-grid">
          {metrics.map(metric => <RecordSection key={metric.key} metric={metric} rows={query.data[metric.key]} />)}
        </div>}
  </div>
}
