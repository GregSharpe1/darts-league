import type { ReactNode } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useLocation } from 'react-router-dom'
import { StateNotice } from '../../components/StateNotice'
import { request } from '../../lib/api'
import { playerAccessKey, usePlayerAccess } from '../../lib/playerAccess'
import { readError } from '../../lib/utils'

export function ReturningPlayerSignup({ action, endpoint, successMessage, children }: {
  action: string
  endpoint: string
  successMessage: (name: string) => string
  children: ReactNode
}) {
  const location = useLocation()
  const accessPath = `/player-access?${new URLSearchParams({ next: location.pathname })}`
  const access = usePlayerAccess()
  const client = useQueryClient()
  const enter = useMutation({
    mutationFn: () => request<{ preferred_name: string }>(endpoint, { method: 'POST' }),
  })
  const logout = useMutation({
    mutationFn: () => request<void>('/api/player-access/sign-out', { method: 'POST' }),
    onSuccess: async () => {
      enter.reset()
      await client.invalidateQueries({ queryKey: playerAccessKey })
    },
  })
  if (!access.data?.enabled) return children

  const account = access.data.account
  if (!account) return <>
    <p><Link to={accessPath}>Returning player? Use your existing profile</Link></p>
    {children}
  </>

  return <div className="form-preview">
    {account.identity_id ? <>
      <p>Entering as <strong>{account.nickname || account.display_name}</strong>.</p>
      {!enter.isSuccess ? <button type="button" disabled={enter.isPending || logout.isPending} onClick={() => enter.mutate()}>
        {enter.isPending ? 'Registering...' : action}
      </button> : <div role="status"><StateNotice message={successMessage(enter.data.preferred_name)} compact /></div>}
      {enter.error ? <StateNotice tone="error" message={readError(enter.error)} compact /> : null}
    </> : <>
      <p>{account.claim_status === 'pending'
        ? 'An admin needs to approve your profile claim before you can enter with it.'
        : 'Claim your existing player profile to use it for registration.'}</p>
      <Link to={accessPath}>Manage player profile</Link>
    </>}
    <div className="toolbar-actions">
      <button type="button" className="secondary-button" disabled={enter.isPending || logout.isPending}
        onClick={() => logout.mutate()}>Sign out of player profile</button>
    </div>
    {logout.error ? <StateNotice tone="error" message={readError(logout.error)} compact /> : null}
  </div>
}
