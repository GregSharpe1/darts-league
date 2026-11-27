import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { StateNotice } from '../../components/StateNotice'
import { request } from '../../lib/api'
import { playerAccessKey, usePlayerAccess } from '../../lib/playerAccess'
import type { ClaimProfile } from '../../lib/playerAccess'
import { readError } from '../../lib/utils'
import '../players/PlayerPage.css'

function signupPath(value: string | null) {
  return value && (value === '/register' || /^\/tournaments\/[\p{L}\p{N}\p{M}%-]+$/u.test(value)) ? value : '/register'
}

export function PlayerAccessPage() {
  const location = useLocation()
  const navigate = useNavigate()
  const client = useQueryClient()
  const next = signupPath(new URLSearchParams(location.search).get('next'))
  const token = new URLSearchParams(location.hash.slice(1)).get('token')
  const access = usePlayerAccess()
  const account = access.data?.account
  const canClaim = Boolean(account && !account.identity_id && account.claim_status !== 'pending')
  const profiles = useQuery({
    queryKey: ['player-access-claim-profiles'],
    queryFn: ({ signal }) => request<ClaimProfile[]>('/api/players', { signal }),
    enabled: canClaim,
    retry: false,
  })
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [profile, setProfile] = useState('')
  const needle = name.trim().toLocaleLowerCase('en-GB')
  const candidates = needle ? (profiles.data ?? []).filter(p =>
    [p.name, p.display_name].some(label => label?.toLocaleLowerCase('en-GB').includes(needle))) : []

  const send = useMutation({
    mutationFn: () => request<void>('/api/player-access/link', { method: 'POST', body: JSON.stringify({ email, next }) }),
  })
  const verify = useMutation({
    mutationFn: () => request<{ next: string }>('/api/player-access/verify', { method: 'POST', body: JSON.stringify({ token }) }),
    onSuccess: async data => {
      navigate(`/player-access?${new URLSearchParams({ next: signupPath(data.next) })}`, { replace: true })
      await client.invalidateQueries({ queryKey: playerAccessKey })
    },
  })
  const claim = useMutation({
    mutationFn: () => request<void>('/api/player-access/claims', { method: 'POST', body: JSON.stringify({ identity_id: Number(profile) }) }),
    onSuccess: () => client.invalidateQueries({ queryKey: playerAccessKey }),
  })
  const signOut = useMutation({
    mutationFn: () => request<void>('/api/player-access/sign-out', { method: 'POST' }),
    onSuccess: () => client.invalidateQueries({ queryKey: playerAccessKey }),
  })
  const error = send.error || verify.error || claim.error || access.error || profiles.error || signOut.error

  return <div className="player-page">
    <section className="page-intro"><h1>Returning player sign-in</h1></section>
    <section className="content-panel form-preview" aria-label="Player sign-in">
      {access.isPending ? <p role="status">Loading player sign-in...</p> :
        !access.data?.enabled ? <StateNotice message="Player sign-in is unavailable." /> :
        token && !verify.isSuccess ? <>
          <p>Confirm sign-in to use the profile associated with this email link.</p>
          <button type="button" disabled={verify.isPending} onClick={() => verify.mutate()}>Confirm player sign-in</button>
          <Link to={`/player-access?${new URLSearchParams({ next })}`}>Request a new email link</Link>
        </> : verify.isPending ? <p role="status">Signing in...</p> :
        !account ? <form className="form-preview" onSubmit={event => { event.preventDefault(); send.mutate() }}>
          <p>We will email you a one-time link. If this is your first sign-in, you can request access to your existing profile after verifying your email.</p>
          <div className="field">
            <label htmlFor="player-email">Email address</label>
            <input id="player-email" type="email" autoComplete="email" maxLength={254} required value={email} disabled={send.isPending}
              onChange={event => { setEmail(event.target.value); send.reset() }} />
          </div>
          <button type="submit" disabled={send.isPending}>{send.isPending ? 'Sending...' : 'Email me a sign-in link'}</button>
          {send.isSuccess ? <div role="status"><StateNotice message="Check your inbox for the sign-in link." /></div> : null}
        </form> : account.identity_id ? <>
          <h2>{account.nickname || account.display_name}</h2>
          <p>Your player profile is ready to use.</p>
        </> : account.claim_status === 'pending' ? <>
          <h2>Profile claim pending</h2>
          <p>An admin will review your claim. Return to this page to check its status.</p>
          <div className="toolbar-actions">
            <button className="secondary-button" type="button" disabled={access.isFetching}
              onClick={() => { void access.refetch() }}>Check claim status</button>
          </div>
        </> : <form className="form-preview" onSubmit={event => { event.preventDefault(); claim.mutate() }}>
          <h2>Find your existing player profile</h2>
          {account.claim_status === 'rejected' ? <StateNotice tone="error" message="Your last claim was declined. Contact an admin if you need help." /> : null}
          <p>Signed in as {account.email}. An admin will confirm that the selected profile belongs to you.</p>
          <div className="field">
            <label htmlFor="claim-name">Player name or nickname</label>
            <input id="claim-name" type="search" value={name} onChange={event => { setName(event.target.value); setProfile('') }} />
          </div>
          {needle && !profiles.isPending && candidates.length === 0 ? <p role="status">No matching public profiles. Check your name or ask an admin for help.</p> : null}
          {candidates.length > 0 ? <div className="field">
            <label htmlFor="claim-profile">Choose your profile</label>
            <select id="claim-profile" value={profile} onChange={event => setProfile(event.target.value)}>
              <option value="">Select a profile</option>
              {candidates.map(p => <option key={p.identity_id} value={p.identity_id}>{p.name} / {p.competition}</option>)}
            </select>
          </div> : null}
          <button type="submit" disabled={!profile || claim.isPending}>Request profile claim</button>
        </form>}
      {error ? <div role="alert"><StateNotice tone="error" message={readError(error)} /></div> : null}
      <div className="toolbar-actions">
        <Link className="button-link" to={next}>{account?.identity_id ? 'Continue to sign-up' : 'Back to sign-up'}</Link>
        {account && !token ? <button className="secondary-button" type="button" disabled={signOut.isPending} onClick={() => signOut.mutate()}>Sign out</button> : null}
      </div>
    </section>
  </div>
}
