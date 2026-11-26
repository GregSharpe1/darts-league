import { NavLink } from 'react-router-dom'
import { formatWhen, useDivisions, useSeasonSummary } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'

export function DivisionsPage() {
  const seasonQuery = useSeasonSummary()
  const divisionsQuery = useDivisions()
  const season = seasonQuery.data
  const leagueFinished = season?.status === 'completed' || (season?.status === 'started' && season.total_fixtures > 0 && season.remaining_fixtures === 0)

  return (
    <>
      <section className="hero-panel competition-hero competition-hero--league">
        <div className="hero-copy">
          <h1>League divisions</h1>
          <p className="fixture-meta">{seasonQuery.data?.name ?? 'Active season'}</p>
          <p>
            Admins assign players to divisions after they register for the league. Your division&apos;s board shows its
            fixtures, match results, and standings.
          </p>
          <div className="hero-actions">
            {seasonQuery.data?.registration_open ? <NavLink to="/register">Register</NavLink> : null}
          </div>
        </div>

        <div className="hero-side">
          <article className="metric-card">
            <span className="section-eyebrow">Registered players</span>
            <strong>{seasonQuery.data?.player_count ?? '-'}</strong>
            <p>{season?.status === 'completed' ? 'League completed. Final results remain available.' : season?.registration_open ? 'Registration is open.' : season?.status === 'started' ? `season live across ${season.division_count} divisions.` : null}</p>
          </article>
          <article className="metric-card">
            <span className="section-eyebrow">Players assigned</span>
            <strong>{seasonQuery.data?.assigned_count ?? '-'}</strong>
            <p>{seasonQuery.data?.waitlist_count ?? 0} player{seasonQuery.data?.waitlist_count === 1 ? '' : 's'} still waitlisted.</p>
          </article>
        </div>
      </section>

      {season ? (
        <section className="content-panel league-deadline" aria-labelledby="league-deadline-heading">
          <h2 id="league-deadline-heading">Final week and catch-up deadline</h2>
          {season.final_week_starts_at && season.fixtures_due_at ? (
            <dl className="league-deadline-dates">
              <div>
                <dt>Week {season.week_count} starts</dt>
                <dd><time dateTime={season.final_week_starts_at}>{formatWhen(season.final_week_starts_at, { year: 'numeric' })} UK time</time></dd>
              </div>
              <div>
                <dt>Finish outstanding fixtures by</dt>
                <dd><time dateTime={season.fixtures_due_at}>{formatWhen(season.fixtures_due_at, { year: 'numeric' })} UK time</time></dd>
              </div>
            </dl>
          ) : <p>Dates will appear once the league schedule is set.</p>}
          <p>Players have one extra week after the final scheduled week ends to finish any outstanding fixtures.</p>
          <p>
            League rules call for Autodarts bots to represent both players in each fixture still unplayed at the deadline.
            A player&apos;s starting bot level equals the number of scheduled league weeks, capped at level 11.
            Each player drops one level per missed fixture, down to level 1.
          </p>
        </section>
      ) : null}

      <section className="content-panel league-divisions">
        <div className="card-header">
          <div className="card-copy">
            <h2>{leagueFinished ? "View the previous league's scores here" : 'Public division boards'}</h2>
          </div>
          {!leagueFinished ? <span className="status-pill live">Monday 09:00 unlocks</span> : null}
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
