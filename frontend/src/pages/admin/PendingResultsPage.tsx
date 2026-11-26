import { useState } from 'react'
import { ApiError, formatWhen, useAdminDivisions, useAdminPendingResults, useAdminPlayers, usePendingDetail, usePollResults, useSeasonSummary } from '../../lib/api'
import type { CompetitionType } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'
import { PendingResultCard } from './PendingResultCard'
import './pendingReview.css'

export function PendingResultsPage() {
  const players = useAdminPlayers()
  const divisions = useAdminDivisions(players.isSuccess)
  const season = useSeasonSummary()
  const playersError = players.error
  // The players handler runs after admin auth and can fail while loading the league.
  const leagueUnavailable = playersError instanceof ApiError &&
    (playersError.status === 404 ||
      (playersError.status >= 500 && playersError.status < 600 && playersError.code !== 'request_failed'))
  const canReview = players.isSuccess || leagueUnavailable
  const pending = useAdminPendingResults(canReview)
  const poll = usePollResults()
  const [selectedId, setSelectedId] = useState<number>()
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')
  const [reloadVersion, setReloadVersion] = useState(0)
  const selected = canReview ? pending.data?.find(item => item.id === selectedId) ?? pending.data?.[0] : undefined
  const detail = usePendingDetail(selected?.id)
  const reload = async (competition: CompetitionType | '') => {
    const leagueQueries = competition === 'league' ? [season.refetch(), players.refetch(), divisions.refetch()] : []
    const results = await Promise.all([detail.refetch(), pending.refetch(), ...leagueQueries])
    for (const result of results) if (result.error) throw result.error
    setReloadVersion(version => version + 1)
  }

  const loginRequired = [playersError, pending.error].some(error =>
    error instanceof ApiError && (error.status === 401 || error.status === 403))

  return <>
    <section className="page-intro">
      <span className="eyebrow">Match results inbox</span><h1>Pending results</h1>
      <p>Verify both players and the intended league fixture or tournament match before approving.</p>
    </section>
    {message && <p role="status" className="pending-feedback">{message}</p>}
    {loginRequired ?
      <StateNotice tone="error" message="Log in from the admin page to review pending results." compact /> :
      !canReview ? players.isPending ?
        <StateNotice message="Checking admin access..." compact /> :
        <StateNotice tone="error" message={readError(playersError)} compact /> :
      <section className="pending-inbox" aria-label="Results inbox" aria-busy={busy}>
        <aside className="admin-card pending-rail" aria-label="Awaiting confirmation">
          <span className="eyebrow">Scoring app imports</span><h2>Awaiting confirmation</h2>
          <div className="toolbar-actions"><button className="button-link" type="button" onClick={() => { setMessage(''); poll.mutate(undefined, { onSuccess: () => setMessage('Fetch complete. Inbox refreshed.') }) }} disabled={busy || poll.isPending}>
            {poll.isPending ? 'Fetching...' : 'Fetch new results'}
          </button></div>
          {poll.error && <StateNotice tone="error" message={readError(poll.error)} compact />}
          {pending.isLoading && <StateNotice message="Loading pending results..." compact />}
          {pending.error && <><StateNotice tone="error" message={readError(pending.error)} compact /><button type="button" className="secondary-button" onClick={() => void pending.refetch()}>Retry inbox</button></>}
          {pending.data?.length === 0 && <StateNotice message="No results are waiting for confirmation." compact />}
          <div className="pending-receipts">{pending.data?.map(item => <button type="button" key={item.id} className="pending-receipt" aria-pressed={item.id === selected?.id} disabled={busy} onClick={() => { setSelectedId(item.id); setMessage('') }}>
            <span>{item.competition_type === 'league' ? 'League match' : item.competition_type === 'tournament' ? 'Tournament match' : 'Competition unspecified'}{item.status === 'review_blocked' && ' / Source conflict'}</span>
            <strong>{item.player_one_name} / {item.player_two_name}</strong>
            <b>{item.player_one_legs} - {item.player_two_legs}</b>
            <small>Received {formatWhen(item.received_at)}</small>
          </button>)}</div>
        </aside>
        <div className="pending-selected">
          {selected && detail.isPending && <StateNotice message="Loading selected result..." compact />}
          {selected && detail.error && <><StateNotice tone="error" message={readError(detail.error)} compact /><button type="button" onClick={() => void detail.refetch()}>Retry selected result</button></>}
          {selected && detail.data && <PendingResultCard key={`${selected.id}-${reloadVersion}`} detail={detail.data} players={players.data ?? []} divisions={divisions.data ?? []} season={season.data} leagueError={players.error ?? divisions.error ?? season.error} disabled={busy || poll.isPending || detail.isFetching || detail.isError} onReload={reload} onBusy={setBusy} onComplete={setMessage} />}
          {!selected && pending.isSuccess && <section className="admin-card"><span className="eyebrow">Inbox clear</span><h2>Ready for the next match</h2><p>Fetch new results to check for new imports.</p></section>}
        </div>
      </section>}
  </>
}
