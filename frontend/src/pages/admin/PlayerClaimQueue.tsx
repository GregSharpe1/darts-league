import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { StateNotice } from '../../components/StateNotice'
import { request } from '../../lib/api'
import { usePlayerAccess } from '../../lib/playerAccess'
import { readError } from '../../lib/utils'

type Claim = {
  id: number
  email: string
  display_name?: string
  nickname?: string
  competition: string
}

const claimsKey = ['admin-player-claims']

export function PlayerClaimQueue() {
  const access = usePlayerAccess()
  const client = useQueryClient()
  const claims = useQuery({
    queryKey: claimsKey,
    queryFn: ({ signal }) => request<Claim[]>('/api/admin/player-claims', { signal, cache: 'no-store' }),
    enabled: access.data?.enabled === true,
    retry: false,
    gcTime: 0,
  })
  const decide = useMutation({
    mutationFn: ({ id, approve }: { id: number; approve: boolean }) =>
      request<void>(`/api/admin/player-claims/${id}/decision`, { method: 'POST', body: JSON.stringify({ approve }) }),
    onSuccess: () => client.invalidateQueries({ queryKey: claimsKey }),
  })

  if (!access.data?.enabled) return null

  return <section className="admin-card" aria-labelledby="player-claims-title">
    <h2 id="player-claims-title">Profile claims</h2>
    <p>Verify who is requesting access before approving. They have verified access to their email address.</p>
    {claims.isPending ? <p role="status">Loading profile claims...</p> :
      claims.isError ? <StateNotice tone="error" message={readError(claims.error)} /> :
        claims.data.length === 0 ? <p>No pending profile claims.</p> :
          <ul className="identity-suggestions">{claims.data.map(claim => <li key={claim.id}>
            <div className="identity-entry-list">
              <strong>{claim.display_name ?? 'Profile unavailable'}{claim.nickname ? ` (${claim.nickname})` : ''}</strong>
              <span>{claim.competition}</span>
              <span>Verified email {claim.email}</span>
            </div>
            <div className="toolbar-actions">
              <button type="button" disabled={!claim.display_name || decide.isPending} onClick={() => {
                if (window.confirm(`Give ${claim.email} access to the player profile for ${claim.display_name}?`)) {
                  decide.mutate({ id: claim.id, approve: true })
                }
              }}>Approve claim</button>
              <button type="button" className="secondary-button" disabled={decide.isPending}
                onClick={() => decide.mutate({ id: claim.id, approve: false })}>Decline claim</button>
            </div>
          </li>)}</ul>}
    {decide.isSuccess ? <div role="status"><StateNotice message={decide.variables.approve ? 'Profile claim approved.' : 'Profile claim declined.'} /></div> : null}
    {decide.error ? <div role="alert"><StateNotice tone="error" message={readError(decide.error)} /></div> : null}
  </section>
}
