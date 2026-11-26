import { useState } from 'react'
import { Link } from 'react-router-dom'
import type { PublicFixtureWeek } from '../../lib/api'
import { formatAverage } from '../../lib/api'

export function CompletedResults({ weeks, title = 'Final results', initiallyOpen = false }: {
  readonly weeks: readonly PublicFixtureWeek[]
  readonly title?: string
  readonly initiallyOpen?: boolean
}) {
  return (
    <section className="content-panel league-results" aria-label={title}>
      <h2>{title}</h2>
      <div className="week-switcher">
        {weeks.filter((week) => week.status === 'unlocked' && week.fixtures.some((fixture) => fixture.result)).map((week) => (
          <RecordedWeek key={week.week_number} week={week} initiallyOpen={initiallyOpen} />
        ))}
      </div>
    </section>
  )
}

function RecordedWeek({ week, initiallyOpen }: { readonly week: PublicFixtureWeek; readonly initiallyOpen: boolean }) {
  const [isOpen, setIsOpen] = useState(initiallyOpen)
  return (
    <article className={`week-switcher-item${isOpen ? ' open' : ''}`}>
      <button type="button" className="week-switcher-trigger" aria-expanded={isOpen} onClick={() => setIsOpen((open) => !open)}>
        <span className="week-switcher-copy"><strong>Week {week.week_number}</strong></span>
        <span className="week-switcher-icon" aria-hidden="true">{isOpen ? '−' : '+'}</span>
      </button>
      {isOpen ? (
        <div className="week-switcher-panel">
          <ul className="match-list">
            {week.fixtures.map((fixture) => fixture.result ? (
              <li key={fixture.id}>
                <div>
                  <strong>{fixture.player_one} {fixture.result.player_one_legs}-{fixture.result.player_two_legs} {fixture.player_two}</strong>
                  <div className="fixture-meta">Averages: {formatAverage(fixture.result.player_one_average) || '-'} / {formatAverage(fixture.result.player_two_average) || '-'}</div>
                  {fixture.id > 0 ? <Link className="match-view-link" to={`/matches/${fixture.id}`} aria-label={`View match: ${fixture.player_one} vs ${fixture.player_two}`}>View match</Link> : null}
                </div>
              </li>
            ) : null)}
          </ul>
        </div>
      ) : null}
    </article>
  )
}
