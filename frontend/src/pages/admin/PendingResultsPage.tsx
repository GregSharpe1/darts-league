import { useState } from 'react'
import { ApiError, formatWhen, useAdminDivisions, useAdminPendingResults, useAdminPlayers, usePendingDetail, usePollResults, useSeasonSummary } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'
import { PendingResultCard } from './PendingResultCard'
import './pendingReview.css'

export function PendingResultsPage() {
  const players = useAdminPlayers()
  const divisions = useAdminDivisions(players.isSuccess)
  const season = useSeasonSummary()
  const pending = useAdminPendingResults(players.isSuccess)
  const poll = usePollResults()
  const [selectedId, setSelectedId] = useState<number>()
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')
  const selected = pending.data?.find(item => item.id === selectedId) ?? pending.data?.[0]
  const detail = usePendingDetail(players.isSuccess ? selected?.id : undefined)
  const reload = async () => {
    const results = await Promise.all([detail.refetch(), season.refetch(), players.refetch(), divisions.refetch(), pending.refetch()])
    for (const result of results) if (result.error) throw result.error
  }

  return <>
    <section className="page-intro">
      <span className="eyebrow">Match results inbox</span><h1>Pending results</h1>
      <p>Review the source, verify both players and choose a league fixture before publishing. Imported results do not affect standings until approved.</p>
    </section>
    {message && <p role="status" className="pending-feedback">{message}</p>}
    {players.isPending ? <StateNotice message="Checking admin access..." compact /> : players.error ?
      <StateNotice tone="error" message={players.error instanceof ApiError && players.error.status === 401 ? 'Log in from the admin page to review pending results.' : readError(players.error)} compact /> :
      <section className="pending-inbox" aria-label="Results inbox" aria-busy={busy}>
        <aside className="admin-card pending-rail" aria-label="Awaiting confirmation">
          <span className="eyebrow">Scoring app imports</span><h2>Awaiting confirmation</h2>
          <button className="refresh-button" type="button" onClick={() => { setMessage(''); poll.mutate(undefined, { onSuccess: () => setMessage('Fetch complete. Inbox refreshed.') }) }} disabled={busy || poll.isPending}>
            {poll.isPending ? 'Fetching...' : 'Fetch new results'}
          </button>
          {poll.error && <StateNotice tone="error" message={readError(poll.error)} compact />}
          {pending.isLoading && <StateNotice message="Loading pending results..." compact />}
          {pending.error && <><StateNotice tone="error" message={readError(pending.error)} compact /><button type="button" className="secondary-button" onClick={() => void pending.refetch()}>Retry inbox</button></>}
          {pending.data?.length === 0 && <StateNotice message="No results are waiting for confirmation." compact />}
          <div className="pending-receipts">{pending.data?.map(item => <button type="button" key={item.id} className="pending-receipt" aria-pressed={item.id === selected?.id} disabled={busy} onClick={() => { setSelectedId(item.id); setMessage('') }}>
            <span>{item.status === 'review_blocked' ? 'Source conflict' : 'Needs review'}</span>
            <strong>{item.player_one_name} / {item.player_two_name}</strong>
            <b>{item.player_one_legs} - {item.player_two_legs}</b>
            <small>Received {formatWhen(item.received_at)}</small>
          </button>)}</div>
        </aside>
        <div className="pending-selected">
          {selected && detail.isPending && <StateNotice message="Loading selected result..." compact />}
          {selected && detail.error && <><StateNotice tone="error" message={readError(detail.error)} compact /><button type="button" onClick={() => void detail.refetch()}>Retry selected result</button></>}
          {(divisions.error || season.error) && <StateNotice tone="error" message={readError(divisions.error ?? season.error)} compact />}
          {selected && detail.data && !detail.error && <PendingResultCard key={`${selected.id}-${detail.dataUpdatedAt}`} detail={detail.data} players={players.data ?? []} divisions={divisions.data ?? []} season={season.isError ? undefined : season.data} disabled={busy || poll.isPending || detail.isFetching} onReload={reload} onBusy={setBusy} onComplete={setMessage} />}
          {!selected && pending.isSuccess && <section className="admin-card"><span className="eyebrow">Inbox clear</span><h2>Ready for the next match</h2><p>Fetch new results to check for new imports.</p></section>}
        </div>
      </section>}
  </>
}
