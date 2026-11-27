import { Link } from 'react-router-dom'
import { useSeasonSummary } from '../../lib/api'
import { useTournaments } from '../../lib/tournamentsApi'
import { TournamentList } from '../tournaments/TournamentList'

export function HomePage() {
  const league = useSeasonSummary()
  const tournaments = useTournaments()
  const season = league.data
  const openEvents = tournaments.data?.filter((event) => event.registration_open && !event.started) ?? []
  const latestBracket = tournaments.data?.find((event) => event.started)
  const featuredEvents = openEvents
    .sort((a, b) => a.starts_on.localeCompare(b.starts_on))
    .slice(0, 2)
  if (latestBracket) featuredEvents.push(latestBracket)

  return (
    <>
      <section className="page-intro">
        <h1>Darts league &amp; tournaments</h1>
        <p>Find your division&apos;s next match or join a standalone knockout event.</p>
      </section>

      <div className="grid-panels">
        <section className="content-panel competition-card competition-card--league" aria-labelledby="home-league">
          <header className="competition-card-cover"><h2 id="home-league">League matches</h2></header>
          <p>Play everyone in your division once and follow the league table as results come in. New fixture details unlock on Mondays at 9am UK time.</p>
          <div className="competition-details">
            {season ? (
              <>
                <h3>{season.name}</h3>
                <p>
                  {season.registration_open
                    ? 'League registration is open. An admin will assign you to a division.'
                    : season.status === 'completed' || season.remaining_fixtures === 0
                      ? 'All matches played. Final results and standings are available.'
                      : `${season.remaining_fixtures} ${season.remaining_fixtures === 1 ? 'match' : 'matches'} left to play across all divisions.`}
                </p>
              </>
            ) : <p>{league.isError ? 'League status is currently unavailable.' : 'Loading league status...'}</p>}
          </div>
          <div className="hero-actions competition-actions">
            <Link to="/divisions">Browse divisions</Link>
            {season?.registration_open ? <Link to="/register">Register for the league</Link> : null}
          </div>
        </section>

        <section className="content-panel competition-card competition-card--tournament" aria-labelledby="home-tournaments">
          <header className="competition-card-cover"><h2 id="home-tournaments">Tournaments</h2></header>
          <p>Sign up for each event separately. One loss ends your run. Follow the bracket from the first draw to the final.</p>
          <div className="competition-details">
            {tournaments.isPending ? <p>Loading tournaments...</p> : null}
            {tournaments.isError ? <p>Tournament listings are currently unavailable.</p> : null}
            {tournaments.isSuccess ? (
              <>
                {openEvents.length === 0 ? <p>No tournaments are accepting entries right now.</p> : null}
                {featuredEvents.length > 0 ? <TournamentList events={featuredEvents} /> : null}
              </>
            ) : null}
          </div>
          <div className="hero-actions competition-actions">
            <Link to="/tournaments">Browse tournaments</Link>
          </div>
        </section>
      </div>
    </>
  )
}
