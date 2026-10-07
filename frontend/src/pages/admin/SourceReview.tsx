import { useState } from 'react'
import { MatchAnalysis, MatchSummary } from '../../components/match'
import { formatWhen } from '../../lib/api'
import { reviewBlocked, toMatchData, type PendingDetail } from '../../lib/pendingReview'

export function SourceReview({ detail }: { readonly detail: PendingDetail }) {
  const [analysis, setAnalysis] = useState(false)
  const match = toMatchData(detail)
  const blocked = reviewBlocked(detail)
  return <>
    <section className="admin-card pending-provenance">
      <span className="eyebrow">Imported source / not yet published</span>
      <h2>Review source result</h2>
      <p>{detail.source} / Received {formatWhen(detail.pending_result.received_at)}</p>
      <p>Played: {detail.played_at ? formatWhen(detail.played_at) : 'Unknown - a reason is required before approval.'}</p>
      {blocked && <div className="pending-warning" role="alert"><strong>Source conflict - approval blocked</strong><p>{detail.review_reason || 'Source detail conflicts with its reported summary.'} Statistics are not presented as validated performance.</p></div>}
      {detail.changed_import && <p className="pending-warning">Changed import: a different version of this source match exists. Verify the previous mapping and explicitly authorize replacement.</p>}
      {detail.settings_evidence !== 'source_reported' && <p className="pending-warning">Legacy summary: match settings were not supplied by the source. Format attestation is required; no throw detail is inferred.</p>}
      {!match && <div className="pending-source-score" aria-label="Unverified source score"><strong>{detail.pending_result.player_one_name}</strong><b>{detail.pending_result.player_one_legs} - {detail.pending_result.player_two_legs}</b><strong>{detail.pending_result.player_two_name}</strong><small>Reported score only / {blocked ? 'conflicting source evidence' : 'unverified format'}</small>
        <small>Reported averages: {detail.pending_result.player_one_average?.toFixed(1) ?? 'Unavailable'} / {detail.pending_result.player_two_average?.toFixed(1) ?? 'Unavailable'}</small></div>}
      {match && <div className="pending-views" aria-label="Review view"><button type="button" className="secondary-button" aria-pressed={!analysis} onClick={() => setAnalysis(false)}>Summary</button><button type="button" className="secondary-button" aria-pressed={analysis} onClick={() => setAnalysis(true)}>Analysis &amp; heatmap</button></div>}
    </section>
    {match && (analysis ? <MatchAnalysis status="ready" match={match} /> : <MatchSummary match={match} />)}
  </>
}
