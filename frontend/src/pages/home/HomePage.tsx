import { useMemo, useState } from 'react'
import { NavLink } from 'react-router-dom'
import { useSeasonSummary, usePublicFixtures, useStandings, formatWhen } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'

export function HomePage() {
  const seasonQuery = useSeasonSummary()
  const fixturesQuery = usePublicFixtures()
  const standingsQuery = useStandings()
  const [fixtureSearch, setFixtureSearch] = useState('')
  const [fixtureSearchOpen, setFixtureSearchOpen] = useState(false)
  const fixtureSearchTerm = fixtureSearch.trim().toLowerCase()

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
  const searchedUnplayedWeeks = useMemo(
    () =>
      fixturesQuery.data?.weeks
        .map((week) => ({
          ...week,
          unplayedFixtures: week.fixtures.filter((fixture) => {
            if (fixture.result) {
              return false
            }

            if (!fixtureSearchTerm) {
              return false
            }

            return `${fixture.player_one} ${fixture.player_two}`.toLowerCase().includes(fixtureSearchTerm)
          }),
        }))
        .filter((week) => week.unplayedFixtures.length > 0) ?? [],
    [fixtureSearchTerm, fixturesQuery.data?.weeks],
  )
  const fixturePlayerOptions = useMemo(() => {
    const players = new Map<string, string>()

    for (const row of standingsQuery.data ?? []) {
      const label = row.player === row.display_name ? row.player : `${row.player} (${row.display_name})`
      players.set(label.toLowerCase(), label)
    }

    return [...players.values()].sort((a, b) => a.localeCompare(b))
  }, [standingsQuery.data])
  const filteredFixturePlayerOptions = fixturePlayerOptions.filter((player) => player.toLowerCase().includes(fixtureSearchTerm))
  const showFixturePlayerOptions = fixtureSearchOpen && fixturePlayerOptions.length > 0
  const gamesLeftToPlay = unplayedUnlockedWeeks.reduce((total, week) => total + week.unplayedFixtures.length, 0)
  const searchedGamesLeft = searchedUnplayedWeeks.reduce((total, week) => total + week.unplayedFixtures.length, 0)
  const defaultOpenWeek =
    unplayedUnlockedWeeks.find((week) => week.week_number === fixturesQuery.data?.current_week)?.week_number ??
    unplayedUnlockedWeeks[0]?.week_number ??
    null
  const [selectedOpenWeekNumber, setSelectedOpenWeekNumber] = useState<number | null>(null)
  const openWeekNumber = selectedOpenWeekNumber !== null && unplayedUnlockedWeeks.some((week) => week.week_number === selectedOpenWeekNumber)
    ? selectedOpenWeekNumber
    : defaultOpenWeek

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
        <div
          className="fixture-search field"
          onBlur={(event) => {
            if (!event.currentTarget.contains(event.relatedTarget)) {
              setFixtureSearchOpen(false)
            }
          }}
        >
          <label htmlFor="fixture-player-search">Find your remaining games</label>
          <input
            id="fixture-player-search"
            role="combobox"
            type="search"
            value={fixtureSearch}
            onChange={(event) => setFixtureSearch(event.target.value)}
            onFocus={() => setFixtureSearchOpen(true)}
            placeholder="Search by your darts name"
            aria-autocomplete="list"
            aria-controls="fixture-player-options"
            aria-expanded={showFixturePlayerOptions}
          />
          {showFixturePlayerOptions ? (
            <div className="fixture-search-options" id="fixture-player-options" role="listbox" aria-label="Players">
              {filteredFixturePlayerOptions.length > 0 ? filteredFixturePlayerOptions.map((player) => (
                <button
                  type="button"
                  role="option"
                  key={player}
                  onClick={() => {
                    setFixtureSearch(player.replace(/\s+\(.+\)$/, ''))
                    setFixtureSearchOpen(false)
                  }}
                >
                  {player}
                </button>
              )) : <span className="fixture-meta">No players match that search.</span>}
            </div>
          ) : null}
          <p className="fixture-meta">Search visible unplayed fixtures by either player name. Locked weeks still keep match details gated.</p>
        </div>
        {fixturesQuery.isLoading ? <StateNotice message="Loading the season board..." /> : null}
        {fixturesQuery.error ? <StateNotice tone="error" message={readError(fixturesQuery.error)} /> : null}
        {!fixturesQuery.isLoading && !fixturesQuery.error && !currentWeek ? (
          <StateNotice message="No public week is unlocked yet. Start the season in admin to generate fixtures." />
        ) : null}
        {!fixtureSearchTerm && !fixturesQuery.isLoading && !fixturesQuery.error && currentWeek && gamesLeftToPlay === 0 ? (
          <StateNotice message="Every unlocked fixture has been played so far. Check locked weeks for what is coming next." />
        ) : null}
        {fixtureSearchTerm && !fixturesQuery.isLoading && !fixturesQuery.error ? (
          <div className="fixture-search-results" aria-live="polite">
            <div className="card-header compact-header">
              <div className="card-copy">
                <span className="section-eyebrow">Search results</span>
                <h3>{searchedGamesLeft} remaining match{searchedGamesLeft !== 1 ? 'es' : ''}</h3>
              </div>
            </div>
            {searchedUnplayedWeeks.length > 0 ? (
              <div className="week-switcher search-results-list">
                {searchedUnplayedWeeks.map((week) => (
                  <section className="week-switcher-item open" key={week.week_number}>
                    <div className="week-switcher-trigger static-trigger">
                      <div className="week-switcher-copy">
                        <span className="section-eyebrow">{week.status === 'locked' ? 'Locked pairing' : 'Unplayed fixture'}</span>
                        <strong>Week {week.week_number}</strong>
                        <span className="fixture-meta">
                          {week.status === 'locked' ? `Reveals ${formatWhen(week.reveal_at)}` : week.week_number === fixturesQuery.data?.current_week ? 'Current week unlocked' : 'Previous week still outstanding'}
                        </span>
                      </div>
                      <div className="week-switcher-meta">
                        <span className={`status-pill ${week.status === 'locked' ? 'locked' : 'live'}`}>{week.status}</span>
                      </div>
                    </div>
                    <div className="week-switcher-panel">
                      <ul className="match-list">
                        {week.unplayedFixtures.map((fixture) => (
                          <li key={fixture.id}>
                            <div>
                              <strong>{fixture.player_one} vs {fixture.player_two}</strong>
                              <div className="fixture-meta">
                                {week.status === 'locked' ? 'Pairing visible. Match details locked.' : `Week ${week.week_number} - ${fixture.game_variant} - First to ${fixture.legs_to_win} legs`}
                              </div>
                            </div>
                            <div className="fixture-meta">{week.status === 'locked' ? 'Unlocks automatically' : 'Arrange within the week'}</div>
                          </li>
                        ))}
                      </ul>
                    </div>
                  </section>
                ))}
              </div>
            ) : (
              <StateNotice message="No remaining visible fixtures match that player name." />
            )}
          </div>
        ) : null}
        {!fixtureSearchTerm && unplayedUnlockedWeeks.length > 0 ? (
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
                    onClick={() => setSelectedOpenWeekNumber(week.week_number)}
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
