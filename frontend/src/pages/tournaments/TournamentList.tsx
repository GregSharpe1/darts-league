import { Link } from 'react-router-dom'
import type { TournamentSummary } from '../../lib/tournamentsApi'
import { TournamentDate } from './TournamentDate'

export function TournamentList({ events, admin = false }: { events: TournamentSummary[]; admin?: boolean }) {
  if (events.length === 0) return <p>No tournaments yet.</p>

  return (
    <ul className="check-list">
      {events.map((event) => {
        const eventPath = `${admin ? '/admin' : ''}/tournaments/${event.slug}`
        const acceptsEntries = event.registration_open && !event.started
        return (
          <li key={event.slug}>
            <div className="tournament-list-label">
              <Link className="tournament-link" to={eventPath}>{event.name}</Link>
              <div className="fixture-meta">
                <TournamentDate startsOn={event.starts_on} /> | {event.started ? 'Tournament started' : acceptsEntries ? 'Registration open' : 'Registration closed'}
              </div>
            </div>
            {!admin && acceptsEntries ? (
              <Link className="cta-link tournament-signup-action" to={eventPath} aria-label={`Sign up for ${event.name}`}>Sign up</Link>
            ) : null}
          </li>
        )
      })}
    </ul>
  )
}
