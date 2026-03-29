import { useSeasonSummary, useStandings } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'

export function StandingsPage() {
  const seasonQuery = useSeasonSummary()
  const standingsQuery = useStandings()

  return (
    <section className="standings-card">
      <div className="page-intro">
        <span className="eyebrow">Live table</span>
        <h1>Standings</h1>
        <p className="fixture-meta">{seasonQuery.data?.name ?? 'Active season'}</p>
        <p>Public labels prefer nicknames, while the table still rewards clean legs and relentless finishing.</p>
      </div>

      {standingsQuery.isLoading ? <StateNotice message="Loading live standings..." /> : null}
      {standingsQuery.error ? <StateNotice tone="error" message={readError(standingsQuery.error)} /> : null}
      {!standingsQuery.isLoading && !standingsQuery.error && standingsQuery.data?.length === 0 ? (
        <StateNotice message="Standings will populate once results are entered." />
      ) : null}

      {standingsQuery.data && standingsQuery.data.length > 0 ? (
        <table className="standings-table">
          <thead>
            <tr>
              <th>Player</th>
              <th>P</th>
              <th>W</th>
              <th>L</th>
              <th>LF</th>
              <th>LA</th>
              <th>LD</th>
              <th>Pts</th>
            </tr>
          </thead>
          <tbody>
            {standingsQuery.data.map((row) => (
              <tr key={row.display_name}>
                <td className="player-cell">
                  <strong>{row.player}</strong>
                  <span>{row.display_name}</span>
                </td>
                <td>{row.played}</td>
                <td>{row.won}</td>
                <td>{row.lost}</td>
                <td>{row.legs_for}</td>
                <td>{row.legs_against}</td>
                <td>{row.leg_difference}</td>
                <td className="table-emphasis">{row.points}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
    </section>
  )
}
