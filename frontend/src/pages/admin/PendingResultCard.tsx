import { useRef, useState } from 'react'
import { ApiError, formatWhen, useAdminDivisionFixtures, useConfirmPendingResult, useRejectPendingResult } from '../../lib/api'
import type { AdminFixture, Division, Player, SeasonSummary } from '../../lib/api'
import { approvalProblem, type PendingDetail } from '../../lib/pendingReview'
import { readError } from '../../lib/utils'
import { SourceReview } from './SourceReview'

type Props = {
  readonly detail: PendingDetail
  readonly players: readonly Player[]
  readonly divisions: readonly Division[]
  readonly season: SeasonSummary | undefined
  readonly disabled: boolean
  readonly onReload: () => Promise<void>
  readonly onBusy: (busy: boolean) => void
  readonly onComplete: (message: string) => void
}

export function PendingResultCard({ detail, players, divisions, season, disabled, onReload, onBusy, onComplete }: Props) {
  const [seasonId, setSeasonId] = useState(0)
  const [divisionId, setDivisionId] = useState('')
  const [playerIds, setPlayerIds] = useState([0, 0])
  const [fixture, setFixture] = useState<AdminFixture>()
  const [replace, setReplace] = useState(false)
  const [reason, setReason] = useState('')
  const [attest, setAttest] = useState(false)
  const [missingDateReason, setMissingDateReason] = useState('')
  const [rejectReason, setRejectReason] = useState('')
  const [error, setError] = useState('')
  const [stale, setStale] = useState(false)
  const [busy, setBusy] = useState(false)
  const controlsBusy = busy || disabled
  const locked = useRef(false)
  const confirm = useConfirmPendingResult()
  const reject = useRejectPendingResult()
  const division = divisions.find(item => String(item.id) === divisionId)
  const fixtures = useAdminDivisionFixtures(division?.slug ?? '', Boolean(division))
  const roster = players.filter(player => player.division_id === division?.id && player.status === 'assigned')
  const pairs = (fixtures.data ?? []).flatMap(week => week.fixtures.map(item => ({ ...item, week: week.week_number })))
    .filter(item => playerIds[0] !== playerIds[1] && [item.player_one_id, item.player_two_id].every(id => id !== undefined && playerIds.includes(id)))
  const selection = { seasonId, playerIds, fixture, replace, reason, attest, missingDateReason }
  const problem = stale ? 'The review is out of date. Reload source and fixture, then review again.' :
    season?.status === 'completed' || !season?.season_started ? 'The current season must be started and writable.' :
    seasonId !== season.id ? 'Choose the current season explicitly.' : approvalProblem(detail, selection)
  const needsReplacement = Boolean(fixture?.expected_result || detail.changed_import)
  const clearFixture = () => { setFixture(undefined); setReplace(false); setReason('') }
  const run = async (action: 'approve' | 'reject' | 'reload') => {
    if (locked.current || controlsBusy || (action === 'approve' && problem) || (action === 'reject' && !rejectReason.trim())) return
    locked.current = true; setBusy(true); onBusy(true); setError('')
    try {
      if (action === 'reload') {
        if (division) {
          const refreshed = await fixtures.refetch()
          if (refreshed.error) throw refreshed.error
        }
        await onReload()
        setStale(false); clearFixture()
      } else if (action === 'reject') {
        await reject.mutateAsync({ pendingId: detail.pending_result.id, reason: rejectReason.trim() })
        onComplete('Result rejected. The league score was not changed.')
      } else if (fixture && fixture.expected_result !== undefined) {
        await confirm.mutateAsync({ pendingId: detail.pending_result.id, payload: {
          season_id: seasonId, fixture_id: fixture.id,
          mapping: Object.fromEntries(detail.players.map((player, index) => [player.match_player_id, playerIds[index]])),
          expected_result: fixture.expected_result, replace: needsReplacement && replace, reason: reason.trim(), attest_format: attest, missing_date_reason: missingDateReason.trim(),
        } })
        onComplete('Result approved. Fixture and standings refreshed.')
      }
    } catch (failure) {
      setError(readError(failure))
      if (failure instanceof ApiError && (failure.status === 409 || failure.status === 422)) setStale(true)
    } finally {
      locked.current = false; setBusy(false); onBusy(false)
    }
  }

  return <article className="pending-review" aria-label="Selected result review">
    <SourceReview detail={detail} />
    <section className="admin-card pending-target">
      <span className="eyebrow">Publish to the league</span><h2>Verify the match</h2>
      <p>Choose the season, league and both players. Names are not matched automatically.</p>
      <fieldset disabled={controlsBusy || stale}>
        <legend className="sr-only">Approval mapping</legend>
        <div className="pending-fields">
          <div className="field"><label htmlFor="review-season">Season</label><select id="review-season" value={seasonId || ''} onChange={event => { setSeasonId(Number(event.target.value)); clearFixture() }}>
            <option value="">Select current season...</option>{season && <option value={season.id}>{season.name} (current season)</option>}
          </select></div>
          <div className="field"><label htmlFor="review-division">League</label><select id="review-division" value={divisionId} onChange={event => { setDivisionId(event.target.value); setPlayerIds([0, 0]); clearFixture() }}>
            <option value="">Select league...</option>{divisions.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}
          </select></div>
          {detail.players.slice(0, 2).map((source, index) => <div className="field" key={source.match_player_id}>
            <label htmlFor={`review-player-${index}`}>Player {index + 1}: {source.display_name}</label>
            <select id={`review-player-${index}`} disabled={!division} value={playerIds[index] || ''} onChange={event => { setPlayerIds(playerIds.map((id, i) => i === index ? Number(event.target.value) : id)); clearFixture() }}>
              <option value="">Select league player...</option>{roster.map(player => <option key={player.id} value={player.id} disabled={player.id === playerIds[1 - index]}>{player.admin_label}</option>)}
            </select>
          </div>)}
        </div>
        <div className="field"><label htmlFor="review-fixture">Fixture</label><select id="review-fixture" disabled={!pairs.length || fixtures.isFetching || Boolean(fixtures.error)} value={fixture?.id ?? ''} onChange={event => { clearFixture(); setFixture(pairs.find(item => item.id === Number(event.target.value))) }}>
          <option value="">Select matching fixture...</option>{pairs.map(item => <option key={item.id} value={item.id}>Week {item.week}: {item.player_one} vs {item.player_two} - {formatWhen(item.scheduled_at)}{item.result ? ' (result recorded)' : ''}</option>)}
        </select></div>
        {fixtures.isFetching && <p role="status">Loading fixtures...</p>}
        {fixtures.error && <p role="alert">{readError(fixtures.error)}</p>}
        {division && playerIds.every(Boolean) && !fixtures.isFetching && !pairs.length && <p>No fixture matches this exact pair in the selected league.</p>}
        {fixture && <div className="pending-comparison">
          <h3>Score comparison</h3><p>{fixture.player_one} vs {fixture.player_two}<br />Scheduled: {formatWhen(fixture.scheduled_at)}</p>
          <dl><div><dt>Existing league result</dt><dd>{fixture.expected_result ? `${fixture.expected_result.player_one_legs} - ${fixture.expected_result.player_two_legs}` : fixture.result ? 'Snapshot unavailable' : 'Not yet scored'}</dd></div>
            <div><dt>Incoming score (fixture order)</dt><dd>{[fixture.player_one_id, fixture.player_two_id].map(id => detail.players[playerIds.indexOf(id ?? 0)]?.legs_won ?? '?').join(' - ')}</dd></div></dl>
          {fixture.expected_result && <p>Existing averages: {fixture.expected_result.player_one_average?.toFixed(1) ?? 'Unavailable'} / {fixture.expected_result.player_two_average?.toFixed(1) ?? 'Unavailable'}</p>}
        </div>}
        {needsReplacement && <div className="pending-warning">
          <label className="pending-check"><input type="checkbox" checked={replace} onChange={event => setReplace(event.target.checked)} />Replace the existing result or changed source import</label>
          <p>This publishes the imported score in place of the prior result. The change and reason will be audited.</p>
          <div className="field"><label htmlFor="review-reason">Replacement reason</label><textarea id="review-reason" value={reason} onChange={event => setReason(event.target.value)} maxLength={2000} /></div>
        </div>}
        {!needsReplacement && <div className="field"><label htmlFor="review-note">Review note</label><textarea id="review-note" value={reason} onChange={event => setReason(event.target.value)} maxLength={2000} /><small>Explain any source date outside the season or ahead of the server clock.</small></div>}
        {detail.settings_evidence !== 'source_reported' && <label className="pending-check"><input type="checkbox" checked={attest} onChange={event => setAttest(event.target.checked)} />I verified this legacy match was 501, first to 3 legs, double out.</label>}
        {!detail.played_at && <div className="field"><label htmlFor="review-date-reason">Missing source date reason</label><textarea id="review-date-reason" value={missingDateReason} onChange={event => setMissingDateReason(event.target.value)} maxLength={2000} /><small>Explain how you verified the intended fixture without a source match date.</small></div>}
      </fieldset>
      <p id="approval-problem" role="status">{problem || 'Mapping verified. Ready to approve this fixture.'}</p>
      {error && <p role="alert" className="form-error">{error}</p>}
      <div className="score-actions"><button type="button" disabled={controlsBusy || Boolean(problem) || fixtures.isFetching || Boolean(fixtures.error)} aria-describedby="approval-problem" onClick={() => void run('approve')}>{confirm.isPending ? 'Approving...' : 'Approve result'}</button>
        <button type="button" className="secondary-button" disabled={controlsBusy} onClick={() => void run('reload')}>Reload source and fixture</button></div>
      <details className="pending-reject"><summary>Reject this import</summary><p>Rejecting removes this import from the inbox without changing a league score.</p>
        <div className="field"><label htmlFor="reject-reason">Rejection reason</label><textarea id="reject-reason" value={rejectReason} onChange={event => setRejectReason(event.target.value)} disabled={controlsBusy} maxLength={2000} /></div>
        <button type="button" className="secondary-button" disabled={controlsBusy || !rejectReason.trim()} onClick={() => void run('reject')}>{reject.isPending ? 'Rejecting...' : 'Reject result'}</button>
      </details>
    </section>
  </article>
}
