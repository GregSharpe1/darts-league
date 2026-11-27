import { Link, useParams } from 'react-router-dom'
import { StateNotice } from '../../components/StateNotice'
import { ApiError } from '../../lib/api'
import { type Tournament, useTournament, useTournamentSignup } from '../../lib/tournamentsApi'
import { readError } from '../../lib/utils'
import { ReturningPlayerSignup } from '../register/ReturningPlayerSignup'
import { PlayerSignupForm } from '../register/PlayerSignupForm'
import { TournamentDate } from './TournamentDate'
import { TournamentBracket } from './TournamentBracket'
import { TournamentFormat } from './TournamentFormat'
import '../register/competitionSignup.css'

export function TournamentPage() {
  const { slug = '' } = useParams()
  return <TournamentRegistration key={slug} slug={slug} />
}

function TournamentRegistration({ slug }: { slug: string }) {
  const eventQuery = useTournament(slug)
  const signup = useTournamentSignup(slug)
  const notFound = eventQuery.error instanceof ApiError && eventQuery.error.status === 404
  const event = notFound ? undefined : eventQuery.data
  const error = eventQuery.error ? <StateNotice tone="error" message={readError(eventQuery.error)} /> : null

  if (!event) {
    return (
      <>
        <section className="page-intro">
          <h1>Tournament registration</h1>
          <Link className="tournament-link" to="/tournaments">All tournaments</Link>
        </section>
        {eventQuery.isPending ? <StateNotice message="Loading tournament..." /> : error}
      </>
    )
  }

  if (event.bracket) {
    return (
      <>
        <Link className="competition-signup-back" to="/tournaments"><span aria-hidden="true">&larr;</span> All tournaments</Link>
        <section className="hero-panel competition-hero competition-hero--tournament">
          <div className="hero-copy">
            <span className="competition-kicker">Knockout tournament</span>
            <h1>{event.name}</h1>
          </div>
          <div className="hero-side">
            <div className="metric-card competition-event-fact">
              <p className="competition-kicker">Starts</p>
              <TournamentDate date={event.starts_on} />
              {event.ends_on ? <><p className="competition-kicker">Play by</p><TournamentDate date={event.ends_on} /></> : null}
            </div>
            <section className="metric-card competition-event-fact" aria-labelledby="tournament-match-format">
              <h2 className="competition-kicker" id="tournament-match-format">Match format</h2>
              <TournamentFormat format={event} />
            </section>
          </div>
        </section>
        {error}
        <TournamentBracket rounds={event.bracket} />
        <TournamentRules event={event} />
      </>
    )
  }

  return (
    <div className="competition-signup competition-signup--tournament">
      <Link className="competition-signup-back" to="/tournaments">
        <span aria-hidden="true">&larr;</span> All tournaments
      </Link>
      {error}
      <div className="competition-signup-card">
        <section className="competition-signup-overview" aria-labelledby="tournament-event-name">
          <h1 id="tournament-event-name">{event.name}</h1>
          <div className="competition-signup-facts">
            <div>
              <p className="competition-signup-label">Starts</p>
              <TournamentDate date={event.starts_on} />
              {event.ends_on ? <><p className="competition-signup-label">Play by</p><TournamentDate date={event.ends_on} /></> : null}
            </div>
            <section aria-labelledby="tournament-match-format">
              <h2 className="competition-signup-label" id="tournament-match-format">Match format</h2>
              <TournamentFormat format={event} />
            </section>
          </div>
        </section>

        <section className="competition-entry" aria-labelledby="competition-entry-title">
          {event.registration_open ? <p className="competition-entry-status">Registration open</p> : null}
          <h2 id="competition-entry-title">{event.registration_open ? 'Player sign-up' : 'Registration closed'}</h2>
          {event.registration_open ? (
            <ReturningPlayerSignup action="Enter tournament"
              endpoint={`/api/tournaments/${encodeURIComponent(event.slug)}/returning-entry`}
              successMessage={name => `${name} is registered for ${event.name}.`}>
              <PlayerSignupForm
                action="Enter tournament"
                busy={signup.isPending}
                error={signup.error}
                onRegister={async (entry) => {
                  const { preferred_name } = await signup.mutateAsync(entry)
                  return `${preferred_name} is registered for ${event.name}.`
                }}
              />
            </ReturningPlayerSignup>
          ) : <Link className="button-link" to="/tournaments">Find another tournament</Link>}
        </section>
      </div>
      <TournamentRules event={event} />
    </div>
  )
}

function TournamentRules({ event }: { event: Tournament }) {
  return (
    <section className="content-panel tournament-rules" aria-labelledby="tournament-rules-heading">
      <h2 id="tournament-rules-heading">Tournament rules</h2>
      <ul className="tournament-rules-list">
        <li>This is a single-elimination knockout. The winner advances and the loser is eliminated. A bye advances a player automatically.</li>
        <li>Each match is {event.game_variant}, first to {event.legs_to_win} {event.legs_to_win === 1 ? 'leg' : 'legs'}.</li>
        {event.ends_on ? (
          <li>Finish your matches by the end of <TournamentDate date={event.ends_on} /> (UK time). The organiser arranges Autodarts simulations for the remaining matches, with a bot representing each player.</li>
        ) : null}
      </ul>
    </section>
  )
}
