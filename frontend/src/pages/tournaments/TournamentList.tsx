import { Link } from 'react-router-dom'
import type { TournamentSummary } from '../../lib/tournamentsApi'
import { TournamentDate } from './TournamentDate'

export function TournamentList({ events, admin = false, cards = false }: { events: TournamentSummary[]; admin?: boolean; cards?: boolean }) {
  if (events.length === 0) return <p>No tournaments yet.</p>

  return (
    <ul className={cards ? 'division-board-grid tournament-directory-cards' : 'check-list'}>
      {events.map((event) => {
        const eventPath = `${admin ? '/admin' : ''}/tournaments/${event.slug}`
        const acceptsEntries = event.registration_open && !event.started
        const status = event.started ? 'Tournament started' : acceptsEntries ? 'Registration open' : 'Registration closed'
        const action = !admin && acceptsEntries ? (
          <Link className="cta-link tournament-signup-action" to={eventPath} aria-label={`Sign up for ${event.name}`}>Sign up</Link>
        ) : cards && event.started ? (
          <Link className="cta-link" to={eventPath} aria-label={`View bracket for ${event.name}`}>View bracket</Link>
        ) : null

        return cards ? (
          <li key={event.slug} className="week-card division-board-card">
            <div className="division-board-copy tournament-list-label">
              <h3><Link className="tournament-link" to={eventPath}>{event.name}</Link></h3>
              <p className="fixture-meta">
                Starts <TournamentDate date={event.starts_on} />
                {event.ends_on ? <> &middot; Play by <TournamentDate date={event.ends_on} /></> : null}
              </p>
              <div><span className={acceptsEntries ? 'status-pill live' : 'status-pill'}>{status}</span></div>
            </div>
            {action ? <div className="division-board-actions">{action}</div> : null}
          </li>
        ) : (
          <li key={event.slug}>
            <div className="tournament-list-label">
              <Link className="tournament-link" to={eventPath}>{event.name}</Link>
              <div className="fixture-meta"><TournamentDate date={event.starts_on} /> | {status}</div>
            </div>
            {action}
          </li>
        )
      })}
    </ul>
  )
}
