import { Link, matchPath, useLocation } from 'react-router-dom'

const destinations = [
  { to: '/divisions', label: 'League', compactLabel: 'League' },
  { to: '/tournaments', label: 'Tournaments', compactLabel: 'Events' },
  { to: '/players', label: 'Player stats', compactLabel: 'Players' },
  { to: '/players/records', label: 'Records', compactLabel: 'Records' },
]

function activeDestination(pathname: string) {
  if (matchPath('/players/records', pathname)) return '/players/records'
  if (matchPath('/players/*', pathname) || matchPath('/seasons/:seasonId/players/:playerId', pathname)) return '/players'
  if (matchPath('/tournaments/*', pathname)) return '/tournaments'
  if (matchPath('/divisions/*', pathname) || matchPath('/register', pathname) || matchPath('/matches/:fixtureId', pathname)) return '/divisions'
}

export function PublicNavigation() {
  const location = useLocation()
  const active = activeDestination(location.pathname)

  return <>
    <nav className="primary-nav public-navigation" aria-label="Primary">
      {destinations.map(({ to, label, compactLabel }) =>
        <Link key={to} to={to} aria-label={label} className={active === to ? 'active' : undefined}
          aria-current={active === to ? (matchPath(to, location.pathname) ? 'page' : 'location') : undefined}>
          <span className="nav-full-label">{label}</span>
          <span className="nav-compact-label" aria-hidden="true">{compactLabel}</span>
        </Link>)}
    </nav>
    <Link className="public-admin-link text-link" to="/admin">Admin</Link>
    <details className="public-admin-menu" key={location.pathname} onKeyDown={event => {
      if (event.key === 'Escape' && event.currentTarget.open) {
        event.stopPropagation()
        event.currentTarget.open = false
        event.currentTarget.querySelector('summary')?.focus()
      }
    }}>
      <summary>More</summary>
      <div className="public-admin-menu-panel"><Link className="text-link" to="/admin">Admin</Link></div>
    </details>
  </>
}
