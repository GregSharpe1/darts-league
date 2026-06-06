import { NavLink } from 'react-router-dom'
import { useDivisions, useSeasonSummary } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'

export function HomePage() {
  const seasonQuery = useSeasonSummary()
  const divisionsQuery = useDivisions()

  return (
    <>
      <section className="hero-panel">
        <div className="hero-copy">
          <span className="eyebrow">One league, many divisions</span>
          <h1>Pick a division and follow the board.</h1>
          <p className="fixture-meta">{seasonQuery.data?.name ?? 'Active season'}</p>
          <p>
            Registration stays global, admins place players manually, and each division runs its own fixtures,
            standings, and weekly reveal schedule.
          </p>
          <div className="hero-actions">
            {seasonQuery.data?.registration_open ? <NavLink to="/register">Join the waitlist</NavLink> : null}
          </div>
        </div>

        <div className="hero-side">
          <article className="metric-card">
            <span className="section-eyebrow">League pulse</span>
            <strong>{seasonQuery.data?.player_count ?? '-'}</strong>
            <p>{seasonQuery.data?.registration_open ? 'registered players waiting for assignment.' : `season live across ${seasonQuery.data?.division_count ?? 0} divisions.`}</p>
          </article>
          <article className="metric-card">
            <span className="section-eyebrow">Assignments</span>
            <strong>{seasonQuery.data?.assigned_count ?? '-'}</strong>
            <p>{seasonQuery.data?.waitlist_count ?? 0} player{seasonQuery.data?.waitlist_count === 1 ? '' : 's'} still waitlisted.</p>
          </article>
        </div>
      </section>

      <section className="content-panel">
        <div className="card-header">
          <div className="card-copy">
            <span className="section-eyebrow">Division list</span>
            <h2>Public division boards</h2>
          </div>
          <span className="status-pill live">Monday 09:00 unlocks</span>
        </div>
        {divisionsQuery.isLoading ? <StateNotice message="Loading divisions..." /> : null}
        {divisionsQuery.error ? <StateNotice tone="error" message={readError(divisionsQuery.error)} /> : null}
        {!divisionsQuery.isLoading && !divisionsQuery.error && divisionsQuery.data?.length === 0 ? (
          <StateNotice message="No divisions have been created yet. Admins can provision them before the season starts." />
        ) : null}
        {divisionsQuery.data && divisionsQuery.data.length > 0 ? (
          <div className="division-board-grid" aria-label="Divisions">
            {divisionsQuery.data.map((division) => (
              <article className="week-card division-board-card" key={division.id}>
                <div className="division-board-copy">
                  <header>
                    <div>
                      <h2>{division.name}</h2>
                    </div>
                  </header>
                  <p>Fixtures, standings, and results stay isolated on this board.</p>
                </div>
                <div className="division-board-actions">
                  <div className="hero-actions">
                    <NavLink to={`/divisions/${division.slug}`}>View fixtures</NavLink>
                    <NavLink to={`/divisions/${division.slug}/standings`}>View standings</NavLink>
                  </div>
                </div>
              </article>
            ))}
          </div>
        ) : null}
      </section>
    </>
  )
}
