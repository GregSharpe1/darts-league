import { Link, NavLink } from 'react-router-dom'

export function DivisionNavigation({ slug }: { slug: string }) {
  const fixturesPath = `/divisions/${encodeURIComponent(slug)}`
  return <nav className="division-navigation" aria-label="Division">
    <NavLink className="text-link" to={fixturesPath} end>Fixtures</NavLink>
    <NavLink className="text-link" to={`${fixturesPath}/standings`} end>Standings</NavLink>
    <Link className="text-link" to="/divisions">All divisions</Link>
  </nav>
}
