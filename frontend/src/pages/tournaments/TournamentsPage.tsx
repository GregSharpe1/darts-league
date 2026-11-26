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
      {tournaments.isPending ? <StateNotice message="Loading tournaments..." /> : null}
      {tournaments.error ? <StateNotice tone="error" message={readError(tournaments.error)} /> : null}
      {tournaments.isSuccess ? (
        <section className="tournament-directory" aria-label="Tournament directory">
          <TournamentList events={tournaments.data} cards />
        </section>
      ) : null}
    </>
  )
}
