import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import {
  ApiError,
  formatWhen,
  useAdminFixtures,
  usePublicFixtures,
  useAdminLogin,
  useAdminLogout,
  useAdminPlayers,
  useAuditLog,
  useDeletePlayer,
  useGamesPerWeekPresets,
  useSaveResult,
  useSchedulePreview,
  useSeasonSummary,
  useSeasonStart,
  useUndoResult,
  useUpdateSeason,
  useUpdateSeasonConfig,
} from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'
import { PlayerRoster } from './PlayerRoster'
import { AdminFixtureCard } from './AdminFixtureCard'
import { AuditEntryCard } from './AuditEntryCard'

export function AdminPage() {
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
  const publicFixturesQuery = usePublicFixtures()
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
  const [settingsOpen, setSettingsOpen] = useState(true)
  const [playersOpen, setPlayersOpen] = useState(true)
  const [auditOpen, setAuditOpen] = useState(false)
  const cardSectionsInitialised = useRef(false)
  const [collapsedWeeks, setCollapsedWeeks] = useState<Set<number>>(new Set())
  const weeksInitialised = useRef(false)

  useEffect(() => {
    if (weeksInitialised.current) return
    const weeks = fixturesQuery.data
    const currentWeek = publicFixturesQuery.data?.current_week
    if (!weeks || currentWeek === undefined) return
    const collapsed = new Set<number>()
    for (const week of weeks) {
      const isPast = week.week_number < currentWeek
      const isFuture = week.week_number > currentWeek
      const allDone = week.fixtures.length > 0 && week.fixtures.every((f) => f.result)
      if ((isPast && allDone) || isFuture) collapsed.add(week.week_number)
    }
    setCollapsedWeeks(collapsed)
    weeksInitialised.current = true
  }, [fixturesQuery.data, publicFixturesQuery.data?.current_week])

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

  useEffect(() => {
    if (cardSectionsInitialised.current || !seasonQuery.data) return
    const registrationOpen = seasonQuery.data.registration_open
    setSettingsOpen(registrationOpen)
    setPlayersOpen(registrationOpen)
    cardSectionsInitialised.current = true
  }, [seasonQuery.data])

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
              <button type="button" className="admin-card-toggle" onClick={() => setSettingsOpen((o) => !o)} aria-expanded={settingsOpen}>
                <h2>League settings</h2>
                <span className={`admin-week-chevron${settingsOpen ? ' admin-week-chevron--open' : ''}`} aria-hidden="true">›</span>
              </button>
              {settingsOpen && (<>
              <p>Configure the league before the season starts. All settings are locked once fixtures are generated.</p>
              <form
                className="admin-login"
                onSubmit={(event) => {
                  event.preventDefault()
                  const nameChanged = seasonName.trim() !== (seasonQuery.data?.name ?? '').trim()
                  const configChanged = gameVariant !== (seasonQuery.data?.game_variant ?? '501') || Number(legsToWin) !== (seasonQuery.data?.legs_to_win ?? 3) || Number(gamesPerWeek) !== (seasonQuery.data?.games_per_week ?? 1)
                  if (nameChanged) updateSeasonMutation.mutate({ name: seasonName })
                  if (configChanged) updateConfigMutation.mutate({ game_variant: gameVariant, legs_to_win: Number(legsToWin), games_per_week: Number(gamesPerWeek) })
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
                  updateSeasonMutation.isPending ||
                  updateConfigMutation.isPending ||
                  (seasonName.trim() === (seasonQuery.data?.name ?? '').trim() &&
                    gameVariant === (seasonQuery.data?.game_variant ?? '501') &&
                    Number(legsToWin) === (seasonQuery.data?.legs_to_win ?? 3) &&
                    Number(gamesPerWeek) === (seasonQuery.data?.games_per_week ?? 1))
                }>
                  {updateSeasonMutation.isPending || updateConfigMutation.isPending ? 'Saving...' : 'Save config'}
                </button>
              </form>
              {updateSeasonMutation.error ? <StateNotice tone="error" message={readError(updateSeasonMutation.error)} compact /> : null}
              {updateConfigMutation.error ? <StateNotice tone="error" message={readError(updateConfigMutation.error)} compact /> : null}
              {seasonQuery.data && !seasonQuery.data.registration_open ? <StateNotice message="All settings are locked once the season has started." compact /> : null}
              </>)}
            </article>

            <article className="admin-card">
              <button type="button" className="admin-card-toggle" onClick={() => setPlayersOpen((o) => !o)} aria-expanded={playersOpen}>
                <h2>Registered players</h2>
                <span className={`admin-week-chevron${playersOpen ? ' admin-week-chevron--open' : ''}`} aria-hidden="true">›</span>
              </button>
              {playersOpen && (<>
              {playersQuery.isLoading ? <StateNotice message="Loading admin roster..." compact /> : null}
              {players.length === 0 ? <StateNotice message="No registered players yet." compact /> : null}
              {players.length > 0 ? <PlayerRoster players={players} registrationOpen={Boolean(seasonQuery.data?.registration_open)} onDelete={(playerId) => deletePlayerMutation.mutateAsync(playerId)} isDeleting={deletePlayerMutation.isPending} /> : null}
              {deletePlayerMutation.error ? <StateNotice tone="error" message={readError(deletePlayerMutation.error)} compact /> : null}
              </>)}
            </article>

            <article className="admin-card">
              <h2>Full season fixtures</h2>
              {seasonQuery.data && !seasonQuery.data.registration_open ? (
                <p className="fixture-meta season-format-summary">
                  {seasonQuery.data.game_variant} &middot; First to {seasonQuery.data.legs_to_win} legs &middot; {seasonQuery.data.games_per_week} game{seasonQuery.data.games_per_week !== 1 ? 's' : ''} per week
                </p>
              ) : null}
              {fixturesQuery.isLoading ? <StateNotice message="Loading full season schedule..." compact /> : null}
              {fixturesQuery.error ? <StateNotice tone="error" message={readError(fixturesQuery.error)} compact /> : null}
              {fixturesQuery.data?.length === 0 ? <StateNotice message="Start the season to generate fixtures." compact /> : null}
              {(fixturesQuery.data ?? []).map((week) => {
                const total = week.fixtures.length
                const complete = week.fixtures.filter((f) => f.result).length
                const outstanding = total - complete
                const allDone = complete === total && total > 0
                const isPastWeek = week.week_number < (publicFixturesQuery.data?.current_week ?? 0)
                const isCollapsed = collapsedWeeks.has(week.week_number)
                const toggleCollapse = () => setCollapsedWeeks((prev) => {
                  const next = new Set(prev)
                  if (next.has(week.week_number)) next.delete(week.week_number)
                  else next.add(week.week_number)
                  return next
                })
                return (
                  <div className="admin-week" key={week.week_number}>
                    <button type="button" className="admin-week-header admin-week-toggle" onClick={toggleCollapse} aria-expanded={!isCollapsed}>
                      <span className="admin-week-title-group">
                        <span className={`week-progress-badge admin-week-chevron${isCollapsed ? '' : ' admin-week-chevron--open'}`} aria-hidden="true">›</span>
                        <strong className="admin-week-title">Week {week.week_number}</strong>
                        <span className={`week-progress-badge${allDone ? ' week-progress-badge--done' : isPastWeek && outstanding > 0 ? ' week-progress-badge--overdue' : ''}`}>
                          {allDone ? '✓ Completed!' : isPastWeek && outstanding > 0 ? `${outstanding}/${total} Outstanding` : `${complete}/${total} Complete`}
                        </span>
                      </span>
                      <span className="admin-week-header-right">
                        <span className="fixture-meta">
                          {week.week_number < (publicFixturesQuery.data?.current_week ?? 0)
                            ? `Revealed ${formatWhen(week.reveal_at)}`
                            : week.week_number === (publicFixturesQuery.data?.current_week ?? 0)
                              ? 'In Progress'
                              : `Reveals ${formatWhen(week.reveal_at)}`}
                        </span>
                      </span>
                    </button>
                    {!isCollapsed && (
                      <div className="admin-fixtures">
                        {week.fixtures.map((fixture) => (
                          <AdminFixtureCard key={fixture.id} fixture={fixture} onSave={(payload) => saveResultMutation.mutateAsync(payload)} onUndo={(fixtureId) => undoResultMutation.mutateAsync(fixtureId)} isSaving={saveResultMutation.isPending} isUndoing={undoResultMutation.isPending} isLocked={week.week_number > (publicFixturesQuery.data?.current_week ?? 0)} isPastWeek={isPastWeek} />
                        ))}
                      </div>
                    )}
                  </div>
                )
              })}
              {saveResultMutation.error ? <StateNotice tone="error" message={readError(saveResultMutation.error)} compact /> : null}
              {undoResultMutation.error ? <StateNotice tone="error" message={readError(undoResultMutation.error)} compact /> : null}
            </article>

            <article className="admin-card">
              <button type="button" className="admin-card-toggle" onClick={() => setAuditOpen((o) => !o)} aria-expanded={auditOpen}>
                <h2>Audit trail</h2>
                <span className={`admin-week-chevron${auditOpen ? ' admin-week-chevron--open' : ''}`} aria-hidden="true">›</span>
              </button>
              {auditOpen && (<>
              {auditQuery.isLoading ? <StateNotice message="Loading audit trail..." compact /> : null}
              {auditQuery.data?.length === 0 ? <StateNotice message="Result edits will appear here." compact /> : null}
              {auditQuery.data?.map((entry) => <AuditEntryCard entry={entry} key={entry.id} />)}
              </>)}
            </article>
          </section>
        </>
      )}
    </>
  )
}
