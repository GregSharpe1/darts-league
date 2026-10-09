import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { MatchAnalysis } from '../../components/match'
import { ApiError } from '../../lib/api'
import { fetchMatch } from '../../lib/matchApi'
import './MatchPage.css'

export function MatchPage({ admin = false }: { readonly admin?: boolean }) {
  const { fixtureId = '' } = useParams()
  const query = useQuery({
    queryKey: ['fixture-match', admin, fixtureId],
    queryFn: ({ signal }) => fetchMatch(fixtureId, admin, signal),
    staleTime: 0, gcTime: 0, retry: false, refetchOnMount: 'always', refetchOnWindowFocus: 'always',
  })
  const status = query.error instanceof ApiError ? query.error.status : null

  return <section className="match-page" aria-label="Match details">
    <header className="match-page-heading">
      <div><span className="eyebrow">{admin ? 'Admin match view' : 'Recorded league match'}</span><h1>Match analysis</h1></div>
      {admin && <Link to="/admin">Admin home</Link>}
    </header>
    {query.isPending || query.isFetching ? <MatchAnalysis status="loading" /> :
      query.isError ? status === 404 ? <div className="content-panel" role="status"><h2>Match not found.</h2><p>This match is not available.</p></div> :
        status === 401 ? <div className="content-panel" role="status"><h2>Admin login required</h2><Link to="/admin">Admin login</Link></div> :
          <MatchAnalysis status="error" onRetry={() => { void query.refetch() }} /> :
        <MatchAnalysis key={`${admin}-${fixtureId}`} status="ready" match={query.data.match} />}
  </section>
}
