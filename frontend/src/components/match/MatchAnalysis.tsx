import { useId, useState } from 'react'
import { Heatmap } from './Heatmap'
import { MatchSummary } from './MatchSummary'
import { MatchStatus } from './MatchStatus'
import { segmentLabel } from './geometry'
import type { MatchAnalysisProps, MatchData } from './types'

export function MatchAnalysis(props: MatchAnalysisProps) {
  switch (props.status) {
    case 'loading': return <MatchStatus status="loading" />
    case 'error': return <MatchStatus {...props} />
    case 'ready': return <Analysis match={props.match} />
    default: return assertNever(props)
  }
}

function assertNever(value: never): never { throw new TypeError(`Unexpected status: ${String(value)}`) }

function Analysis({ match }: { readonly match: MatchData }) {
  const legId = useId()
  const [playerId, setPlayerId] = useState<string | null>(null)
  const [legNumber, setLegNumber] = useState('all')
  const player = match.players.find(p => p.id === playerId) ?? match.players.find(p => p.legs_won === 3) ?? match.players[0]
  const legs = [...(match.detail?.legs ?? [])].sort((a, b) => a.number - b.number)
  const selectedLeg = legs.some(leg => String(leg.number) === legNumber) ? legNumber : 'all'
  const visits = legs.filter(leg => selectedLeg === 'all' || String(leg.number) === selectedLeg)
    .flatMap(leg => [...leg.visits].sort((a, b) => a.number - b.number)
      .filter(visit => visit.player_id === player.id).map(visit => ({ leg: leg.number, visit })))
  const throws = visits.flatMap(({ visit }) => visit.throws)
  const segments = new Map<string, number>()
  for (const dart of throws) {
    const label = segmentLabel(dart.segment)
    segments.set(label, (segments.get(label) ?? 0) + 1)
  }
  const scope = `${player.label} / ${selectedLeg === 'all' ? 'Whole match' : `Leg ${selectedLeg}`}`

  return <div className="ma ma-analysis">
    <MatchSummary match={match} />
    {!match.detail ? <section className="ma-panel"><h2>Throw detail unavailable</h2>
      <p className="ma-muted">Only the recorded match summary is available.</p></section> : <>
      <section className="ma-panel" aria-label="Throw analysis">
        <header className="ma-section-heading"><div><span className="ma-eyebrow">Recorded detail</span><h2>Dart placement</h2></div>
          <div className="ma-leg-filter"><label htmlFor={legId}>Leg</label><select id={legId} value={selectedLeg} onChange={event => setLegNumber(event.target.value)}>
            <option value="all">All recorded legs</option>{legs.map(leg => <option key={leg.number} value={leg.number}>Leg {leg.number}{leg.completed ? '' : ' (unfinished)'}</option>)}
          </select></div></header>
        <div className="ma-player-switch" aria-label="Player filter">{match.players.map(p =>
          <button type="button" key={p.id} aria-pressed={player.id === p.id} onClick={() => setPlayerId(p.id)}>{p.label}</button>)}</div>
        <div className="ma-analysis-grid"><Heatmap throws={throws} label={scope} synthetic={match.synthetic} />
          <div className="ma-analysis-stats">
            <div className="ma-scope" aria-live="polite"><h3>{scope}</h3><strong>{throws.length} <small>recorded throws</small></strong></div>
            <dl className="ma-stats">
              <div><dt>First-nine average (source)</dt><dd>{selectedLeg === 'all' ? player.stats?.first_nine_average?.toFixed(1) ?? 'Not available' : 'Not available'}</dd></div>
              <div><dt>180s (source)</dt><dd>{selectedLeg === 'all' ? player.stats?.total_180 ?? 'Not available' : 'Not available'}</dd></div>
              <div><dt>Highest finish (source)</dt><dd>{selectedLeg === 'all' ? player.stats?.highest_finish ?? 'Not available' : 'Not available'}</dd></div>
              <div><dt>Recorded visits</dt><dd>{visits.length}</dd></div>
              <div><dt>Legs with selected player data</dt><dd>{new Set(visits.map(item => item.leg)).size}</dd></div>
              <div><dt>Recorded points (busts score 0)</dt><dd>{visits.reduce((sum, { visit }) => sum + (visit.bust ? 0 : visit.start_remaining - visit.end_remaining), 0)}</dd></div>
              <div><dt>Recorded checkout visits</dt><dd>{visits.filter(({ visit }) => !visit.bust && visit.end_remaining === 0).length}</dd></div>
            </dl>
            <h3>Segment distribution</h3>
            <ul className="ma-segments" aria-label="Segment totals">{[...segments].map(([label, count]) => <li key={label}>
              <span>{label}: {count}</span><meter min="0" max={throws.length} value={count} aria-label={`${label} count`} />
            </li>)}</ul>
            {!throws.length && <p>No recorded throws in this selection.</p>}
            <p className="ma-note">{match.detail.coverage === 'partial' ? 'Partial detail: these counts are not full match totals.' : 'Counts cover recorded throws in this selection.'} Checkout attempts are not inferred from segments.</p>
            <p className="ma-note">Entry provenance: {(['automatic', 'manual', 'unknown'] as const).map(p => `${p} ${throws.filter(d => d.entry_type === p).length}`).join(' / ')}.<br />
              Position provenance: {(['automatic', 'manual', 'unknown'] as const).map(p => `${p} ${throws.filter(d => d.position?.provenance === p).length}`).join(' / ')}.</p>
          </div>
        </div>
      </section>
      <section className="ma-panel"><h2>Visits &amp; checkout routes</h2>
        <p className="ma-scroll-hint">Scroll horizontally for scores and routes.</p>
        <div className="ma-table-wrap" role="region" aria-label="Recorded visits and checkout routes" tabIndex={0}>
          <table><caption>{scope} - {match.detail.coverage} detail</caption>
            <thead><tr><th scope="col">Leg / visit</th><th scope="col">Start</th><th scope="col">Scored</th><th scope="col">Remaining</th><th scope="col">Route</th></tr></thead>
            <tbody>{visits.map(({ leg, visit }) => <tr key={`${leg}-${visit.number}`}>
              <th scope="row">{leg} / {visit.number}</th><td>{visit.start_remaining}</td>
              <td>{visit.bust ? '0 (bust)' : visit.start_remaining - visit.end_remaining}</td><td>{visit.end_remaining}</td>
              <td>{visit.throws.map(dart => segmentLabel(dart.segment)).join(' / ')}{!visit.bust && visit.end_remaining === 0 && <b> - Checkout</b>}</td>
            </tr>)}</tbody>
          </table>
        </div>
        <details className="ma-raw"><summary>Recorded coordinates &amp; provenance (text)</summary>
          <ul>{visits.flatMap(({ leg, visit }) => visit.throws.map(dart => <li key={`${leg}-${visit.number}-${dart.number}`}>
            Leg {leg}, visit {visit.number}, dart {dart.number}: {segmentLabel(dart.segment)}; entry {dart.entry_type}; {dart.position ?
              `x=${dart.position.x}, y=${dart.position.y}; units=${dart.position.units ?? 'unknown'}, origin=${dart.position.origin ?? 'unknown'}, axes=${dart.position.axis_orientation ?? 'unknown'}; position ${dart.position.provenance}` : 'position unavailable'}
          </li>))}</ul>
        </details>
      </section>
    </>}
  </div>
}
