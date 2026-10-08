import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { ApiError } from '../../lib/api'
import { fetchPlayerStatistics } from '../../lib/playerStatistics'
import { PlayerStatisticsView } from './PlayerStatisticsView'
import './PlayerPage.css'

export function PlayerPage({ admin = false }: { readonly admin?: boolean }) {
  const { seasonId = '', playerId = '' } = useParams()
  const query = useQuery({
    queryKey: ['player-statistics', admin, seasonId, playerId],
    queryFn: ({ signal }) => fetchPlayerStatistics({ seasonId, playerId, admin }, signal),
    staleTime: 0, gcTime: 0, retry: false, refetchOnMount: 'always', refetchOnWindowFocus: 'always',
  })
  const status = query.error instanceof ApiError ? query.error.status : null
  return <section className="player-page" aria-label="Player season statistics">
    <header className="player-page-heading">
      <span className="eyebrow">{admin ? 'Admin player view' : 'Player season statistics'} / Season {seasonId}</span>
      <Link to={admin ? '/admin' : '/'}>{admin ? 'Admin home' : 'All divisions'}</Link>
    </header>
    {query.isPending || query.isFetching ? <div className="content-panel" role="status">Loading player statistics...</div> :
      query.isError ? <div className="content-panel" role="status">
        <h1>{status === 404 ? 'Player not found in this season.' : status === 401 ? 'Admin login required' : 'Player statistics unavailable'}</h1>
        {status === 401 ? <Link to="/admin">Admin login</Link> : status === 404 ? <p>Check the season and player link.</p> :
          <><p>Statistics could not be loaded. Please try again.</p><button type="button" onClick={() => { void query.refetch() }}>Try again</button></>}
      </div> : <PlayerStatisticsView key={`${admin}-${seasonId}-${playerId}`} stats={query.data} admin={admin} />}
  </section>
}
