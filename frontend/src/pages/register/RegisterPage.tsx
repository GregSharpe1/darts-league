import { useState } from 'react'
import type { FormEvent } from 'react'
import { useSeasonSummary, useRegisterPlayer } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'

export function RegisterPage() {
  const seasonQuery = useSeasonSummary()
  const registerMutation = useRegisterPlayer()
  const [displayName, setDisplayName] = useState('')
  const [nickname, setNickname] = useState('')
  const [successMessage, setSuccessMessage] = useState('')

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSuccessMessage('')
    try {
      const player = await registerMutation.mutateAsync({ display_name: displayName, nickname })
      setSuccessMessage(`${player.preferred_name} is in for the active season.`)
      setDisplayName('')
      setNickname('')
    } catch {
      // handled by mutation error state
    }
  }

  return (
    <>
      <section className="page-intro">
        <span className="eyebrow">Public registration</span>
        <h1>Enter the league</h1>
        <p className="fixture-meta">{seasonQuery.data?.name ?? 'Active season'}</p>
        <p>Registration stays open until the admin starts the season. Display names are unique per season and nicknames stay optional.</p>
      </section>

      {seasonQuery.data && !seasonQuery.data.registration_open ? (
        <section className="register-grid">
          <article className="register-card">
            <h2>Registration closed</h2>
            <p>The active season has already started, so new entries are locked out until the next registration window opens.</p>
            <StateNotice message="Check fixtures and standings to follow the season in progress." compact />
          </article>

          <article className="register-card">
            <h2>Where to go instead</h2>
            <ul className="check-list">
              <li><strong>Fixtures</strong><span className="fixture-meta">See who is due to play this week.</span></li>
              <li><strong>Standings</strong><span className="fixture-meta">Track points, legs for, and league position.</span></li>
              <li><strong>Next season</strong><span className="fixture-meta">Registration will reopen when a new season is created.</span></li>
            </ul>
          </article>
        </section>
      ) : (
        <section className="register-grid">
          <article className="register-card">
            <h2>Player sign-up</h2>
            <p>Keep it fast for MVP: one required name, one optional darts nickname.</p>
            <form className="form-preview" onSubmit={handleSubmit}>
              <div className="field">
                <label htmlFor="display-name">Display name</label>
                <input id="display-name" name="display-name" value={displayName} onChange={(event) => setDisplayName(event.target.value)} placeholder="Luke Humphries" disabled={!seasonQuery.data?.registration_open || registerMutation.isPending} />
              </div>
              <div className="field">
                <label htmlFor="nickname">Nickname</label>
                <input id="nickname" name="nickname" value={nickname} onChange={(event) => setNickname(event.target.value)} placeholder="The Freeze" disabled={!seasonQuery.data?.registration_open || registerMutation.isPending} />
              </div>
              <button type="submit" disabled={!seasonQuery.data?.registration_open || registerMutation.isPending}>{registerMutation.isPending ? 'Registering...' : 'Register for this season'}</button>
            </form>
            {registerMutation.error ? <StateNotice tone="error" message={readError(registerMutation.error)} compact /> : null}
            {successMessage ? <StateNotice message={successMessage} compact /> : null}
            {seasonQuery.data && !seasonQuery.data.registration_open ? <StateNotice message="Registration is now closed for the active season." compact /> : null}
          </article>

          <article className="register-card">
            <h2>What happens next</h2>
            <ul className="check-list">
              <li><strong>Before start</strong><span className="fixture-meta">Admin can review and trim the roster.</span></li>
              <li><strong>On start</strong><span className="fixture-meta">Fixtures generate once and registration closes.</span></li>
              <li><strong>Each week</strong><span className="fixture-meta">Match cards unlock every Monday at 09:00.</span></li>
              <li><strong>Current roster</strong><span className="fixture-meta">{seasonQuery.data?.player_count ?? 0} players in the active season.</span></li>
            </ul>
          </article>
        </section>
      )}
    </>
  )
}
