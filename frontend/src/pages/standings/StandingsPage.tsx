import { Link, useParams } from 'react-router-dom'
import { useDivisions, useDivisionStandings, useSeasonSummary } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'
import { DivisionNavigation } from '../home/DivisionNavigation'

function formatAverage(average?: number | null) {
  return average === undefined || average === null ? '-' : average.toFixed(2)
}

export function StandingsPage() {
  const { slug = '' } = useParams()
  const seasonQuery = useSeasonSummary()
  const divisionsQuery = useDivisions()
  const standingsQuery = useDivisionStandings(slug)
  const divisions = [...(divisionsQuery.data ?? [])].sort((a, b) => a.position - b.position)
  const divisionIndex = divisions.findIndex(division => division.slug === slug)
  const division = divisions[divisionIndex]
  const standings = standingsQuery.data ?? []
  const showZones = !standingsQuery.error && divisionIndex >= 0 && standings.some(row => row.played > 0)
  const higherDivision = showZones ? divisions[divisionIndex - 1] : undefined
  const lowerDivision = showZones ? divisions[divisionIndex + 1] : undefined

  return (
    <>
      <section className="hero-panel competition-hero competition-hero--league competition-hero--compact">
        <div className="hero-copy">
          <span className="competition-kicker">{seasonQuery.data?.status === 'completed' ? 'Final standings' : 'League standings'}</span>
          <h1>{division?.name ?? 'Division standings'}</h1>
          <p>{seasonQuery.data?.name ?? 'Active season'}</p>
          {slug ? <DivisionNavigation slug={slug} /> : null}
        </div>
      </section>

      <section className="standings-card league-standings" aria-label="Standings">
        {standingsQuery.isLoading ? <StateNotice message="Loading live standings..." /> : null}
        {standingsQuery.error ? <StateNotice tone="error" message={readError(standingsQuery.error)} /> : null}
        {!standingsQuery.isLoading && !standingsQuery.error && standingsQuery.data?.length === 0 ? (
          <StateNotice message="Standings will populate once results are entered." />
        ) : null}

        {higherDivision || lowerDivision ? (
          <div className="standings-zones" aria-label="Division movement" role="group">
            {higherDivision && <span><strong className="promotion-label">Top two</strong> &middot; Promotion to <Link className="text-link" to={`/divisions/${higherDivision.slug}/standings`}>{higherDivision.name}</Link></span>}
            {lowerDivision && <span><strong className="relegation-label">Bottom two</strong> &middot; Relegation to <Link className="text-link" to={`/divisions/${lowerDivision.slug}/standings`}>{lowerDivision.name}</Link></span>}
            {seasonQuery.data?.status !== 'completed' && <span className="standings-zones-note">Provisional until the league ends.</span>}
          </div>
        ) : null}
        {standings.length > 0 ? (
          <div className="standings-scroll" role="region" aria-label="Standings table" tabIndex={0}>
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
                {standings.map((row, index) => {
                  const promotion = higherDivision && index < 2
                  const relegation = lowerDivision && index >= standings.length - 2

                  return (
                    <tr key={row.display_name} data-promotion={promotion || undefined} data-relegation={relegation || undefined}>
                      <td className="player-cell">
                        <strong>
                          <span className="position-badge" aria-label={`Position ${index + 1}`}>{index + 1}</span>
                          {row.player_id && seasonQuery.data?.id ? <Link className="text-link" to={`/seasons/${seasonQuery.data.id}/players/${row.player_id}`}>{row.player}</Link> : row.player}
                        </strong>
                        <span>{row.display_name}</span>
                        {promotion || relegation ? (
                          <div className="standings-zone-labels">
                            {promotion && <strong className="promotion-label"><span aria-hidden="true">&#8593;</span> Promotion</strong>}
                            {relegation && <strong className="relegation-label"><span aria-hidden="true">&#8595;</span> Relegation</strong>}
                          </div>
                        ) : null}
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
                  )
                })}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>
    </>
  )
}
