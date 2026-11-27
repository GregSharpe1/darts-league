import { useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useDivisions, useDivisionFixtures, useDivisionStandings, useSeasonSummary, formatWhen } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'
import { CompletedResults } from './CompletedResults'
import { DivisionNavigation } from './DivisionNavigation'

export function DivisionHomePage() {
  const { slug = '' } = useParams()
  const seasonQuery = useSeasonSummary()
  const divisionsQuery = useDivisions()
  const fixturesQuery = useDivisionFixtures(slug)
  const standingsQuery = useDivisionStandings(slug)
  const [fixtureSearch, setFixtureSearch] = useState('')
  const [fixtureSearchOpen, setFixtureSearchOpen] = useState(false)
  const fixtureSearchTerm = fixtureSearch.trim().toLowerCase()
  const division = divisionsQuery.data?.find((item) => item.slug === slug)

  const currentWeek = fixturesQuery.data?.weeks.find((week) => week.week_number === fixturesQuery.data.current_week)
  const futureWeeks = fixturesQuery.data?.weeks.filter((week) => week.status === 'locked') ?? []
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
  const exactPlayer = fixturePlayerOptions.some((player) => player.toLowerCase() === fixtureSearchTerm)
  const matchesSearch = (label: string) => {
    const normalizedLabel = label.toLowerCase()
    return exactPlayer ? normalizedLabel === fixtureSearchTerm : normalizedLabel.includes(fixtureSearchTerm)
  }
  const unlockedWeeks = fixturesQuery.data?.weeks.filter((week) => week.status === 'unlocked') ?? []
  const matchingWeeks = unlockedWeeks.map((week) => ({
    ...week,
    fixtures: week.fixtures.filter((fixture) => matchesSearch(fixture.player_one) || matchesSearch(fixture.player_two)),
  }))
  const unplayedWeeks = matchingWeeks
    .map((week) => ({ ...week, fixtures: week.fixtures.filter((fixture) => !fixture.result) }))
    .filter((week) => week.fixtures.length > 0)
  const gamesLeftToPlay = unplayedWeeks.reduce((total, week) => total + week.fixtures.length, 0)
  const hasRecordedResults = matchingWeeks.some((week) => week.fixtures.some((fixture) => fixture.result))
  const seasonCompleted = seasonQuery.data?.status === 'completed'

  if (!slug) {
    return <StateNotice tone="error" message="Division not found." />
  }

  return (
    <>
      <section className="hero-panel competition-hero competition-hero--league">
        <div className="hero-copy">
          <span className="competition-kicker">League fixtures</span>
          <h1>{division?.name ?? 'Division'}</h1>
          <p className="fixture-meta">{seasonQuery.data?.name ?? 'Active season'}</p>
          <p>
            Follow this division&apos;s fixtures and results. Match details unlock Mondays at 09:00 UK time.
          </p>
          <DivisionNavigation slug={slug} />
        </div>

        <div className="hero-side">
          <article className="metric-card">
            <span className="section-eyebrow">Division roster</span>
            <strong>{standingsQuery.data?.length ?? '-'}</strong>
            <p>{seasonQuery.data?.status === 'completed' ? 'League completed. Final results and standings remain available.' : seasonQuery.data?.registration_open ? 'assigned players will appear here once divisions are set.' : `season live across ${fixturesQuery.data?.weeks.length ?? 0} weeks.`}</p>
          </article>
          {seasonQuery.data && !seasonQuery.data.registration_open ? (
            <article className="metric-card">
              <span className="section-eyebrow">Match format</span>
              <strong>{seasonQuery.data.game_variant}</strong>
              <p>First to {seasonQuery.data.legs_to_win} legs | {seasonQuery.data.games_per_week} game{seasonQuery.data.games_per_week !== 1 ? 's' : ''} per week</p>
            </article>
          ) : null}
        </div>
      </section>

      <section className="content-panel league-fixtures">
        {!seasonCompleted ? <div className="card-header">
          <div className="card-copy">
            <h2>Games left to play</h2>
          </div>
          <span className="status-pill live">{gamesLeftToPlay > 0 ? `${gamesLeftToPlay} live` : currentWeek ? 'Unlocked' : 'Waiting'}</span>
        </div> : null}
        <div className="fixture-search field" onBlur={(event) => { if (!event.currentTarget.contains(event.relatedTarget)) setFixtureSearchOpen(false) }}>
          <label htmlFor="fixture-player-search">Find a player&apos;s games</label>
          <input
            id="fixture-player-search"
            role="combobox"
            type="search"
            value={fixtureSearch}
            onChange={(event) => { setFixtureSearch(event.target.value); setFixtureSearchOpen(true) }}
            onFocus={() => setFixtureSearchOpen(true)}
            placeholder="Search by player"
            aria-autocomplete="list"
            aria-controls="fixture-player-options"
            aria-expanded={showFixturePlayerOptions}
          />
          {showFixturePlayerOptions ? (
            <div className="fixture-search-options" id="fixture-player-options" role="listbox" aria-label="Players">
              {filteredFixturePlayerOptions.length > 0 ? filteredFixturePlayerOptions.map((player) => (
                <button key={player} type="button" role="option" onClick={() => { setFixtureSearch(player); setFixtureSearchOpen(false) }}>{player}</button>
              )) : <span className="fixture-meta">No players match that search.</span>}
            </div>
          ) : null}
          <p className="fixture-meta">Search by name or nickname across unlocked weeks.</p>
        </div>
        {fixturesQuery.isLoading ? <StateNotice message="Loading the division board..." /> : null}
        {fixturesQuery.error ? <StateNotice tone="error" message={readError(fixturesQuery.error)} /> : null}
        {!fixturesQuery.isLoading && !fixturesQuery.error && !division ? <StateNotice message="Division not found." /> : null}
        {!fixturesQuery.isLoading && !fixturesQuery.error && division && !currentWeek ? (
          <StateNotice message="No public week is unlocked yet. Start the season in admin to generate fixtures." />
        ) : null}
        {!seasonCompleted && !fixtureSearchTerm && !fixturesQuery.isLoading && !fixturesQuery.error && currentWeek && gamesLeftToPlay === 0 ? (
          <StateNotice message="Every unlocked fixture in this division has been played so far. Check locked weeks for what is coming next." />
        ) : null}
        {fixtureSearchTerm && !fixturesQuery.isLoading && !fixturesQuery.error && gamesLeftToPlay === 0 && !hasRecordedResults ? (
          <StateNotice message="No visible fixtures match that player name." />
        ) : null}
        {!seasonCompleted && unplayedWeeks.length > 0 ? (
          <div className="week-switcher" aria-label="Weeks with matches left to play">
            {unplayedWeeks.map((week) => (
              <section className="week-switcher-item open" key={week.week_number}>
                <div className="week-switcher-trigger">
                  <div className="week-switcher-copy">
                    <span className="section-eyebrow">Unplayed fixtures</span>
                    <strong>Week {week.week_number}</strong>
                    <span className="fixture-meta">{week.week_number === fixturesQuery.data?.current_week ? 'Current week unlocked' : 'Previous week still outstanding'}</span>
                  </div>
                  <span className={`status-pill ${week.week_number === fixturesQuery.data?.current_week ? 'live' : 'locked'}`}>{week.fixtures.length} match{week.fixtures.length !== 1 ? 'es' : ''} left</span>
                </div>
                <div className="week-switcher-panel">
                  <ul className="match-list">
                    {week.fixtures.map((fixture) => (
                      <li key={fixture.id}>
                        <div>
                          <strong>{fixture.player_one} vs {fixture.player_two}</strong>
                          <div className="fixture-meta">Week {week.week_number} - {fixture.game_variant} - First to {fixture.legs_to_win} legs</div>
                        </div>
                      </li>
                    ))}
                  </ul>
                </div>
              </section>
            ))}
          </div>
        ) : null}
      </section>

      {hasRecordedResults || (seasonCompleted && !fixtureSearchTerm) ? (
        <CompletedResults
          key={fixtureSearchTerm}
          weeks={matchingWeeks}
          title={seasonCompleted ? 'Final results' : 'Recorded results'}
          initiallyOpen={Boolean(fixtureSearchTerm)}
        />
      ) : null}

      {!fixtureSearchTerm && <section className="week-grid" aria-label="Visible season weeks">
        {futureWeeks.length === 0 && !fixturesQuery.isLoading && seasonQuery.data?.status !== 'completed' ? (
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
          </article>
        ))}
      </section>}
    </>
  )
}
