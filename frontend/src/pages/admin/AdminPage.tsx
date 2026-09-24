import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router-dom'
import {
  ApiError,
  useAdminDivisions,
  useAdminLogin,
  useAdminPlayers,
  useAssignPlayer,
  useDeletePlayer,
  useGamesPerWeekPresets,
  useProvisionDivisions,
  useSchedulePreview,
  useSeasonStart,
  useSeasonSummary,
  useUpdateDivision,
  useUpdateSeason,
  useUpdateSeasonConfig,
} from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'
import { PlayerRoster } from './PlayerRoster'

export function AdminPage() {
  const seasonQuery = useSeasonSummary()
  const playersQuery = useAdminPlayers()
  const isAuthenticated = playersQuery.isSuccess
  const divisionsQuery = useAdminDivisions(isAuthenticated)
  const loginMutation = useAdminLogin()
  const updateSeasonMutation = useUpdateSeason()
  const updateConfigMutation = useUpdateSeasonConfig()
  const provisionDivisionsMutation = useProvisionDivisions()
  const updateDivisionMutation = useUpdateDivision()
  const assignPlayerMutation = useAssignPlayer()
  const seasonStartMutation = useSeasonStart()
  const deletePlayerMutation = useDeletePlayer()
  const presetsQuery = useGamesPerWeekPresets(isAuthenticated && Boolean(seasonQuery.data?.can_edit_settings))
  const previewQuery = useSchedulePreview(isAuthenticated && Boolean(seasonQuery.data?.can_edit_settings))

  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [seasonName, setSeasonName] = useState('')
  const [gameVariant, setGameVariant] = useState('501')
  const [legsToWin, setLegsToWin] = useState('3')
  const [gamesPerWeek, setGamesPerWeek] = useState('1')
  const [divisionCount, setDivisionCount] = useState('2')

  useEffect(() => {
    setSeasonName(seasonQuery.data?.name ?? '')
  }, [seasonQuery.data?.name])

  useEffect(() => {
    if (!seasonQuery.data) return
    setGameVariant(seasonQuery.data.game_variant || '501')
    setLegsToWin(String(seasonQuery.data.legs_to_win || 3))
    setGamesPerWeek(String(seasonQuery.data.games_per_week || 1))
  }, [seasonQuery.data?.game_variant, seasonQuery.data?.legs_to_win, seasonQuery.data?.games_per_week])

  const handleLogin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    await loginMutation.mutateAsync({ username, password })
    setPassword('')
  }

  const unauthenticated = playersQuery.error instanceof ApiError && playersQuery.error.status === 401

  return (
    <>
      <section className="page-intro">
        <span className="eyebrow">Restricted area</span>
        <h1>Admin control</h1>
        <p>Use one shared password to shape divisions, assign players, start the season, and jump into division-specific scoring pages.</p>
      </section>

      {!isAuthenticated ? (
        <section className="admin-grid">
          <article className="admin-card">
            <h2>Login</h2>
            <p>The /admin page uses the backend session cookie and only unlocks admin tools after a successful login.</p>
            <form className="admin-login" onSubmit={handleLogin}>
              <div className="field">
                <label htmlFor="admin-user">Username</label>
                <input id="admin-user" value={username} onChange={(event) => setUsername(event.target.value)} />
              </div>
              <div className="field">
                <label htmlFor="admin-pass">Password</label>
                <input id="admin-pass" type="password" value={password} onChange={(event) => setPassword(event.target.value)} />
              </div>
              <button type="submit" disabled={loginMutation.isPending}>{loginMutation.isPending ? 'Unlocking...' : 'Unlock admin tools'}</button>
            </form>
            {loginMutation.error ? <StateNotice tone="error" message={readError(loginMutation.error)} compact /> : null}
            {playersQuery.error && !unauthenticated ? <StateNotice tone="error" message={readError(playersQuery.error)} compact /> : null}
          </article>

          <article className="admin-card">
            <h2>What unlocks after login</h2>
            <ul className="check-list">
              <li><strong>Season config</strong><span className="fixture-meta">Shared format and naming across every division.</span></li>
              <li><strong>Division setup</strong><span className="fixture-meta">Provision division count, name each board, and wire Slack channels.</span></li>
              <li><strong>Roster assignment</strong><span className="fixture-meta">Place players into one division or leave them waitlisted.</span></li>
              <li><strong>Division scoring</strong><span className="fixture-meta">Open a dedicated admin page for each division once fixtures exist.</span></li>
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
              {seasonQuery.data?.can_start_season ? (
                <button type="button" onClick={() => seasonStartMutation.mutate()} disabled={seasonStartMutation.isPending || (seasonQuery.data?.assigned_count ?? 0) < 2}>{seasonStartMutation.isPending ? 'Starting season...' : 'Start season'}</button>
              ) : null}
            </div>
          </section>

          {seasonStartMutation.error ? <StateNotice tone="error" message={readError(seasonStartMutation.error)} compact /> : null}
          {seasonQuery.data?.season_started && !seasonQuery.data?.admin_locked ? <StateNotice message="Registration is locked. League settings and Slack channels stay editable until the first week is released." compact /> : null}
          {seasonQuery.data?.admin_locked ? <StateNotice message="The first week is live. Central season setup is now locked and only division scoring pages remain editable." compact /> : null}

          <section className="admin-grid admin-grid-wide">
            <article className="admin-card">
              <h2>League settings</h2>
              <p>Configure the league before kickoff. Match rules apply to every division and stay editable until the first week is released.</p>
              <form
                className="admin-login"
                onSubmit={(event) => {
                  event.preventDefault()
                  if (seasonName.trim() !== (seasonQuery.data?.name ?? '').trim()) {
                    updateSeasonMutation.mutate({ name: seasonName })
                  }
                  if (gameVariant !== (seasonQuery.data?.game_variant ?? '501') || Number(legsToWin) !== (seasonQuery.data?.legs_to_win ?? 3) || Number(gamesPerWeek) !== (seasonQuery.data?.games_per_week ?? 1)) {
                    updateConfigMutation.mutate({ game_variant: gameVariant, legs_to_win: Number(legsToWin), games_per_week: Number(gamesPerWeek) })
                  }
                }}
              >
                <div className="field">
                  <label htmlFor="season-name">League name</label>
                  <input id="season-name" value={seasonName} onChange={(event) => setSeasonName(event.target.value)} disabled={!seasonQuery.data?.can_edit_settings} />
                </div>
                <div className="field">
                  <label htmlFor="game-variant">Game variant</label>
                  <select id="game-variant" value={gameVariant} onChange={(event) => setGameVariant(event.target.value)} disabled={!seasonQuery.data?.can_edit_settings}>
                    <option value="501">501</option>
                    <option value="301">301</option>
                  </select>
                </div>
                <div className="field">
                  <label htmlFor="legs-to-win">First to (legs)</label>
                  <input id="legs-to-win" type="number" min="1" value={legsToWin} onChange={(event) => setLegsToWin(event.target.value)} disabled={!seasonQuery.data?.can_edit_settings} />
                </div>
                <div className="field">
                  <label htmlFor="games-per-week">Games per player per week</label>
                  <select id="games-per-week" value={gamesPerWeek} onChange={(event) => setGamesPerWeek(event.target.value)} disabled={!seasonQuery.data?.can_edit_settings}>
                    {(presetsQuery.data && presetsQuery.data.length > 0 ? presetsQuery.data : [{ games_per_week: Number(gamesPerWeek), week_count: previewQuery.data?.week_count ?? 0 }]).map((preset) => (
                      <option key={preset.games_per_week} value={String(preset.games_per_week)}>{preset.games_per_week} game{preset.games_per_week !== 1 ? 's' : ''}/week</option>
                    ))}
                  </select>
                </div>
                {seasonQuery.data?.can_edit_settings ? (
                  <button type="submit" disabled={updateSeasonMutation.isPending || updateConfigMutation.isPending}>{updateSeasonMutation.isPending || updateConfigMutation.isPending ? 'Saving...' : 'Save config'}</button>
                ) : null}
              </form>
              {updateSeasonMutation.error ? <StateNotice tone="error" message={readError(updateSeasonMutation.error)} compact /> : null}
              {updateConfigMutation.error ? <StateNotice tone="error" message={readError(updateConfigMutation.error)} compact /> : null}
            </article>

            <article className="admin-card">
              <h2>Divisions</h2>
              <p>Choose a count and rename divisions until the first week is released. Reprovisioning divisions resets all assignments back to the waitlist.</p>
              <form className="admin-login" onSubmit={(event) => { event.preventDefault(); provisionDivisionsMutation.mutate(Number(divisionCount)) }}>
                <div className="field">
                  <label htmlFor="division-count">Division count</label>
                  <input id="division-count" type="number" min="1" value={divisionCount} onChange={(event) => setDivisionCount(event.target.value)} disabled={!seasonQuery.data?.can_edit_divisions} />
                </div>
                {seasonQuery.data?.can_edit_divisions ? (
                  <button type="submit" disabled={provisionDivisionsMutation.isPending}>{provisionDivisionsMutation.isPending ? 'Creating...' : 'Create divisions'}</button>
                ) : null}
              </form>
              {provisionDivisionsMutation.error ? <StateNotice tone="error" message={readError(provisionDivisionsMutation.error)} compact /> : null}
              {divisionsQuery.data?.map((division) => (
                <form
                  key={division.id}
                  className="admin-login"
                  onSubmit={(event) => {
                    event.preventDefault()
                    const formData = new FormData(event.currentTarget)
                    updateDivisionMutation.mutate({
                      id: division.id,
                      name: String(formData.get(`division-name-${division.id}`) ?? division.name),
                      slug: division.slug,
                      slack_public_channel_id: String(formData.get(`division-slack-${division.id}`) ?? division.slack_public_channel_id ?? ''),
                    })
                  }}
                >
                  <div className="field">
                    <label htmlFor={`division-name-${division.id}`}>Division name</label>
                      <input id={`division-name-${division.id}`} name={`division-name-${division.id}`} defaultValue={division.name} disabled={!seasonQuery.data?.can_edit_divisions} />
                    </div>
                    <div className="field">
                      <label htmlFor={`division-slack-${division.id}`}>Slack public channel</label>
                      <input id={`division-slack-${division.id}`} name={`division-slack-${division.id}`} defaultValue={division.slack_public_channel_id ?? ''} disabled={!seasonQuery.data?.can_edit_division_channel} />
                    </div>
                    <div className="toolbar-actions">
                     {seasonQuery.data?.can_edit_divisions ? (
                       <button type="submit" disabled={updateDivisionMutation.isPending}>{updateDivisionMutation.isPending ? 'Saving...' : 'Save division'}</button>
                     ) : null}
                     {seasonQuery.data?.season_started && !seasonQuery.data?.admin_locked && !seasonQuery.data?.can_edit_divisions ? (
                       <button type="submit" disabled={updateDivisionMutation.isPending}>{updateDivisionMutation.isPending ? 'Saving...' : 'Save channel'}</button>
                     ) : null}
                     <Link className="button-link secondary-button" to={`/admin/divisions/${division.slug}`}>Open scoring page</Link>
                   </div>
                 </form>
              ))}
              {updateDivisionMutation.error ? <StateNotice tone="error" message={readError(updateDivisionMutation.error)} compact /> : null}
            </article>

            <article className="admin-card">
              <h2>Registered players</h2>
              <p>Assign each player to one division or leave them waitlisted and excluded from the season schedule.</p>
              {playersQuery.isLoading ? <StateNotice message="Loading admin roster..." compact /> : null}
              {playersQuery.data && playersQuery.data.length > 0 ? (
                <PlayerRoster
                  players={playersQuery.data}
                  divisions={divisionsQuery.data ?? []}
                  canAssign={Boolean(seasonQuery.data?.can_assign_players)}
                  canDelete={Boolean(seasonQuery.data?.registration_open)}
                  onDelete={(playerId) => deletePlayerMutation.mutateAsync(playerId)}
                  onAssign={(playerId, divisionId) => assignPlayerMutation.mutateAsync({ playerId, divisionId })}
                  isDeleting={deletePlayerMutation.isPending}
                  isAssigning={assignPlayerMutation.isPending}
                />
              ) : null}
              {deletePlayerMutation.error ? <StateNotice tone="error" message={readError(deletePlayerMutation.error)} compact /> : null}
              {assignPlayerMutation.error ? <StateNotice tone="error" message={readError(assignPlayerMutation.error)} compact /> : null}
            </article>
          </section>
        </>
      )}
    </>
  )
}
