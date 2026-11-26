import { StateNotice } from '../../components/StateNotice'
import { useTournaments } from '../../lib/tournamentsApi'
import { readError } from '../../lib/utils'
import { TournamentList } from './TournamentList'

export function TournamentsPage() {
  const tournaments = useTournaments()

  return (
    <>
      <section className="hero-panel competition-hero competition-hero--tournament competition-hero--compact">
        <div className="hero-copy">
          <h1>Tournaments</h1>
          <p>Enter an open knockout or follow a published draw through to the final.</p>
        </div>
      </section>
      <section className="content-panel tournament-directory" aria-label="Tournament directory">
        <div className="card-header">
          <h2>Public tournament boards</h2>
        </div>
        {tournaments.isPending ? <StateNotice message="Loading tournaments..." /> : null}
        {tournaments.error ? <StateNotice tone="error" message={readError(tournaments.error)} /> : null}
        {tournaments.isSuccess ? <TournamentList events={tournaments.data} cards /> : null}
      </section>
    </>
  )
}
