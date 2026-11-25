import { useParams } from 'react-router-dom'
import { StateNotice } from '../../components/StateNotice'
import { useTournament, useTournamentSignup } from '../../lib/tournamentsApi'
import { readError } from '../../lib/utils'
import { PlayerSignupForm } from '../register/PlayerSignupForm'
import { TournamentDate } from './TournamentDate'

export function TournamentPage() {
  const { slug = '' } = useParams()
  return <TournamentRegistration key={slug} slug={slug} />
}

function TournamentRegistration({ slug }: { slug: string }) {
  const eventQuery = useTournament(slug)
  const signup = useTournamentSignup(slug)
  const event = eventQuery.data

  return (
    <>
      <section className="page-intro">
        <span className="eyebrow">Darts tournament</span>
        <h1>{event?.name ?? 'Tournament registration'}</h1>
        {event ? <p>Starts <TournamentDate startsOn={event.starts_on} /></p> : null}
      </section>
      {eventQuery.isPending ? <StateNotice message="Loading tournament..." /> : null}
      {eventQuery.error ? <StateNotice tone="error" message={readError(eventQuery.error)} /> : null}
      {event ? (
        <section className="register-grid">
          <article className="register-card">
            <h2>{event.registration_open ? 'Player sign-up' : 'Registration closed'}</h2>
            {event.registration_open ? (
              <PlayerSignupForm
                action="Enter tournament"
                busy={signup.isPending}
                error={signup.error}
                onRegister={async (entry) => {
                  const { preferred_name } = await signup.mutateAsync(entry)
                  return `${preferred_name} is registered for ${event.name}.`
                }}
              />
            ) : <p>This tournament is currently closed to new entries.</p>}
          </article>
        </section>
      ) : null}
    </>
  )
}
