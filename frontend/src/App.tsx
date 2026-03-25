import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { NavLink, Route, Routes } from 'react-router-dom'
import './App.css'
import {
  ApiError,
  formatWhen,
  formatAverage,
  useAdminFixtures,
  useAdminLogin,
  useAdminLogout,
  useAdminPlayers,
  useAuditLog,
  useDeletePlayer,
  useGamesPerWeekPresets,
  usePublicFixtures,
  useRegisterPlayer,
  useSaveResult,
  useSchedulePreview,
  useUndoResult,
  useUpdateSeason,
  useUpdateSeasonConfig,
  useSeasonStart,
  useSeasonSummary,
  useStandings,
} from './lib/api'
import type { AdminFixture, AuditEntry, Player } from './lib/api'

function App() {
  const seasonQuery = useSeasonSummary()
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
        <p className="footer-note">Built for a single active season, weekly reveals, and admin-controlled score entry.</p>
      </main>
    </div>
  )
}

function HomePage() {
  const seasonQuery = useSeasonSummary()
  const fixturesQuery = usePublicFixtures()

  const currentWeek = fixturesQuery.data?.weeks.find((week) => week.week_number === fixturesQuery.data.current_week)
  const futureWeeks = fixturesQuery.data?.weeks.filter((week) => week.status === 'locked') ?? []
  const gamesLeftToPlay =
    fixturesQuery.data?.weeks
      .filter((week) => week.status === 'unlocked')
      .flatMap((week) =>
        week.fixtures
          .filter((fixture) => !fixture.result)
          .map((fixture) => ({
            ...fixture,
            weekNumber: week.week_number,
          })),
      ) ?? []

  return (
    <>
      <section className="hero-panel">
        <div className="hero-copy">
          <span className="eyebrow">Current week unlocked</span>
          <h1>Fixtures with a little theatre.</h1>
          <p className="fixture-meta">{seasonQuery.data?.name ?? 'Active season'}</p>
          <p>
            The public board shows this week in full, keeps future pairings on the radar, and holds the reveal until Monday morning at 09:00 Europe/London.
          </p>
          <div className="hero-actions">
            {seasonQuery.data?.registration_open ? <NavLink to="/register">Join before season start</NavLink> : null}
            <NavLink to="/standings">View points table</NavLink>
          </div>
        </div>

        <div className="hero-side">
          <article className="metric-card">
            <span className="section-eyebrow">League pulse</span>
            <strong>{seasonQuery.data?.player_count ?? '-'}</strong>
            <p>{seasonQuery.data?.registration_open ? 'players registered before the season start action.' : `season live across ${seasonQuery.data?.week_count ?? 0} weeks.`}</p>
          </article>
          {seasonQuery.data && !seasonQuery.data.registration_open ? (
            <article className="metric-card">
              <span className="section-eyebrow">Match format</span>
              <strong>{seasonQuery.data.game_variant}</strong>
              <p>First to {seasonQuery.data.legs_to_win} legs | {seasonQuery.data.games_per_week} game{seasonQuery.data.games_per_week !== 1 ? 's' : ''} per week</p>
            </article>
          ) : null}
          <article className="info-card">
            <div className="card-header">
              <div className="card-copy">
                <h2>Next reveal</h2>
                <p>Future week cards stay visible, with pairings shown and details intentionally obscured.</p>
              </div>
              <span className="status-pill locked">09:00 Monday</span>
            </div>
          </article>
        </div>
      </section>

      <section className="content-panel">
        <div className="card-header">
          <div className="card-copy">
            <span className="section-eyebrow">This week</span>
            <h2>Games left to play</h2>
          </div>
          <span className="status-pill live">{gamesLeftToPlay.length > 0 ? `${gamesLeftToPlay.length} live` : currentWeek ? 'Unlocked' : 'Waiting'}</span>
        </div>
        {fixturesQuery.isLoading ? <StateNotice message="Loading the season board..." /> : null}
        {fixturesQuery.error ? <StateNotice tone="error" message={readError(fixturesQuery.error)} /> : null}
        {!fixturesQuery.isLoading && !fixturesQuery.error && !currentWeek ? (
          <StateNotice message="No public week is unlocked yet. Start the season in admin to generate fixtures." />
        ) : null}
        {!fixturesQuery.isLoading && !fixturesQuery.error && currentWeek && gamesLeftToPlay.length === 0 ? (
          <StateNotice message="Every unlocked fixture has been played so far. Check locked weeks for what is coming next." />
        ) : null}
        {gamesLeftToPlay.length > 0 ? (
          <ul className="match-list">
            {gamesLeftToPlay.map((fixture) => (
              <li key={fixture.id}>
                <div>
                  <strong>{fixture.player_one} vs {fixture.player_two}</strong>
                  <div className="fixture-meta">Week {fixture.weekNumber} - {fixture.game_variant} - First to {fixture.legs_to_win} legs</div>
                </div>
                <div className="fixture-meta">Arrange within the week</div>
              </li>
            ))}
          </ul>
        ) : null}
      </section>

      <section className="week-grid" aria-label="Visible season weeks">
        {futureWeeks.length === 0 && !fixturesQuery.isLoading ? (
          <article className="week-card empty-card">
            <header>
              <div>
                <span className="section-eyebrow">Season map</span>
                <h2>No locked weeks yet</h2>
              </div>
            </header>
            <p>Once the season starts, upcoming weeks appear here with pairings shown and details held back until unlock.</p>
          </article>
        ) : null}
        {futureWeeks.map((week) => (
          <article className="week-card locked" key={week.week_number}>
            <header>
              <div>
                <span className="section-eyebrow">Season map</span>
                <h2>Week {week.week_number}</h2>
              </div>
              <span className="status-pill locked">Locked</span>
            </header>
            <p>Reveals {formatWhen(week.reveal_at)}</p>
            <ul className="match-list">
              {week.fixtures.map((fixture) => (
                <li key={fixture.id}>
                  <strong>{fixture.player_one} vs {fixture.player_two}</strong>
                  <div className="fixture-meta">Pairing visible</div>
                </li>
              ))}
            </ul>
            <div className="lock-overlay">
              <strong>Visible pairings. Hidden details.</strong>
              <span className="fixture-meta">Unlocks automatically</span>
            </div>
          </article>
        ))}
      </section>
    </>
  )
}

