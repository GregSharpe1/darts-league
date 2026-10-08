import { useState } from 'react'
import { Heatmap } from '../../components/match'
import type { Provenance } from '../../components/match'
import { segmentLabel } from '../../components/match/geometry'
import { playerPlotThrows } from '../../lib/playerStatistics'
import type { PlayerStatistics } from '../../lib/playerStatistics'

export function PlayerThrows({ stats }: { readonly stats: PlayerStatistics }) {
  const [entry, setEntry] = useState<'all' | Provenance>('all')
  const selected = stats.throws.filter(d => entry === 'all' || d.entry_type === entry)
  const positions = selected.filter(d => d.position !== null).length
  const plottable = selected.filter(d => d.plottable).length
  const segments = new Map<string, number>()
  for (const dart of selected) { const label = segmentLabel(dart.segment); segments.set(label, (segments.get(label) ?? 0) + 1) }
  return <section className="content-panel" aria-labelledby="player-throws-title">
    <div className="player-section-heading"><h2 id="player-throws-title">Recorded throws</h2><span>Season {stats.season_id}</span></div>
    <div className="player-filters" role="group" aria-label="Entry type">
      {(['all', 'manual', 'automatic', 'unknown'] as const).map(type => <button key={type} type="button" aria-pressed={entry === type} onClick={() => setEntry(type)}>{type[0].toUpperCase() + type.slice(1)}</button>)}
    </div>
    <div className="player-throw-grid">
      <div className="player-heatmap"><Heatmap throws={playerPlotThrows(selected)} label={stats.preferred_name} synthetic={false} /></div>
      <div className="player-throw-notes">
        <h3>Position coverage</h3>
        <p role="status">{selected.length} recorded throws; {positions} known positions; {plottable} plottable in this selection.</p>
        <p>{stats.coverage.matches_with_recorded_throws} of {stats.coverage.eligible_matches} eligible matches have recorded throws; {stats.coverage.matches_with_positions} have positions.</p>
        <p>Across all entries: {stats.coverage.known_positions} / {stats.coverage.recorded_throws} recorded throws have positions{stats.coverage.position_fraction === null ? ' (fraction unavailable)' : ` (${(stats.coverage.position_fraction * 100).toFixed(1)}%)`}; {stats.coverage.plottable_positions} are plottable. This is not coverage of all match darts.</p>
        <p>Only positions marked plottable by the server are passed to the board. {positions - plottable} known positions in this selection are withheld because their geometry is not verified; {selected.length - positions} throws have no position.</p>
        <p>Manual board inputs do not establish physical-board accuracy. Automatic and unknown geometry remain unavailable until verified. No locations are inferred from segments.</p>
        {selected.length === 0 ? <p>No recorded throws for this selection.</p> : <><h3>Recorded segments</h3><ul className="player-segments">{[...segments].map(([label, total]) => <li key={label}>{label}: <strong>{total}</strong></li>)}</ul></>}
      </div>
    </div>
  </section>
}
