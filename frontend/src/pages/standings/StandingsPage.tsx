import { useParams } from 'react-router-dom'
import { useDivisions, useDivisionStandings, useSeasonSummary } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'

function formatAverage(average?: number | null) {
  return average === undefined || average === null ? '-' : average.toFixed(2)
}

export function StandingsPage() {
  const { slug = '' } = useParams()
  const seasonQuery = useSeasonSummary()
  const divisionsQuery = useDivisions()
  const standingsQuery = useDivisionStandings(slug)
  const division = divisionsQuery.data?.find((item) => item.slug === slug)

  return (
    <section className="standings-card">
      <div className="page-intro">
        <span className="eyebrow">Live table</span>
        <h1>{division?.name ?? 'Division standings'}</h1>
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
              <th>LW</th>
              <th>LL</th>
              <th>LD</th>
              <th>Avg</th>
              <th>Pts</th>
            </tr>
          </thead>
          <tbody>
            {standingsQuery.data.map((row, index) => (
              <tr key={row.display_name}>
                <td className="player-cell">
                  <strong>
                    <span className="position-badge" aria-label={`Position ${index + 1}`}>{index + 1}</span>
                    {row.player}
                  </strong>
                  <span>{row.display_name}</span>
                </td>
                <td>{row.played}</td>
                <td>{row.won}</td>
                <td>{row.lost}</td>
                <td>{row.legs_for}</td>
                <td>{row.legs_against}</td>
                <td>{row.leg_difference}</td>
                <td>{formatAverage(row.average)}</td>
                <td className="table-emphasis">{row.points}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
    </section>
  )
}
