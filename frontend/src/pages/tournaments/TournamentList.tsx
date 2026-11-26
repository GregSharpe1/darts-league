import { Link } from 'react-router-dom'
import type { TournamentSummary } from '../../lib/tournamentsApi'
import { TournamentDate } from './TournamentDate'

export function TournamentList({ events, admin = false, cards = false }: { events: TournamentSummary[]; admin?: boolean; cards?: boolean }) {
  if (events.length === 0) return <p>No tournaments yet.</p>

  return (
    <ul className={cards ? 'tournament-directory-cards' : 'check-list'}>
      {events.map((event) => {
        const eventPath = `${admin ? '/admin' : ''}/tournaments/${event.slug}`
        const acceptsEntries = event.registration_open && !event.started
        return (
          <li key={event.slug} className={cards && acceptsEntries ? 'tournament-accepting-entries' : undefined}>
            <div className="tournament-list-label">
              {cards ? <span className="tournament-directory-status">{event.started ? 'Tournament started' : acceptsEntries ? 'Registration open' : 'Registration closed'}</span> : null}
              <Link className="tournament-link" to={eventPath}>{event.name}</Link>
              <div className="fixture-meta">
                <TournamentDate startsOn={event.starts_on} />
                {!cards ? ` | ${event.started ? 'Tournament started' : acceptsEntries ? 'Registration open' : 'Registration closed'}` : null}
              </div>
            </div>
            {!admin && acceptsEntries ? (
              <Link className="cta-link tournament-signup-action" to={eventPath} aria-label={`Sign up for ${event.name}`}>Sign up</Link>
            ) : cards && event.started ? <Link className="button-link tournament-bracket-action" to={eventPath} aria-label={`View bracket for ${event.name}`}>View bracket</Link> : null}
          </li>
        )
      })}
    </ul>
  )
}
