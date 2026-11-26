import { useEffect } from 'react'
import { NavLink, Route, Routes, matchPath, useLocation, useNavigate } from 'react-router-dom'
import './App.css'
import { useSeasonSummary, useBackendVersion, useDivisions, useAdminLogout, useAdminPlayers } from './lib/api'
import { frontendVersion } from './version'
import { HomePage } from './pages/home/HomePage'
import { DivisionsPage } from './pages/home/DivisionsPage'
import { DivisionHomePage } from './pages/home/DivisionHomePage'
import { StandingsPage } from './pages/standings/StandingsPage'
import { RegisterPage } from './pages/register/RegisterPage'
import { AdminTournamentsPage } from './pages/tournaments/AdminTournamentsPage'
import { TournamentPage } from './pages/tournaments/TournamentPage'
import { TournamentsPage } from './pages/tournaments/TournamentsPage'
import { AdminPage } from './pages/admin/AdminPage'
import { DivisionAdminPage } from './pages/admin/DivisionAdminPage'
import { PendingResultsPage } from './pages/admin/PendingResultsPage'
import { MatchPage } from './pages/matches/MatchPage'
import { PlayerPage } from './pages/players/PlayerPage'

function App() {
  const location = useLocation()
  const navigate = useNavigate()
  const onHomePage = location.pathname === '/'
  const onTournamentPage = Boolean(matchPath('/tournaments/*', location.pathname))
  const onAdminPage = Boolean(matchPath('/admin/*', location.pathname))
  const onAdminHome = Boolean(matchPath('/admin', location.pathname))
  const seasonQuery = useSeasonSummary()
  const divisionsQuery = useDivisions()
  const backendVersionQuery = useBackendVersion()
  const adminPlayersQuery = useAdminPlayers(onAdminPage)
  const adminLogoutMutation = useAdminLogout()
  const instanceName = seasonQuery.data?.instance_name ?? 'Darts League'
  const divisionMatch = matchPath('/divisions/:slug/*', location.pathname) ?? matchPath('/divisions/:slug', location.pathname) ?? matchPath('/admin/divisions/:slug', location.pathname)
  const divisionSlug = divisionMatch?.params.slug
  const currentDivision = divisionsQuery.data?.find((division) => division.slug === divisionSlug)
  const onDivisionPage = Boolean(matchPath('/divisions/:slug', location.pathname) || matchPath('/divisions/:slug/standings', location.pathname))
  const adminAuthenticated = adminPlayersQuery.isSuccess

  useEffect(() => {
    document.title = currentDivision ? `${instanceName} | ${currentDivision.name}` : instanceName
  }, [currentDivision, instanceName])

  return (
    <div className="app-shell">
      <header className="topbar">
        <NavLink to="/" className="brand-mark" aria-label="Go to home page">
          <div className="brand-badge" aria-hidden="true" />
          <div className="brand-copy">
            <strong>{instanceName}</strong>
            <span>{currentDivision ? currentDivision.name : 'Monday unlocks. Saturday bragging rights.'}</span>
          </div>
        </NavLink>
        {!onHomePage ? (
          <nav className="primary-nav" aria-label="Primary">
            {onAdminPage ? (
              <>
                {!onAdminHome ? <NavLink to="/admin">Admin home</NavLink> : null}
                {onAdminHome && adminAuthenticated ? <NavLink to="/admin/pending-results">Pending results</NavLink> : null}
                {onAdminHome && adminAuthenticated ? <NavLink to="/admin/tournaments">Tournaments</NavLink> : null}
                {adminAuthenticated ? (
                  <button
                    type="button"
                    onClick={async () => {
                      await adminLogoutMutation.mutateAsync()
                      navigate('/')
                    }}
                    disabled={adminLogoutMutation.isPending}
                  >
                    {adminLogoutMutation.isPending ? 'Logging out...' : 'Log out'}
                  </button>
                ) : null}
              </>
            ) : onTournamentPage ? (
              <NavLink to="/divisions">League</NavLink>
            ) : (
              <>
                {onDivisionPage ? (
                  <>
                    <NavLink to={`/divisions/${divisionSlug}`}>Fixtures</NavLink>
                    <NavLink to={`/divisions/${divisionSlug}/standings`}>Standings</NavLink>
                    <NavLink to="/divisions">All Divisions</NavLink>
                  </>
                ) : <NavLink to="/divisions">Divisions</NavLink>}
                {seasonQuery.data?.registration_open ? <NavLink to="/register">Register</NavLink> : null}
                <NavLink to="/tournaments">Tournaments</NavLink>
              </>
            )}
          </nav>
        ) : null}
      </header>

      <main className="page-frame">
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/divisions" element={<DivisionsPage />} />
          <Route path="/divisions/:slug" element={<DivisionHomePage />} />
          <Route path="/divisions/:slug/standings" element={<StandingsPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/tournaments" element={<TournamentsPage />} />
          <Route path="/tournaments/:slug" element={<TournamentPage />} />
          <Route path="/matches/:fixtureId" element={<MatchPage />} />
          <Route path="/seasons/:seasonId/players/:playerId" element={<PlayerPage />} />
          <Route path="/admin/seasons/:seasonId/players/:playerId" element={<PlayerPage admin />} />
          <Route path="/admin/matches/:fixtureId" element={<MatchPage admin />} />
          <Route path="/admin" element={<AdminPage />} />
          <Route path="/admin/pending-results" element={<PendingResultsPage />} />
          <Route path="/admin/tournaments/:slug?" element={<AdminTournamentsPage />} />
          <Route path="/admin/divisions/:slug" element={<DivisionAdminPage />} />
        </Routes>
        <footer className="footer-note" aria-label="Application build details">
          <p className="footer-version">Frontend {frontendVersion} | Backend {backendVersionQuery.data?.version ?? 'unavailable'}</p>
        </footer>
      </main>
    </div>
  )
}

export default App
