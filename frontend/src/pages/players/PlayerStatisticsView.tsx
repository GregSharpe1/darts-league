import { Link } from 'react-router-dom'
import type { PlayerStatistics } from '../../lib/playerStatistics'
import { PlayerThrows } from './PlayerThrows'

function Metric({ label, value, known, total, note }: {
  readonly label: string; readonly value: string | number | null; readonly known: number; readonly total: number; readonly note: string
}) {
  return <div className="player-metric" role="group" aria-label={label}>
    <h3>{label}</h3><strong>{value ?? 'Unavailable'}</strong>
    <span>{known} of {total} eligible matches</span><p>{note}</p>
  </div>
}
const average = (value: number | null) => value === null ? null : value.toFixed(2)

export function PlayerStatisticsView({ stats: s, admin }: { readonly stats: PlayerStatistics; readonly admin: boolean }) {
  const c = s.coverage
  const prefix = admin ? '/admin' : ''
  const checkout = s.checkout_hits === null || s.checkout_attempts === null ? null :
    s.checkout_attempts === 0 ? '0 / 0 - no attempts' : `${s.checkout_hits} / ${s.checkout_attempts} (${s.checkout_percentage?.toFixed(1)}%)`
  return <>
    <div className="player-hero content-panel">
      <div><span className="eyebrow">Season {s.season_id} / 501 / First to 3</span><h1>{s.preferred_name}</h1>
        <p>{admin ? 'Current recorded results in this season.' : 'Revealed results in this season only.'} Missing evidence is never treated as zero.</p></div>
      <dl className="player-record">
        {([['Played', s.played], ['Won', s.won], ['Lost', s.lost], ['Points', s.points], ['Legs for', s.legs_for], ['Legs against', s.legs_against], ['Leg difference', s.leg_difference]] as const)
          .map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}
      </dl>
    </div>
    <section className="content-panel" aria-labelledby="player-metrics-title">
      <div className="player-section-heading"><h2 id="player-metrics-title">Scoring & finishing</h2><span>{c.matches_with_detail} of {c.eligible_matches} matches with throw detail</span></div>
      <p className="player-note">Coverage is specific to each metric. Values describe known evidence, not estimates for missing matches.</p>
      <div className="player-metrics">
        <Metric label="Mean match average" value={average(s.match_average_mean)} known={c.match_average_mean} total={c.eligible_matches} note="Arithmetic mean of known match averages, including manual results." />
        <Metric label="Dart-weighted average" value={average(s.dart_weighted_average)} known={c.dart_weighted_average} total={c.eligible_matches} note="3 x total points / actual darts, from complete validated histories only." />
        <Metric label="First-nine match average mean" value={average(s.first_nine_match_average_mean)} known={c.first_nine_match_average_mean} total={c.eligible_matches} note="Mean of known per-match first-nine averages; not dart-weighted or per-leg." />
        <Metric label="Checkout hits / attempts" value={checkout} known={c.checkout} total={c.eligible_matches} note="Sum of known hits / sum of known attempts, not a mean of percentages." />
        <Metric label="180s" value={s.total_180} known={c.total_180} total={c.eligible_matches} note="Sum of known match totals; partial visits are not extrapolated." />
        <Metric label="Best leg" value={s.best_leg_darts === null ? null : `${s.best_leg_darts} darts`} known={c.best_leg_darts} total={c.eligible_matches} note="Fewest actual darts in a won leg from complete validated history." />
        <Metric label="Highest finish" value={s.highest_finish} known={c.highest_finish} total={c.eligible_matches} note="Highest known supplied or witnessed checkout, not a claim of full coverage." />
      </div>
    </section>
    <PlayerThrows stats={s} />
    <section className="content-panel" aria-labelledby="player-history-title">
      <div className="player-section-heading"><h2 id="player-history-title">Match history</h2><span>{s.played} eligible matches</span></div>
      {s.history.length === 0 ? <p>No eligible matches yet. Statistics will appear when results are available.</p> :
        <ol className="player-history">{s.history.map(h => <li key={h.fixture_id}>
          <div><span className="eyebrow">Week {h.week_number} / <time dateTime={h.scheduled_at}>{new Date(h.scheduled_at).toLocaleDateString('en-GB', { timeZone: 'Europe/London', day: 'numeric', month: 'short', year: 'numeric' })}</time></span>
            <h3>vs <Link to={`${prefix}/seasons/${s.season_id}/players/${h.opponent_id}`}>{h.opponent_name}</Link></h3>
            <p>Average {average(h.match_average) ?? 'unavailable'} / Detail: {h.detail_coverage}</p></div>
          <div className="player-history-result"><strong>{h.legs_for} - {h.legs_against}</strong><span>{h.won ? 'Won' : 'Lost'}</span></div>
          <div className="toolbar-actions"><Link className="button-link" to={`${prefix}/matches/${h.fixture_id}`}>Match {h.fixture_id}</Link></div>
        </li>)}</ol>}
    </section>
  </>
}
