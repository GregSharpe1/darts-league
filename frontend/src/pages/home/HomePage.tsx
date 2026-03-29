import { useEffect, useMemo, useState } from 'react'
import { NavLink } from 'react-router-dom'
import { useSeasonSummary, usePublicFixtures, formatWhen } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'

export function HomePage() {
  const seasonQuery = useSeasonSummary()
  const fixturesQuery = usePublicFixtures()

  const currentWeek = fixturesQuery.data?.weeks.find((week) => week.week_number === fixturesQuery.data.current_week)
  const futureWeeks = fixturesQuery.data?.weeks.filter((week) => week.status === 'locked') ?? []
  const unplayedUnlockedWeeks = useMemo(
    () =>
      fixturesQuery.data?.weeks
        .filter((week) => week.status === 'unlocked')
        .map((week) => ({
          ...week,
          unplayedFixtures: week.fixtures.filter((fixture) => !fixture.result),
        }))
        .filter((week) => week.unplayedFixtures.length > 0) ?? [],
    [fixturesQuery.data?.weeks],
  )
  const gamesLeftToPlay = unplayedUnlockedWeeks.reduce((total, week) => total + week.unplayedFixtures.length, 0)
  const defaultOpenWeek =
    unplayedUnlockedWeeks.find((week) => week.week_number === fixturesQuery.data?.current_week)?.week_number ??
    unplayedUnlockedWeeks[0]?.week_number ??
    null
  const [openWeekNumber, setOpenWeekNumber] = useState<number | null>(null)

  useEffect(() => {
    setOpenWeekNumber((previousOpenWeek) => {
      if (unplayedUnlockedWeeks.length === 0) {
        return null
      }

      if (previousOpenWeek !== null && unplayedUnlockedWeeks.some((week) => week.week_number === previousOpenWeek)) {
        return previousOpenWeek
      }

      return defaultOpenWeek
    })
  }, [defaultOpenWeek, unplayedUnlockedWeeks])

  return (
    <>
      <section className="hero-panel">
        <div className="hero-copy">
          <span className="eyebrow">Current week unlocked</span>
          <h1>Fixtures with a little theatre.</h1>
          <p className="fixture-meta">{seasonQuery.data?.name ?? 'Active season'}</p>
          <p>
            The public board shows this week in full, keeps future pairings on the radar, and holds the reveal until Monday morning at 09:00 Europe/London.
          </p>
          <div className="hero-actions">
            {seasonQuery.data?.registration_open ? <NavLink to="/register">Join before season start</NavLink> : null}
            <NavLink to="/standings">View points table</NavLink>
          </div>
        </div>

        <div className="hero-side">
          <article className="metric-card">
            <span className="section-eyebrow">League pulse</span>
            <strong>{seasonQuery.data?.player_count ?? '-'}</strong>
            <p>{seasonQuery.data?.registration_open ? 'players registered before the season start action.' : `season live across ${seasonQuery.data?.week_count ?? 0} weeks.`}</p>
          </article>
          {seasonQuery.data && !seasonQuery.data.registration_open ? (
            <article className="metric-card">
              <span className="section-eyebrow">Match format</span>
              <strong>{seasonQuery.data.game_variant}</strong>
              <p>First to {seasonQuery.data.legs_to_win} legs | {seasonQuery.data.games_per_week} game{seasonQuery.data.games_per_week !== 1 ? 's' : ''} per week</p>
            </article>
          ) : null}
          <article className="info-card">
            <div className="card-header">
              <div className="card-copy">
                <h2>Next reveal</h2>
                <p>Future week cards stay visible, with pairings shown and details intentionally obscured.</p>
              </div>
              <span className="status-pill locked">09:00 Monday</span>
            </div>
          </article>
        </div>
      </section>

      <section className="content-panel">
        <div className="card-header">
          <div className="card-copy">
            <span className="section-eyebrow">This week</span>
            <h2>Games left to play</h2>
          </div>
          <span className="status-pill live">{gamesLeftToPlay > 0 ? `${gamesLeftToPlay} live` : currentWeek ? 'Unlocked' : 'Waiting'}</span>
        </div>
        {fixturesQuery.isLoading ? <StateNotice message="Loading the season board..." /> : null}
        {fixturesQuery.error ? <StateNotice tone="error" message={readError(fixturesQuery.error)} /> : null}
        {!fixturesQuery.isLoading && !fixturesQuery.error && !currentWeek ? (
          <StateNotice message="No public week is unlocked yet. Start the season in admin to generate fixtures." />
        ) : null}
        {!fixturesQuery.isLoading && !fixturesQuery.error && currentWeek && gamesLeftToPlay === 0 ? (
          <StateNotice message="Every unlocked fixture has been played so far. Check locked weeks for what is coming next." />
        ) : null}
        {unplayedUnlockedWeeks.length > 0 ? (
          <div className="week-switcher" aria-label="Weeks with matches left to play">
            {unplayedUnlockedWeeks.map((week) => {
              const isOpen = week.week_number === openWeekNumber
              const fixtureCountLabel = `${week.unplayedFixtures.length} match${week.unplayedFixtures.length !== 1 ? 'es' : ''} left`

              return (
                <section className={`week-switcher-item${isOpen ? ' open' : ''}`} key={week.week_number}>
                  <button
                    type="button"
                    className="week-switcher-trigger"
                    aria-expanded={isOpen}
                    onClick={() => setOpenWeekNumber(week.week_number)}
                  >
                    <div className="week-switcher-copy">
                      <span className="section-eyebrow">Unplayed fixtures</span>
                      <strong>Week {week.week_number}</strong>
                      <span className="fixture-meta">
                        {week.week_number === fixturesQuery.data?.current_week ? 'Current week unlocked' : 'Previous week still outstanding'}
                      </span>
                    </div>
                    <div className="week-switcher-meta">
                      <span className={`status-pill ${week.week_number === fixturesQuery.data?.current_week ? 'live' : 'locked'}`}>{fixtureCountLabel}</span>
                      <span className="week-switcher-icon" aria-hidden="true">{isOpen ? '−' : '+'}</span>
                    </div>
                  </button>
                  {isOpen ? (
                    <div className="week-switcher-panel">
                      <ul className="match-list">
                        {week.unplayedFixtures.map((fixture) => (
                          <li key={fixture.id}>
                            <div>
                              <strong>{fixture.player_one} vs {fixture.player_two}</strong>
                              <div className="fixture-meta">Week {week.week_number} - {fixture.game_variant} - First to {fixture.legs_to_win} legs</div>
                            </div>
                            <div className="fixture-meta">Arrange within the week</div>
                          </li>
                        ))}
                      </ul>
                    </div>
                  ) : null}
                </section>
              )
            })}
          </div>
        ) : null}
      </section>

      <section className="week-grid" aria-label="Visible season weeks">
        {futureWeeks.length === 0 && !fixturesQuery.isLoading ? (
          <article className="week-card empty-card">
            <header>
              <div>
                <span className="section-eyebrow">Season map</span>
                <h2>No locked weeks yet</h2>
              </div>
            </header>
            <p>Once the season starts, upcoming weeks appear here with pairings shown and details held back until unlock.</p>
          </article>
        ) : null}
        {futureWeeks.map((week) => (
          <article className="week-card locked" key={week.week_number}>
            <header>
              <div>
                <span className="section-eyebrow">Season map</span>
                <h2>Week {week.week_number}</h2>
              </div>
              <span className="status-pill locked">Locked</span>
            </header>
            <p>Reveals {formatWhen(week.reveal_at)}</p>
            <ul className="match-list">
              {week.fixtures.map((fixture) => (
                <li key={fixture.id}>
                  <strong>{fixture.player_one} vs {fixture.player_two}</strong>
                  <div className="fixture-meta">Pairing visible</div>
                </li>
              ))}
            </ul>
            <div className="lock-overlay">
              <strong>Visible pairings. Hidden details.</strong>
              <span className="fixture-meta">Unlocks automatically</span>
            </div>
          </article>
        ))}
      </section>
    </>
  )
}
