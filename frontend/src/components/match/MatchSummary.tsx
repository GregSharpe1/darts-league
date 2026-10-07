import type { MatchData, MatchStats } from './types'
import './match.css'

export interface MatchSummaryProps { readonly match: MatchData }

function checkout(stats: MatchStats | null) {
  if (stats?.checkout_attempts === 0) return `${stats.checkout_hits ?? 'Unknown'} / 0 (not applicable)`
  if (stats?.checkout_attempts == null || stats.checkout_hits == null) return 'Unknown'
  return `${stats.checkout_hits} / ${stats.checkout_attempts} (${(100 * stats.checkout_hits / stats.checkout_attempts).toFixed(1)}%)`
}

export function MatchSummary({ match }: MatchSummaryProps) {
  return <section className="ma ma-panel ma-summary" aria-label="Match summary">
    <div className="ma-meta"><span>501 <span className="ma-accent">/</span> First to 3 <span className="ma-accent">/</span> Double out</span>
      <span className="ma-badge">{match.synthetic ? 'Synthetic test fixture' : 'Recorded match'}</span></div>
    <div className="ma-score">
      {match.players.map((player, index) => <div className={`ma-player ma-player-${index}`} key={player.id}>
        <span className="ma-eyebrow">{player.legs_won === 3 ? 'Winner' : 'Opponent'}</span>
        <h2>{player.label}</h2>
      </div>)}
      <div className="ma-score-number" aria-label={`${match.players[0].label} ${match.players[0].legs_won}, ${match.players[1].label} ${match.players[1].legs_won}`}>
        <strong>{match.players[0].legs_won}</strong><span>:</span><strong>{match.players[1].legs_won}</strong>
      </div>
    </div>
    <div className="ma-meta"><span>Played: {match.playedAt ? <time dateTime={match.playedAt}>{match.playedAt}</time> : 'Unknown'}</span>
      <span>{match.detail ? `${match.detail.coverage === 'complete' ? 'Complete' : 'Partial'} recorded detail` : 'Summary only'}</span></div>
    <div className="ma-paired-stats">{match.players.map(player => <div key={player.id}>
      <h3>{player.label}</h3><dl className="ma-stats">
        <div><dt>Match average (source)</dt><dd>{player.stats?.match_average?.toFixed(1) ?? 'Unknown'}</dd></div>
        <div><dt>First-nine average (source)</dt><dd>{player.stats?.first_nine_average?.toFixed(1) ?? 'Not available'}</dd></div>
        <div><dt>180s (source)</dt><dd>{player.stats?.total_180 ?? 'Not available'}</dd></div>
        <div><dt>Highest finish (source)</dt><dd>{player.stats?.highest_finish ?? 'Not available'}</dd></div>
        <div><dt>Points scored</dt><dd>{player.stats?.points_scored ?? 'Unknown'}</dd></div>
        <div><dt>Darts thrown</dt><dd>{player.stats?.darts_thrown ?? 'Unknown'}</dd></div>
        <div><dt>Checkouts</dt><dd>{checkout(player.stats)}</dd></div>
      </dl>
    </div>)}</div>
  </section>
}
