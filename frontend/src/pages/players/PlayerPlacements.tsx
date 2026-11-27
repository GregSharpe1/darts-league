import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { ApiError, request } from '../../lib/api'

type Placement = {
  position: number
  competition: string
  division?: string
  tournament_slug?: string
  tied?: boolean
}

type Placements = {
  name: string
  league: Placement | null
  tournament: Placement | null
}

function usePlacements(path: string) {
  return useQuery({
    queryKey: ['player-placements', path],
    queryFn: ({ signal }) => request<Placements>(path, { signal, cache: 'no-store' }),
    retry: false,
  })
}

function ordinal(position: number) {
  const lastTwo = position % 100
  const suffix = lastTwo >= 11 && lastTwo <= 13 ? 'th' : ({ 1: 'st', 2: 'nd', 3: 'rd' }[position % 10] ?? 'th')
  return `${position}${suffix}`
}

function PlacementCard({ label, placement, empty }: {
  label: string
  placement: Placement | null
  empty: string
}) {
  return <div className="player-metric">
    <h3>{label}</h3>
    {placement ? <>
      <strong>{placement.tied ? 'Tied ' : ''}{ordinal(placement.position)}</strong>
      <p>{placement.tournament_slug
        ? <Link className="text-link" to={`/tournaments/${placement.tournament_slug}`}>{placement.competition}</Link>
        : <>{placement.competition} / {placement.division}</>}</p>
    </> : <p>{empty}</p>}
  </div>
}

function PlacementSection({ placements }: { placements: Placements }) {
  return <section className="content-panel" aria-labelledby="player-placements-title">
    <h2 id="player-placements-title">Best placements</h2>
    <div className="player-best-placements">
      <PlacementCard label="League" placement={placements.league} empty="No final league placement yet." />
      <PlacementCard label="Tournament" placement={placements.tournament} empty="No recorded tournament finish yet." />
    </div>
  </section>
}

export function LeaguePlayerPlacements({ seasonId, playerId }: { seasonId: string; playerId: string }) {
  const query = usePlacements(`/api/seasons/${seasonId}/players/${playerId}/placements`)
  if (query.isPending) return <p className="content-panel" role="status">Loading best placements...</p>
  if (query.isError) return <p className="content-panel" role="status">Best placements could not be loaded.</p>
  return <PlacementSection placements={query.data} />
}

export function TournamentPlayerPage() {
  const { slug = '', entryId = '' } = useParams()
  const query = usePlacements(`/api/tournaments/${encodeURIComponent(slug)}/players/${encodeURIComponent(entryId)}/placements`)
  return <div className="player-page">
    <header className="player-page-heading"><Link className="text-link" to="/players">Player stats</Link></header>
    {query.isPending ? <p className="content-panel" role="status">Loading player statistics...</p> :
      query.isError ? <div className="content-panel" role="status">
        <h1>{query.error instanceof ApiError && query.error.status === 404 ? 'Player not found' : 'Player statistics unavailable'}</h1>
      </div> : <>
        <div className="player-hero content-panel"><h1>{query.data.name}</h1></div>
        <PlacementSection placements={query.data} />
      </>}
  </div>
}