function StandingsPage() {
  const seasonQuery = useSeasonSummary()
  const standingsQuery = useStandings()

  return (
    <section className="standings-card">
      <div className="page-intro">
        <span className="eyebrow">Live table</span>
        <h1>Standings</h1>
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
              <th>LF</th>
              <th>LA</th>
              <th>LD</th>
              <th>Pts</th>
            </tr>
          </thead>
          <tbody>
            {standingsQuery.data.map((row) => (
              <tr key={row.display_name}>
                <td className="player-cell">
                  <strong>{row.player}</strong>
                  <span>{row.display_name}</span>
                </td>
                <td>{row.played}</td>
                <td>{row.won}</td>
                <td>{row.lost}</td>
                <td>{row.legs_for}</td>
                <td>{row.legs_against}</td>
                <td>{row.leg_difference}</td>
                <td className="table-emphasis">{row.points}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
    </section>
  )
}

function RegisterPage() {
  const seasonQuery = useSeasonSummary()
  const registerMutation = useRegisterPlayer()
  const [displayName, setDisplayName] = useState('')
  const [nickname, setNickname] = useState('')
  const [successMessage, setSuccessMessage] = useState('')

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSuccessMessage('')
    try {
      const player = await registerMutation.mutateAsync({ display_name: displayName, nickname })
      setSuccessMessage(`${player.preferred_name} is in for the active season.`)
      setDisplayName('')
      setNickname('')
    } catch {
      // handled by mutation error state
    }
  }

  return (
    <>
      <section className="page-intro">
        <span className="eyebrow">Public registration</span>
        <h1>Enter the league</h1>
        <p className="fixture-meta">{seasonQuery.data?.name ?? 'Active season'}</p>
        <p>Registration stays open until the admin starts the season. Display names are unique per season and nicknames stay optional.</p>
      </section>

      {seasonQuery.data && !seasonQuery.data.registration_open ? (
        <section className="register-grid">
          <article className="register-card">
            <h2>Registration closed</h2>
            <p>The active season has already started, so new entries are locked out until the next registration window opens.</p>
            <StateNotice message="Check fixtures and standings to follow the season in progress." compact />
          </article>

          <article className="register-card">
            <h2>Where to go instead</h2>
            <ul className="check-list">
              <li><strong>Fixtures</strong><span className="fixture-meta">See who is due to play this week.</span></li>
              <li><strong>Standings</strong><span className="fixture-meta">Track points, legs for, and league position.</span></li>
              <li><strong>Next season</strong><span className="fixture-meta">Registration will reopen when a new season is created.</span></li>
            </ul>
          </article>
        </section>
      ) : (
        <section className="register-grid">
          <article className="register-card">
            <h2>Player sign-up</h2>
            <p>Keep it fast for MVP: one required name, one optional darts nickname.</p>
            <form className="form-preview" onSubmit={handleSubmit}>
              <div className="field">
                <label htmlFor="display-name">Display name</label>
                <input id="display-name" name="display-name" value={displayName} onChange={(event) => setDisplayName(event.target.value)} placeholder="Luke Humphries" disabled={!seasonQuery.data?.registration_open || registerMutation.isPending} />
              </div>
              <div className="field">
                <label htmlFor="nickname">Nickname</label>
                <input id="nickname" name="nickname" value={nickname} onChange={(event) => setNickname(event.target.value)} placeholder="The Freeze" disabled={!seasonQuery.data?.registration_open || registerMutation.isPending} />
              </div>
              <button type="submit" disabled={!seasonQuery.data?.registration_open || registerMutation.isPending}>{registerMutation.isPending ? 'Registering...' : 'Register for this season'}</button>
            </form>
            {registerMutation.error ? <StateNotice tone="error" message={readError(registerMutation.error)} compact /> : null}
            {successMessage ? <StateNotice message={successMessage} compact /> : null}
            {seasonQuery.data && !seasonQuery.data.registration_open ? <StateNotice message="Registration is now closed for the active season." compact /> : null}
          </article>

          <article className="register-card">
            <h2>What happens next</h2>
            <ul className="check-list">
              <li><strong>Before start</strong><span className="fixture-meta">Admin can review and trim the roster.</span></li>
              <li><strong>On start</strong><span className="fixture-meta">Fixtures generate once and registration closes.</span></li>
              <li><strong>Each week</strong><span className="fixture-meta">Match cards unlock every Monday at 09:00.</span></li>
              <li><strong>Current roster</strong><span className="fixture-meta">{seasonQuery.data?.player_count ?? 0} players in the active season.</span></li>
            </ul>
          </article>
        </section>
      )}
    </>
  )
}

function AdminPage() {
  const seasonQuery = useSeasonSummary()
  const playersQuery = useAdminPlayers()
  const isAuthenticated = playersQuery.isSuccess
  const loginMutation = useAdminLogin()
  const logoutMutation = useAdminLogout()
  const updateSeasonMutation = useUpdateSeason()
  const updateConfigMutation = useUpdateSeasonConfig()
  const seasonStartMutation = useSeasonStart()
  const deletePlayerMutation = useDeletePlayer()
  const undoResultMutation = useUndoResult()
  const fixturesQuery = useAdminFixtures(isAuthenticated)
  const auditQuery = useAuditLog(isAuthenticated)
  const saveResultMutation = useSaveResult()
  const presetsQuery = useGamesPerWeekPresets(isAuthenticated && Boolean(seasonQuery.data?.registration_open))
  const previewQuery = useSchedulePreview(isAuthenticated && Boolean(seasonQuery.data?.registration_open))
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [seasonName, setSeasonName] = useState('')
  const [gameVariant, setGameVariant] = useState('501')
  const [legsToWin, setLegsToWin] = useState('3')
  const [gamesPerWeek, setGamesPerWeek] = useState('1')
  const [showStartConfirm, setShowStartConfirm] = useState(false)

  const unauthenticated = playersQuery.error instanceof ApiError && playersQuery.error.status === 401
  const players = playersQuery.data ?? []

  useEffect(() => {
    setSeasonName(seasonQuery.data?.name ?? '')
  }, [seasonQuery.data?.name])

  useEffect(() => {
    if (seasonQuery.data) {
      setGameVariant(seasonQuery.data.game_variant || '501')
      setLegsToWin(String(seasonQuery.data.legs_to_win || 3))
      setGamesPerWeek(String(seasonQuery.data.games_per_week || 1))
    }
  }, [seasonQuery.data?.game_variant, seasonQuery.data?.legs_to_win, seasonQuery.data?.games_per_week])

  const handleLogin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    await loginMutation.mutateAsync({ username, password })
    setPassword('')
  }

  return (
    <>
      <section className="page-intro">
        <span className="eyebrow">Restricted area</span>
        <h1>Admin control</h1>
        <p>One shared admin account manages player cleanup, season start, and fixed-format result entry across the full schedule.</p>
      </section>

      {!isAuthenticated ? (
        <section className="admin-grid">
          <article className="admin-card">
            <h2>Login</h2>
            <p>The /admin page uses the backend session cookie and only unlocks admin tools after a successful login.</p>
            <form className="admin-login" onSubmit={handleLogin}>
              <div className="field">
                <label htmlFor="admin-user">Username</label>
                <input id="admin-user" name="admin-user" placeholder="admin" value={username} onChange={(event) => setUsername(event.target.value)} />
              </div>
              <div className="field">
                <label htmlFor="admin-pass">Password</label>
                <input id="admin-pass" name="admin-pass" type="password" placeholder="Enter password" value={password} onChange={(event) => setPassword(event.target.value)} />
              </div>
              <button type="submit" disabled={loginMutation.isPending}>Unlock admin tools</button>
            </form>
            {loginMutation.error ? <StateNotice tone="error" message={readError(loginMutation.error)} compact /> : null}
            {playersQuery.error && !unauthenticated ? <StateNotice tone="error" message={readError(playersQuery.error)} compact /> : null}
          </article>

          <article className="admin-card">
            <h2>What unlocks after login</h2>
            <ul className="check-list">
              <li><strong>Roster control</strong><span className="fixture-meta">List and delete players before season start.</span></li>
              <li><strong>Season start</strong><span className="fixture-meta">Generate the schedule once and close registration.</span></li>
              <li><strong>Score entry</strong><span className="fixture-meta">Create or edit first-to-3 results.</span></li>
              <li><strong>Audit feed</strong><span className="fixture-meta">Review before/after result corrections.</span></li>
            </ul>
          </article>
        </section>
      ) : (
        <>
          <section className="admin-toolbar">
            <div className="toolbar-block">
              <strong>{seasonQuery.data?.name ?? 'Active season'}</strong>
              <span className="fixture-meta">{seasonQuery.data?.registration_open ? 'Registration open' : 'Season started'}</span>
            </div>
            <div className="toolbar-actions">
              <button className="secondary-button" type="button" onClick={() => logoutMutation.mutate()} disabled={logoutMutation.isPending}>{logoutMutation.isPending ? 'Logging out...' : 'Log out'}</button>
              <button type="button" onClick={() => setShowStartConfirm(true)} disabled={!seasonQuery.data?.registration_open || seasonStartMutation.isPending || (seasonQuery.data?.player_count ?? 0) < 2}>{seasonStartMutation.isPending ? 'Starting season...' : 'Start season'}</button>
            </div>
          </section>
          {showStartConfirm && previewQuery.data ? (
            <section className="admin-confirm-overlay">
              <article className="admin-card admin-confirm-card">
                <h2>Start Season?</h2>
                <div className="confirm-summary">
                  <p><strong>{previewQuery.data.player_count}</strong> players | <strong>{previewQuery.data.game_variant}</strong> | First to <strong>{previewQuery.data.legs_to_win}</strong> legs</p>
                  <p><strong>{previewQuery.data.games_per_week}</strong> game{previewQuery.data.games_per_week !== 1 ? 's' : ''} per player per week</p>
                  <p className="fixture-meta">{previewQuery.data.week_count} weeks, {previewQuery.data.total_fixtures} total fixtures</p>
                </div>
                <p className="fixture-meta">This cannot be undone.</p>
                <div className="toolbar-actions">
                  <button className="secondary-button" type="button" onClick={() => setShowStartConfirm(false)}>Cancel</button>
                  <button type="button" onClick={() => { seasonStartMutation.mutate(); setShowStartConfirm(false) }} disabled={seasonStartMutation.isPending}>{seasonStartMutation.isPending ? 'Starting...' : 'Start Season'}</button>
                </div>
              </article>
            </section>
          ) : null}
          {seasonStartMutation.error ? <StateNotice tone="error" message={readError(seasonStartMutation.error)} compact /> : null}
          {seasonQuery.data && !seasonQuery.data.registration_open ? <StateNotice message="Registration is locked and player deletion is now disabled for this season." compact /> : null}

          <section className="admin-grid admin-grid-wide">
            <article className="admin-card">
              <h2>League settings</h2>
              <p>Configure the league before the season starts. All settings are locked once fixtures are generated.</p>
              <form
                className="admin-login"
                onSubmit={(event) => {
                  event.preventDefault()
                  updateSeasonMutation.mutate({ name: seasonName })
                }}
              >
                <div className="field">
                  <label htmlFor="season-name">League name</label>
                  <input
                    id="season-name"
                    name="season-name"
                    value={seasonName}
                    onChange={(event) => setSeasonName(event.target.value)}
                    placeholder="Cardiff Premier League"
                    disabled={!seasonQuery.data?.registration_open || updateSeasonMutation.isPending}
                  />
                </div>
                <button type="submit" disabled={!seasonQuery.data?.registration_open || updateSeasonMutation.isPending || seasonName.trim() === (seasonQuery.data?.name ?? '').trim()}>
                  {updateSeasonMutation.isPending ? 'Saving...' : 'Save league name'}
                </button>
              </form>
              {updateSeasonMutation.error ? <StateNotice tone="error" message={readError(updateSeasonMutation.error)} compact /> : null}

              <form
                className="admin-login"
                onSubmit={(event) => {
                  event.preventDefault()
                  updateConfigMutation.mutate({ game_variant: gameVariant, legs_to_win: Number(legsToWin), games_per_week: Number(gamesPerWeek) })
                }}
              >
                <div className="field">
                  <label htmlFor="game-variant">Game variant</label>
                  <select
                    id="game-variant"
                    value={gameVariant}
                    onChange={(event) => setGameVariant(event.target.value)}
                    disabled={!seasonQuery.data?.registration_open || updateConfigMutation.isPending}
                  >
                    <option value="501">501</option>
                    <option value="301">301</option>
                  </select>
                </div>
                <div className="field">
                  <label htmlFor="legs-to-win">First to (legs)</label>
                  <input
                    id="legs-to-win"
                    name="legs-to-win"
                    type="number"
                    min="1"
                    value={legsToWin}
                    onChange={(event) => setLegsToWin(event.target.value)}
                    disabled={!seasonQuery.data?.registration_open || updateConfigMutation.isPending}
                  />
                </div>
                <div className="field">
                  <label htmlFor="games-per-week">Games per player per week</label>
                  <select
                    id="games-per-week"
                    value={gamesPerWeek}
                    onChange={(event) => setGamesPerWeek(event.target.value)}
                    disabled={!seasonQuery.data?.registration_open || updateConfigMutation.isPending || !presetsQuery.data}
                  >
                    {presetsQuery.data?.map((preset) => (
                      <option key={preset.games_per_week} value={String(preset.games_per_week)}>
                        {preset.games_per_week} game{preset.games_per_week !== 1 ? 's' : ''}/week ({preset.week_count} week{preset.week_count !== 1 ? 's' : ''})
                      </option>
                    ))}
                    {(!presetsQuery.data || presetsQuery.data.length === 0) ? <option value={gamesPerWeek}>{gamesPerWeek} game{Number(gamesPerWeek) !== 1 ? 's' : ''}/week</option> : null}
                  </select>
                </div>
                <button type="submit" disabled={
                  !seasonQuery.data?.registration_open ||
                  updateConfigMutation.isPending ||
                  (gameVariant === (seasonQuery.data?.game_variant ?? '501') && Number(legsToWin) === (seasonQuery.data?.legs_to_win ?? 3) && Number(gamesPerWeek) === (seasonQuery.data?.games_per_week ?? 1))
                }>
                  {updateConfigMutation.isPending ? 'Saving...' : 'Save match config'}
                </button>
              </form>
              {updateConfigMutation.error ? <StateNotice tone="error" message={readError(updateConfigMutation.error)} compact /> : null}
              {seasonQuery.data && !seasonQuery.data.registration_open ? <StateNotice message="All settings are locked once the season has started." compact /> : null}
            </article>

            <article className="admin-card">
              <h2>Registered players</h2>
              {playersQuery.isLoading ? <StateNotice message="Loading admin roster..." compact /> : null}
              {players.length === 0 ? <StateNotice message="No registered players yet." compact /> : null}
              {players.length > 0 ? <PlayerRoster players={players} registrationOpen={Boolean(seasonQuery.data?.registration_open)} onDelete={(playerId) => deletePlayerMutation.mutateAsync(playerId)} isDeleting={deletePlayerMutation.isPending} /> : null}
              {deletePlayerMutation.error ? <StateNotice tone="error" message={readError(deletePlayerMutation.error)} compact /> : null}
            </article>

            <article className="admin-card">
              <h2>Full season fixtures</h2>
              {fixturesQuery.isLoading ? <StateNotice message="Loading full season schedule..." compact /> : null}
              {fixturesQuery.error ? <StateNotice tone="error" message={readError(fixturesQuery.error)} compact /> : null}
              {fixturesQuery.data?.length === 0 ? <StateNotice message="Start the season to generate fixtures." compact /> : null}
              {fixturesQuery.data?.map((week) => (
                <div className="admin-week" key={week.week_number}>
                  <div className="admin-week-header">
                    <strong>Week {week.week_number}</strong>
                    <span className="fixture-meta">Reveals {formatWhen(week.reveal_at)}</span>
                  </div>
                  <div className="admin-fixtures">
                    {week.fixtures.map((fixture) => (
                      <AdminFixtureCard key={fixture.id} fixture={fixture} onSave={(payload) => saveResultMutation.mutateAsync(payload)} onUndo={(fixtureId) => undoResultMutation.mutateAsync(fixtureId)} isSaving={saveResultMutation.isPending} isUndoing={undoResultMutation.isPending} />
                    ))}
                  </div>
                </div>
              ))}
              {saveResultMutation.error ? <StateNotice tone="error" message={readError(saveResultMutation.error)} compact /> : null}
              {undoResultMutation.error ? <StateNotice tone="error" message={readError(undoResultMutation.error)} compact /> : null}
            </article>

            <article className="admin-card">
              <h2>Audit trail</h2>
              {auditQuery.isLoading ? <StateNotice message="Loading audit trail..." compact /> : null}
              {auditQuery.data?.length === 0 ? <StateNotice message="Result edits will appear here." compact /> : null}
              {auditQuery.data?.map((entry) => <AuditEntryCard entry={entry} key={entry.id} />)}
            </article>
          </section>
        </>
      )}
    </>
  )
}

function PlayerRoster({ players, registrationOpen, onDelete, isDeleting }: { players: Player[]; registrationOpen: boolean; onDelete: (playerId: number) => Promise<unknown>; isDeleting: boolean }) {
  return (
    <ul className="check-list">
      {players.map((player) => (
        <li key={player.id}>
          <div>
            <strong>{player.admin_label}</strong>
            <div className="fixture-meta">Registered {player.registered_at ? formatWhen(player.registered_at) : 'recently'}</div>
          </div>
          {registrationOpen ? <button className="ghost-button" type="button" onClick={() => onDelete(player.id)} disabled={isDeleting}>{isDeleting ? 'Deleting...' : 'Delete'}</button> : <span className="fixture-meta">Roster locked</span>}
        </li>
      ))}
    </ul>
  )
}

function AdminFixtureCard({ fixture, onSave, onUndo, isSaving, isUndoing }: { fixture: AdminFixture; onSave: (payload: { fixtureId: number; playerOneLegs: number; playerTwoLegs: number; playerOneAverage?: number; playerTwoAverage?: number }) => Promise<unknown>; onUndo: (fixtureId: number) => Promise<unknown>; isSaving: boolean; isUndoing: boolean }) {
  const [playerOneLegs, setPlayerOneLegs] = useState(String(fixture.result?.player_one_legs ?? fixture.legs_to_win))
  const [playerTwoLegs, setPlayerTwoLegs] = useState(String(fixture.result?.player_two_legs ?? 0))
  const [playerOneAverage, setPlayerOneAverage] = useState(formatAverage(fixture.result?.player_one_average))
  const [playerTwoAverage, setPlayerTwoAverage] = useState(formatAverage(fixture.result?.player_two_average))
  const [statusMessage, setStatusMessage] = useState('')

  const playerOneLegsValue = Number(playerOneLegs)
  const playerTwoLegsValue = Number(playerTwoLegs)
  const legsToWin = fixture.legs_to_win
  const bestOfLegs = (legsToWin * 2) - 1
  const isValidScoreline = Number.isInteger(playerOneLegsValue) && Number.isInteger(playerTwoLegsValue) && (
    (playerOneLegsValue === legsToWin && playerTwoLegsValue >= 0 && playerTwoLegsValue < legsToWin) ||
    (playerTwoLegsValue === legsToWin && playerOneLegsValue >= 0 && playerOneLegsValue < legsToWin)
  )
  const scorelineHint = `Valid scores: ${legsToWin}-0 to ${legsToWin}-${Math.max(0, legsToWin - 1)} (or reversed).`

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setStatusMessage('')
    if (!isValidScoreline) {
      setStatusMessage(`Enter a valid first-to-${legsToWin} score. ${scorelineHint}`)
      return
    }
    await onSave({
      fixtureId: fixture.id,
      playerOneLegs: playerOneLegsValue,
      playerTwoLegs: playerTwoLegsValue,
      playerOneAverage: playerOneAverage === '' ? undefined : Number(playerOneAverage),
      playerTwoAverage: playerTwoAverage === '' ? undefined : Number(playerTwoAverage),
    })
    setStatusMessage('Score saved.')
  }

  const handleUndo = async () => {
    setStatusMessage('')
    await onUndo(fixture.id)
    setStatusMessage('Recorded result removed.')
  }

  return (
    <article className="admin-fixture-card">
      <div className="admin-week-header">
        <div>
          <strong>{fixture.player_one} vs {fixture.player_two}</strong>
          <div className="fixture-meta">{fixture.game_variant} first to {fixture.legs_to_win} - players arrange within the week</div>
        </div>
        <span className={`status-pill ${fixture.result ? 'live' : 'locked'}`}>{fixture.result ? `Recorded ${fixture.result.player_one_legs}-${fixture.result.player_two_legs}` : 'No result yet'}</span>
      </div>
      <p className="score-rule-badge">Scoring rule: first to {legsToWin} (best of {bestOfLegs}).</p>
      <form className="score-form" onSubmit={handleSubmit}>
        <div className="score-field">
          <label htmlFor={`p1-${fixture.id}`}>{fixture.player_one} legs</label>
          <input id={`p1-${fixture.id}`} type="number" min={0} max={legsToWin} step={1} value={playerOneLegs} onChange={(event) => setPlayerOneLegs(event.target.value)} inputMode="numeric" />
        </div>
        <div className="score-field">
          <label htmlFor={`p2-${fixture.id}`}>{fixture.player_two} legs</label>
          <input id={`p2-${fixture.id}`} type="number" min={0} max={legsToWin} step={1} value={playerTwoLegs} onChange={(event) => setPlayerTwoLegs(event.target.value)} inputMode="numeric" />
        </div>
        <div className="score-field">
          <label htmlFor={`a1-${fixture.id}`}>{fixture.player_one} avg</label>
          <input id={`a1-${fixture.id}`} value={playerOneAverage} onChange={(event) => setPlayerOneAverage(event.target.value)} inputMode="decimal" placeholder="95.4" />
        </div>
        <div className="score-field">
          <label htmlFor={`a2-${fixture.id}`}>{fixture.player_two} avg</label>
          <input id={`a2-${fixture.id}`} value={playerTwoAverage} onChange={(event) => setPlayerTwoAverage(event.target.value)} inputMode="decimal" placeholder="88.2" />
        </div>
        <button type="submit" disabled={isSaving || !isValidScoreline}>{isSaving ? 'Saving...' : 'Save score'}</button>
        {fixture.result ? <button className="secondary-button" type="button" onClick={handleUndo} disabled={isUndoing}>{isUndoing ? 'Undoing...' : 'Undo result'}</button> : null}
      </form>
      {!isValidScoreline ? <p className="fixture-meta">{scorelineHint}</p> : null}
      {statusMessage ? <p className="fixture-meta">{statusMessage}</p> : null}
      {fixture.result?.player_one_average !== undefined || fixture.result?.player_two_average !== undefined ? (
        <p className="fixture-meta">Averages: {fixture.player_one} {formatAverage(fixture.result?.player_one_average) || '-'} / {fixture.player_two} {formatAverage(fixture.result?.player_two_average) || '-'}</p>
      ) : null}
    </article>
  )
}

