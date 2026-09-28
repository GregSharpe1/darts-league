import {
  useAdminDivisions,
  useAdminPendingResults,
  useAdminPlayers,
  useConfirmPendingResult,
  usePollResults,
  useRejectPendingResult,
} from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'
import { PendingResultCard } from './PendingResultCard'

export function PendingResultsPage() {
  const playersQuery = useAdminPlayers()
  const isAuthenticated = playersQuery.isSuccess
  const divisionsQuery = useAdminDivisions(isAuthenticated)
  const pendingQuery = useAdminPendingResults(isAuthenticated)
  const confirmMutation = useConfirmPendingResult()
  const pollMutation = usePollResults()
  const rejectMutation = useRejectPendingResult()

  return (
    <>
      <section className="page-intro">
        <span className="eyebrow">Match results inbox</span>
        <h1>Pending results</h1>
        <p>Results reported automatically from the scoring app land here every 15 minutes on weekdays between 9am and 5pm. Match each reported player to a league player before the score is set.</p>
      </section>

      {!isAuthenticated ? (
        <StateNotice message="Log in from the admin page to review pending results." compact />
      ) : (
        <section className="admin-grid admin-grid-wide">
          <article className="admin-card">
            <div className="toolbar-actions">
              <h2>Awaiting confirmation</h2>
              <button
                className="refresh-button"
                type="button"
                title="Fetch new results"
                onClick={() => pollMutation.mutate()}
                disabled={pollMutation.isPending}
              >
                <span aria-hidden="true">↻</span>
                {pollMutation.isPending ? 'Fetching...' : 'Fetch new results'}
              </button>
            </div>
            {pendingQuery.isLoading ? <StateNotice message="Loading pending results..." compact /> : null}
            {pendingQuery.error ? <StateNotice tone="error" message={readError(pendingQuery.error)} compact /> : null}
            {pollMutation.error ? <StateNotice tone="error" message={readError(pollMutation.error)} compact /> : null}
            {pendingQuery.data?.length === 0 ? <StateNotice message="No results are waiting for confirmation." compact /> : null}
            {confirmMutation.error ? <StateNotice tone="error" message={readError(confirmMutation.error)} compact /> : null}
            {rejectMutation.error ? <StateNotice tone="error" message={readError(rejectMutation.error)} compact /> : null}
            <div className="admin-fixtures">
              {(pendingQuery.data ?? []).map((pending) => (
                <PendingResultCard
                  key={pending.id}
                  pending={pending}
                  players={playersQuery.data ?? []}
                  divisions={divisionsQuery.data ?? []}
                  onConfirm={(payload) => confirmMutation.mutateAsync(payload)}
                  onReject={(pendingId) => rejectMutation.mutateAsync(pendingId)}
                  isConfirming={confirmMutation.isPending}
                  isRejecting={rejectMutation.isPending}
                />
              ))}
            </div>
          </article>
        </section>
      )}
    </>
  )
}
