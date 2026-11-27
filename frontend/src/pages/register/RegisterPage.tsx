import { Link } from 'react-router-dom'
import { useSeasonSummary, useRegisterPlayer } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'
import { ReturningPlayerSignup } from './ReturningPlayerSignup'
import { PlayerSignupForm } from './PlayerSignupForm'
import './competitionSignup.css'

export function RegisterPage() {
  const seasonQuery = useSeasonSummary()
  const registerMutation = useRegisterPlayer()
  const season = seasonQuery.data

  return (
    <div className="competition-signup competition-signup--league">
      <Link className="competition-signup-back" to="/divisions">
        <span aria-hidden="true">&larr;</span> League divisions
      </Link>
      <div className="competition-signup-card">
        <section className="competition-signup-overview" aria-labelledby="league-signup-title">
          <h1 id="league-signup-title">Enter the league</h1>
          {season ? (
            <div className="competition-signup-facts">
              <div>
                <p className="competition-signup-label">Season</p>
                <p>{season.name}</p>
              </div>
              <div>
                <p className="competition-signup-label">Match format</p>
                <p>{season.game_variant}, first to {season.legs_to_win} {season.legs_to_win === 1 ? 'leg' : 'legs'}</p>
              </div>
            </div>
          ) : null}
        </section>

        <section className="competition-entry" aria-labelledby="league-entry-title">
          {season?.registration_open ? <p className="competition-entry-status">Registration open</p> : null}
          <h2 id="league-entry-title">{season && !season.registration_open ? 'Registration closed' : 'Player sign-up'}</h2>
          {seasonQuery.error ? <StateNotice tone="error" message={readError(seasonQuery.error)} compact /> : null}
          {seasonQuery.isPending ? <StateNotice message="Loading registration..." compact /> : null}
          {season?.registration_open ? (
            <>
              <p className="competition-entry-description">An admin will assign you to a division after you register.</p>
              <ReturningPlayerSignup action="Register for the league"
                endpoint={`/api/player-access/leagues/${season.id}/entry`}
                successMessage={name => `${name} is registered and waiting for division assignment.`}>
                <PlayerSignupForm
                  action="Register for the league"
                  busy={registerMutation.isPending}
                  error={registerMutation.error}
                  onRegister={async (entry) => {
                    const player = await registerMutation.mutateAsync(entry)
                    return `${player.preferred_name} is registered and waiting for division assignment.`
                  }}
                />
              </ReturningPlayerSignup>
            </>
          ) : season ? (
            <>
              <p className="competition-entry-description">{season.status === 'completed' ? 'This league is completed. Registration will reopen when the admin creates the next season.' : 'The active season has already started. Registration will reopen next season.'}</p>
              <Link className="button-link" to="/divisions">Browse divisions</Link>
            </>
          ) : null}
        </section>
      </div>
    </div>
  )
}
