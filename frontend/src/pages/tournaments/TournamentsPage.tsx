import { StateNotice } from '../../components/StateNotice'
import { useTournaments } from '../../lib/tournamentsApi'
import { readError } from '../../lib/utils'
import { TournamentList } from './TournamentList'

export function TournamentsPage() {
  const tournaments = useTournaments()

  return (
    <>
      <section className="page-intro">
        <h1>Tournaments</h1>
      </section>
      {tournaments.isPending ? <StateNotice message="Loading tournaments..." /> : null}
      {tournaments.error ? <StateNotice tone="error" message={readError(tournaments.error)} /> : null}
      {tournaments.isSuccess ? (
        <section className="content-panel" aria-label="Tournament directory">
          <TournamentList events={tournaments.data} />
        </section>
      ) : null}
    </>
  )
}
