import type { PublicFixtureWeek } from '../../lib/api'
import { formatAverage } from '../../lib/api'
import { Link } from 'react-router-dom'

export function CompletedResults({ weeks, title = 'Final results' }: { readonly weeks: readonly PublicFixtureWeek[]; readonly title?: string }) {
  return (
    <section className="content-panel" aria-label={title}>
      <h2>{title}</h2>
      {weeks.filter((week) => week.status === 'unlocked' && week.fixtures.some(fixture => fixture.result)).map((week) => (
        <article className="week-card" key={week.week_number}>
          <h3>Week {week.week_number}</h3>
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
        </article>
      ))}
    </section>
  )
}
