import { useEffect, useLayoutEffect } from 'react'
import { NavLink, Route, Routes, matchPath, useLocation, useNavigate, useNavigationType } from 'react-router-dom'
import './App.css'
import './publicPages.css'
import { PublicNavigation } from './components/PublicNavigation'
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
import { PlayerIdentityPage } from './pages/admin/PlayerIdentityPage'
import { DivisionAdminPage } from './pages/admin/DivisionAdminPage'
import { PendingResultsPage } from './pages/admin/PendingResultsPage'
import { MatchPage } from './pages/matches/MatchPage'
import { PlayerPage } from './pages/players/PlayerPage'
import { PlayerDirectoryPage } from './pages/players/PlayerDirectoryPage'
import { PlayerRecordsPage } from './pages/players/PlayerRecordsPage'
import { TournamentPlayerPage } from './pages/players/PlayerPlacements'

function App() {
  const location = useLocation()
  const navigate = useNavigate()
  const navigationType = useNavigationType()
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
  const adminAuthenticated = adminPlayersQuery.isSuccess

  useEffect(() => {
    document.title = currentDivision ? `${instanceName} | ${currentDivision.name}` : instanceName
  }, [currentDivision, instanceName])

  useLayoutEffect(() => {
    if (navigationType !== 'POP' && !location.hash) {
      window.scrollTo({ top: 0, left: 0, behavior: 'instant' })
    }
  }, [location.pathname, location.hash, navigationType])

  return (
    <div className="app-shell">
      <header className="topbar public-topbar">
        <NavLink to="/" className="brand-mark" aria-label="Go to home page">
          <img className="brand-badge" src="/favicon.png" width={96} height={96} alt="" />
          <div className="brand-copy">
            <strong>{instanceName}</strong>
            <span>{currentDivision ? currentDivision.name : 'Monday unlocks. Saturday bragging rights.'}</span>
          </div>
        </NavLink>
        <PublicNavigation />
      </header>

      <main className="page-frame">
        {onAdminPage && (adminAuthenticated || !onAdminHome) ? (
          <nav className="primary-nav admin-navigation" aria-label="Administration">
            <NavLink to="/admin" end aria-label="Admin home"><span className="nav-full-label">Admin home</span><span className="nav-compact-label" aria-hidden="true">Home</span></NavLink>
            {adminAuthenticated ? (
              <>
                <NavLink to="/admin/pending-results" aria-label="Pending results"><span className="nav-full-label">Pending results</span><span className="nav-compact-label" aria-hidden="true">Results</span></NavLink>
                <NavLink to="/admin/tournaments" aria-label="Manage tournaments"><span className="nav-full-label">Manage tournaments</span><span className="nav-compact-label" aria-hidden="true">Events</span></NavLink>
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
              </>
            ) : null}
          </nav>
        ) : null}
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/divisions" element={<DivisionsPage />} />
          <Route path="/divisions/:slug" element={<DivisionHomePage />} />
          <Route path="/divisions/:slug/standings" element={<StandingsPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/tournaments" element={<TournamentsPage />} />
          <Route path="/tournaments/:slug" element={<TournamentPage />} />
          <Route path="/matches/:fixtureId" element={<MatchPage />} />
          <Route path="/players" element={<PlayerDirectoryPage />} />
          <Route path="/players/records" element={<PlayerRecordsPage />} />
          <Route path="/players/tournaments/:slug/:entryId" element={<TournamentPlayerPage />} />
          <Route path="/seasons/:seasonId/players/:playerId" element={<PlayerPage />} />
          <Route path="/admin/seasons/:seasonId/players/:playerId" element={<PlayerPage admin />} />
          <Route path="/admin/matches/:fixtureId" element={<MatchPage admin />} />
          <Route path="/admin" element={<AdminPage />} />
          <Route path="/admin/player-identities" element={<PlayerIdentityPage />} />
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
