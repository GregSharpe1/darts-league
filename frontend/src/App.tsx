import { useEffect } from 'react'
import { NavLink, Route, Routes } from 'react-router-dom'
import './App.css'
import { useSeasonSummary, useBackendVersion } from './lib/api'
import { frontendVersion } from './version'
import { HomePage } from './pages/home/HomePage'
import { StandingsPage } from './pages/standings/StandingsPage'
import { RegisterPage } from './pages/register/RegisterPage'
import { AdminPage } from './pages/admin/AdminPage'

function App() {
  const seasonQuery = useSeasonSummary()
  const backendVersionQuery = useBackendVersion()
  const instanceName = seasonQuery.data?.instance_name ?? 'Darts League'

  useEffect(() => {
    document.title = instanceName
  }, [instanceName])

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand-mark">
          <div className="brand-badge" aria-hidden="true" />
          <div className="brand-copy">
            <strong>{instanceName}</strong>
            <span>Monday unlocks. Saturday bragging rights.</span>
          </div>
        </div>
        <nav className="primary-nav" aria-label="Primary">
          <NavLink to="/">Fixtures</NavLink>
          <NavLink to="/standings">Standings</NavLink>
          {seasonQuery.data?.registration_open ? <NavLink to="/register">Register</NavLink> : null}
        </nav>
      </header>

      <main className="page-frame">
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/standings" element={<StandingsPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/admin" element={<AdminPage />} />
        </Routes>
        <footer className="footer-note" aria-label="Application build details">
          <p>Built for a single active season, weekly reveals, and admin-controlled score entry.</p>
          <p className="footer-version">Frontend {frontendVersion} | Backend {backendVersionQuery.data?.version ?? 'unavailable'}</p>
        </footer>
      </main>
    </div>
  )
}

export default App
