import { useState } from 'react'
import type { ReactNode } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Link, useLocation, useSearchParams } from 'react-router-dom'
import { StateNotice } from '../../components/StateNotice'
import { request } from '../../lib/api'
import { filterPlayerProfiles, usePlayerDirectory } from '../../lib/playerDirectory'
import { readError } from '../../lib/utils'

type ReturningProps = {
  action: string
  endpoint: string
  successMessage: (name: string) => string
}

export function ReturningPlayerSignup({ children, ...props }: ReturningProps & { children: ReactNode }) {
  const { pathname } = useLocation()
  const [params] = useSearchParams()
  return params.get('returning') === '1' ? <>
    <ReturningPlayerForm {...props} />
    <p className="competition-signup-alternate"><Link className="text-link" to={pathname}>Sign up as a new player</Link></p>
  </> : <>
    <p className="competition-signup-alternate"><Link className="text-link" to="?returning=1">Returning player? Select your profile</Link></p>
    {children}
  </>
}

function ReturningPlayerForm({ action, endpoint, successMessage }: ReturningProps) {
  const [search, setSearch] = useState('')
  const [identity, setIdentity] = useState('')
  const directory = usePlayerDirectory()
  const enter = useMutation({
    mutationFn: () => request<{ preferred_name: string }>(endpoint, {
      method: 'POST', body: JSON.stringify({ identity_id: Number(identity) }),
    }),
  })

  if (directory.isPending) return <p role="status">Loading previous players...</p>
  if (directory.isError) return <div className="toolbar-actions">
    <StateNotice tone="error" message="The player list could not be loaded." compact />
    <button type="button" className="secondary-button" onClick={() => { void directory.refetch() }}>Try again</button>
  </div>
  if (directory.data.length === 0) return <p>No previous players are available to select yet.</p>
  if (enter.isSuccess) return <div role="status">
    <StateNotice message={successMessage(enter.data.preferred_name)} compact />
    <p>An admin will review the link to your previous results.</p>
  </div>

  const profiles = filterPlayerProfiles(directory.data, search)
  return <form className="form-preview" onSubmit={event => { event.preventDefault(); enter.mutate() }}>
    <p>Choose the profile you played under. An admin will confirm the link to your previous results.</p>
    <div className="field">
      <label htmlFor="returning-search">Search previous players</label>
      <input id="returning-search" type="search" value={search} disabled={enter.isPending} onChange={event => {
        setSearch(event.target.value)
        setIdentity('')
        enter.reset()
      }} />
    </div>
    <div className="field">
      <label htmlFor="returning-profile">Previous player profile</label>
      <select id="returning-profile" required value={identity} disabled={enter.isPending} onChange={event => {
        setIdentity(event.target.value)
        enter.reset()
      }}>
        <option value="">{profiles.length ? 'Choose a player' : 'No matching players'}</option>
        {profiles.map(profile => <option key={profile.identity_id} value={profile.identity_id}>{profile.name} / {profile.competition}</option>)}
      </select>
    </div>
    <button type="submit" disabled={!identity || enter.isPending}>{enter.isPending ? 'Registering...' : action}</button>
    {enter.error ? <div role="alert"><StateNotice tone="error" message={readError(enter.error)} compact /></div> : null}
  </form>
}
