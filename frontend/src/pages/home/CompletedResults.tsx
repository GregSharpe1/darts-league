import type { PublicFixtureWeek } from '../../lib/api'
import { formatAverage } from '../../lib/api'

export function CompletedResults({ weeks }: { readonly weeks: readonly PublicFixtureWeek[] }) {
  return (
    <section className="content-panel" aria-label="Final results">
      <h2>Final results</h2>
      {weeks.filter((week) => week.status === 'unlocked').map((week) => (
        <article className="week-card" key={week.week_number}>
          <h3>Week {week.week_number}</h3>
          <ul className="match-list">
            {week.fixtures.map((fixture) => fixture.result ? (
              <li key={fixture.id}>
                <div>
                  <strong>{fixture.player_one} {fixture.result.player_one_legs}-{fixture.result.player_two_legs} {fixture.player_two}</strong>
                  <div className="fixture-meta">Averages: {formatAverage(fixture.result.player_one_average) || '-'} / {formatAverage(fixture.result.player_two_average) || '-'}</div>
                </div>
              </li>
            ) : null)}
          </ul>
        </article>
      ))}
    </section>
  )
}
