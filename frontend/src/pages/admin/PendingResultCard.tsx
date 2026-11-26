import { useRef, useState } from 'react'
import { ApiError, formatWhen, useAdminDivisionFixtures, useConfirmPendingResult, useRejectPendingResult } from '../../lib/api'
import type { AdminFixture, CompetitionType, Division, Player, SeasonSummary } from '../../lib/api'
import { approvalProblem, reviewBlocked, type PendingDetail } from '../../lib/pendingReview'
import { readError } from '../../lib/utils'
import { SourceReview } from './SourceReview'

type Props = {
  readonly detail: PendingDetail
  readonly players: readonly Player[]
  readonly divisions: readonly Division[]
  readonly season: SeasonSummary | undefined
  readonly leagueError: Error | null
  readonly disabled: boolean
  readonly onReload: (competition: CompetitionType | '') => Promise<void>
  readonly onBusy: (busy: boolean) => void
  readonly onComplete: (message: string) => void
}

export function PendingResultCard({ detail: latestDetail, players, divisions, season, leagueError, disabled, onReload, onBusy, onComplete }: Props) {
  const [detail] = useState(latestDetail)
  const reportedCompetition = detail.pending_result.competition_type
  const [competition, setCompetition] = useState<CompetitionType | ''>(reportedCompetition ?? '')
  const tournament = competition === 'tournament'
  const candidates = latestDetail.tournament_candidates ?? []
  const candidateKey = (candidate: typeof candidates[number]) => `${candidate.tournament_slug}/${candidate.match_number}`
  const [matchKey, setMatchKey] = useState(() => candidates.length === 1 ? candidateKey(candidates[0]) : '')
  const [mappingIndex, setMappingIndex] = useState('')
  const candidate = candidates.find(item => candidateKey(item) === matchKey)
  const tournamentMapping = candidate?.mappings.length === 1 ? candidate.mappings[0] : candidate?.mappings[mappingIndex === '' ? -1 : Number(mappingIndex)]
  const suggestion = detail.suggestion?.season_id === season?.id ? detail.suggestion : undefined
  const [seasonId, setSeasonId] = useState(detail.suggestion?.season_id ?? 0)
  const [divisionId, setDivisionId] = useState(detail.suggestion?.division_id ? String(detail.suggestion.division_id) : '')
  const [playerIds, setPlayerIds] = useState(() => detail.players.slice(0, 2).map(player => detail.suggestion?.mapping[player.match_player_id] ?? 0))
  const [prefillPending, setPrefillPending] = useState(Boolean(detail.suggestion?.fixture_id))
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
  const division = competition === 'league' ? divisions.find(item => String(item.id) === divisionId) : undefined
  const fixtures = useAdminDivisionFixtures(division?.slug ?? '', Boolean(division))
  const roster = players.filter(player => player.division_id === division?.id && player.status === 'assigned')
  const pairs = (fixtures.data ?? []).flatMap(week => week.fixtures.map(item => ({ ...item, week: week.week_number })))
    .filter(item => playerIds[0] !== playerIds[1] && [item.player_one_id, item.player_two_id].every(id => id !== undefined && playerIds.includes(id)))
  if (prefillPending && suggestion && fixtures.isSuccess && !fixtures.isFetching) {
    setPrefillPending(false)
    setFixture(pairs.find(item => item.id === suggestion?.fixture_id && item.game_variant === '501' && item.legs_to_win === Math.max(...detail.players.map(player => player.legs_won)) && item.expected_result !== undefined))
  }
  const selection = { seasonId, playerIds, fixture, replace, reason, attest, missingDateReason }
  const sourceChanged = latestDetail.digest !== detail.digest || latestDetail.pending_result.status !== detail.pending_result.status || latestDetail.review_reason !== detail.review_reason || latestDetail.changed_import !== detail.changed_import
  const needsReplacement = Boolean((!tournament && fixture?.expected_result) || detail.changed_import)
  const tournamentProblem = () => {
    if (reviewBlocked(detail)) return 'Source conflict: approval is blocked. Reject this import or resolve it at the source.'
    if (detail.pending_result.status !== 'pending') return 'This import is no longer pending. Reload the inbox.'
    if (!candidates.length) return 'No eligible unplayed tournament match is available.'
    if (!candidate) return 'Choose the tournament and bracket match.'
    if (!tournamentMapping) return 'Verify which entrant corresponds to each source player.'
    if (needsReplacement && (!replace || !reason.trim())) return 'Confirm the changed source version and give a reason.'
    if (detail.settings_evidence !== 'source_reported' && !attest) return 'Attest the legacy match format before approving.'
    if (!detail.played_at && !missingDateReason.trim()) return 'Explain how the match was verified without a source date.'
    return ''
  }
  const leagueProblem = () => {
    if (leagueError) return readError(leagueError)
    if (season?.status === 'completed' || !season?.season_started) return 'The current season must be started and writable.'
    if (seasonId !== season.id) return 'Choose the current season explicitly.'
    return approvalProblem(detail, selection)
  }
  const problem = stale || sourceChanged ? 'The review is out of date. Reload source and match, then review again.' :
    competition === '' ? 'Choose the competition before approving.' : tournament ? tournamentProblem() : leagueProblem()
  const clearReview = () => { setReplace(false); setReason('') }
  const clearFixture = () => { setPrefillPending(false); setFixture(undefined); clearReview() }
  const run = async (action: 'approve' | 'reject' | 'reload') => {
    if (locked.current || controlsBusy || (action === 'approve' && problem) || (action === 'reject' && !rejectReason.trim())) return
    locked.current = true; setBusy(true); onBusy(true); setError('')
    try {
      if (action === 'reload') {
        if (division) {
          const refreshed = await fixtures.refetch()
          if (refreshed.error) throw refreshed.error
        }
        await onReload(competition)
        setStale(false); clearFixture()
      } else if (action === 'reject') {
        await reject.mutateAsync({ pendingId: detail.pending_result.id, reason: rejectReason.trim() })
        onComplete(tournament ? 'Result rejected.' : 'Result rejected. The league score was not changed.')
      } else if (tournament && candidate && tournamentMapping) {
        await confirm.mutateAsync({ pendingId: detail.pending_result.id, payload: {
          tournament_slug: candidate.tournament_slug, match_number: candidate.match_number,
          mapping: tournamentMapping, expected_result: null, replace: needsReplacement && replace,
          reason: reason.trim(), attest_format: attest, missing_date_reason: missingDateReason.trim(),
        } })
        onComplete('Result approved. Tournament bracket refreshed.')
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
      <h2>Verify the match</h2>
      {competition === 'league' && suggestion && <p className="fixture-meta">Prefilled from saved Autodarts mappings. Verify these suggestions; pending results still require approval.</p>}
      <fieldset disabled={controlsBusy || stale}>
        <legend className="sr-only">Approval mapping</legend>
        {!reportedCompetition && <div className="field">
          <label htmlFor="review-competition">Competition type</label>
          <select id="review-competition" value={competition} onChange={event => {
            const value = event.target.value
            setCompetition(value === 'league' || value === 'tournament' ? value : '')
            clearReview()
          }}>
            <option value="">Choose competition...</option><option value="league">League match</option><option value="tournament">Tournament match</option>
          </select>
        </div>}
        {tournament && <>
          <div className="field"><label htmlFor="review-tournament-match">Tournament and bracket match</label>
            <select id="review-tournament-match" value={candidate ? matchKey : ''} disabled={!candidates.length} onChange={event => { setMatchKey(event.target.value); setMappingIndex(''); clearReview() }}>
              <option value="">Select matching bracket match...</option>
              {candidates.map(item => <option value={candidateKey(item)} key={candidateKey(item)}>{item.tournament_name} ({item.tournament_slug}) - Match {item.match_number}: {item.players.join(' vs ')}</option>)}
            </select>
          </div>
          {candidate && candidate.mappings.length > 1 && <div className="field">
            <label htmlFor="review-entrant-order">Tournament entrant for source player 1 ({detail.players[0]?.display_name}, {detail.players[0]?.legs_won} legs)</label>
            <select id="review-entrant-order" value={mappingIndex} onChange={event => { setMappingIndex(event.target.value); clearReview() }}>
              <option value="">Verify entrant...</option>
              {candidate.players.map((player, index) => <option key={player} value={index}>{player}</option>)}
            </select>
          </div>}
        </>}
        {competition === 'league' && <>
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
        </>}
        {competition && <>{needsReplacement && <div className="pending-warning">
          <label className="pending-check"><input type="checkbox" checked={replace} onChange={event => setReplace(event.target.checked)} />{tournament ? 'Approve this changed source version' : 'Replace the existing result or changed source import'}</label>
          <p>{tournament ? 'The source version and review reason will be audited.' : 'This publishes the imported score in place of the prior result. The change and reason will be audited.'}</p>
          <div className="field"><label htmlFor="review-reason">Replacement reason</label><textarea id="review-reason" value={reason} onChange={event => setReason(event.target.value)} maxLength={2000} /></div>
        </div>}
        {!needsReplacement && <div className="field"><label htmlFor="review-note">Review note</label><textarea id="review-note" value={reason} onChange={event => setReason(event.target.value)} maxLength={2000} /><small>{tournament ? 'Explain a source date ahead of the server clock.' : 'Explain any source date outside the season or ahead of the server clock.'}</small></div>}
        {detail.settings_evidence !== 'source_reported' && <label className="pending-check"><input type="checkbox" checked={attest} onChange={event => setAttest(event.target.checked)} />I verified this legacy match was 501, first to {fixture?.legs_to_win ?? Math.max(...detail.players.map(player => player.legs_won))} legs, double out.</label>}
        {!detail.played_at && <div className="field"><label htmlFor="review-date-reason">Missing source date reason</label><textarea id="review-date-reason" value={missingDateReason} onChange={event => setMissingDateReason(event.target.value)} maxLength={2000} /><small>Explain how you verified the intended match without a source match date.</small></div>}
        </>}
      </fieldset>
      <p id="approval-problem" role="status">{problem || 'Mapping verified. Ready to approve this match.'}</p>
      {error && <p role="alert" className="form-error">{error}</p>}
      <div className="score-actions"><button type="button" disabled={controlsBusy || Boolean(problem) || fixtures.isFetching || Boolean(fixtures.error)} aria-describedby="approval-problem" onClick={() => void run('approve')}>{confirm.isPending ? 'Approving...' : 'Approve result'}</button>
        <button type="button" className="secondary-button" disabled={controlsBusy} onClick={() => void run('reload')}>{tournament ? 'Reload source and bracket' : 'Reload source and fixture'}</button></div>
      <details className="pending-reject"><summary>Reject this import</summary><p>Rejecting removes this import from the inbox without changing a competition score.</p>
        <div className="field"><label htmlFor="reject-reason">Rejection reason</label><textarea id="reject-reason" value={rejectReason} onChange={event => setRejectReason(event.target.value)} disabled={controlsBusy} maxLength={2000} /></div>
        <button type="button" className="secondary-button" disabled={controlsBusy || !rejectReason.trim()} onClick={() => void run('reject')}>{reject.isPending ? 'Rejecting...' : 'Reject result'}</button>
      </details>
    </section>
  </article>
}