function AuditEntryCard({ entry }: { entry: AuditEntry }) {
  const scoreDelta = useMemo(() => {
    if (!entry.old_result || !entry.new_result) {
      return 'Change recorded.'
    }
    const oldAverages = entry.old_result.player_one_average !== undefined || entry.old_result.player_two_average !== undefined
      ? ` (${formatAverage(entry.old_result.player_one_average)} / ${formatAverage(entry.old_result.player_two_average)})`
      : ''
    const newAverages = entry.new_result.player_one_average !== undefined || entry.new_result.player_two_average !== undefined
      ? ` (${formatAverage(entry.new_result.player_one_average)} / ${formatAverage(entry.new_result.player_two_average)})`
      : ''
    return `${entry.old_result.player_one_legs}-${entry.old_result.player_two_legs}${oldAverages} -> ${entry.new_result.player_one_legs}-${entry.new_result.player_two_legs}${newAverages}`
  }, [entry])

  return (
    <div className="audit-entry">
      <strong>{entry.action.replace(/_/g, ' ')}</strong>
      <div className="fixture-meta">{entry.fixture_label || `Fixture #${entry.fixture_id}`} - {scoreDelta}</div>
      <div className="fixture-meta">By {entry.actor} at {formatWhen(entry.created_at)}</div>
    </div>
  )
}

function StateNotice({ message, tone = 'neutral', compact = false }: { message: string; tone?: 'neutral' | 'error'; compact?: boolean }) {
  return <p className={`state-notice ${tone}${compact ? ' compact' : ''}`}>{message}</p>
}

function readError(error: unknown) {
  if (error instanceof ApiError) {
    return error.message
  }
  if (error instanceof Error) {
    return error.message
  }
  return 'Something went wrong.'
}

export default App
