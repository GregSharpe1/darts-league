import { useState } from 'react'
import type { FormEvent } from 'react'
import { useSeasonSummary, useRegisterPlayer } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'

function nameError(value: string, label: string, limit: number) {
  if (/\p{Cc}/u.test(value)) return `${label} must not contain control characters.`
  // Unicode White_Space matches Go strings.Fields; JS trim also removes U+FEFF.
  const normalized = value.split(/\p{White_Space}+/u).filter(Boolean).join(' ')
  if (label === 'Display name' && !normalized) return 'Display name is required.'
  if (Array.from(normalized).length > limit) return `${label} must be at most ${limit} characters.`
  return ''
}

export function RegisterPage() {
  const seasonQuery = useSeasonSummary()
  const registerMutation = useRegisterPlayer()
  const [displayName, setDisplayName] = useState('')
  const [nickname, setNickname] = useState('')
  const [successMessage, setSuccessMessage] = useState('')
  const [submitted, setSubmitted] = useState(false)
  const displayNameError = submitted ? nameError(displayName, 'Display name', 60) : ''
  const nicknameError = submitted ? nameError(nickname, 'Nickname', 30) : ''

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSuccessMessage('')
    setSubmitted(true)
    if (nameError(displayName, 'Display name', 60) || nameError(nickname, 'Nickname', 30)) return
    try {
      const player = await registerMutation.mutateAsync({ display_name: displayName, nickname })
      setSuccessMessage(`${player.preferred_name} is registered and waiting for division assignment.`)
      setDisplayName('')
      setNickname('')
      setSubmitted(false)
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
        <p>Registration stays open until the admin starts the season. Display names must be unique within this season and nicknames remain optional.</p>
      </section>

      {seasonQuery.data && !seasonQuery.data.registration_open ? (
        <section className="register-grid">
          <article className="register-card">
            <h2>Registration closed</h2>
            <p>{seasonQuery.data.status === 'completed' ? 'This league is completed. Registration will reopen when the admin creates the next season.' : 'The active season has already started, so new entries are locked out until the next registration window opens.'}</p>
            <StateNotice message={seasonQuery.data.status === 'completed' ? 'Final results and standings remain available on the division boards.' : 'Check fixtures and standings to follow the season in progress.'} compact />
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
            <p>Register once, then wait for admin placement into a division or onto the waitlist.</p>
            <form className="form-preview" onSubmit={handleSubmit}>
              <div className="field">
                <label htmlFor="display-name">Display name</label>
                <input id="display-name" name="display-name" required aria-invalid={Boolean(displayNameError)} aria-describedby="display-name-help display-name-error" value={displayName} onChange={(event) => setDisplayName(event.target.value)} placeholder="Luke Humphries" disabled={!seasonQuery.data?.registration_open || registerMutation.isPending} />
                <span id="display-name-help" className="fixture-meta">Required. Up to 60 characters.</span>
                <span id="display-name-error" role="alert">{displayNameError}</span>
              </div>
              <div className="field">
                <label htmlFor="nickname">Nickname</label>
                <input id="nickname" name="nickname" aria-invalid={Boolean(nicknameError)} aria-describedby="nickname-help nickname-error" value={nickname} onChange={(event) => setNickname(event.target.value)} placeholder="The Freeze" disabled={!seasonQuery.data?.registration_open || registerMutation.isPending} />
                <span id="nickname-help" className="fixture-meta">Optional. Up to 30 characters.</span>
                <span id="nickname-error" role="alert">{nicknameError}</span>
              </div>
              <button type="submit" disabled={!seasonQuery.data?.registration_open || registerMutation.isPending}>{registerMutation.isPending ? 'Registering...' : 'Register for the league'}</button>
            </form>
            {registerMutation.error ? <StateNotice tone="error" message={readError(registerMutation.error)} compact /> : null}
            {successMessage ? <StateNotice message={successMessage} compact /> : null}
          </article>

          <article className="register-card">
            <h2>What happens next</h2>
            <ul className="check-list">
              <li><strong>Before start</strong><span className="fixture-meta">Admin reviews the roster and assigns players to divisions manually.</span></li>
              <li><strong>On start</strong><span className="fixture-meta">Each division generates its own fixtures once and registration closes.</span></li>
              <li><strong>Each week</strong><span className="fixture-meta">Match cards unlock every Monday at 09:00.</span></li>
              <li><strong>Current roster</strong><span className="fixture-meta">{seasonQuery.data?.player_count ?? 0} players registered, {seasonQuery.data?.waitlist_count ?? 0} still waitlisted.</span></li>
            </ul>
          </article>
        </section>
      )}
    </>
  )
}
