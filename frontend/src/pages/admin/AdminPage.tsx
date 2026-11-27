import { useState } from 'react'
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
import { SeasonLifecycle } from './SeasonLifecycle'
import { AutodartsMappings } from './AutodartsMappings'

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

  const settingsKey = JSON.stringify([seasonQuery.data?.id, seasonQuery.data?.name, seasonQuery.data?.game_variant, seasonQuery.data?.legs_to_win, seasonQuery.data?.games_per_week])
  const [previousSettingsKey, setPreviousSettingsKey] = useState('')
  if (settingsKey !== previousSettingsKey) {
    setPreviousSettingsKey(settingsKey)
    setSeasonName(seasonQuery.data?.name ?? '')
    setGameVariant(seasonQuery.data?.game_variant ?? '501')
    setLegsToWin(String(seasonQuery.data?.legs_to_win ?? 3))
    setGamesPerWeek(String(seasonQuery.data?.games_per_week ?? 1))
  }

  const handleLogin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    await loginMutation.mutateAsync({ username, password })
    setPassword('')
  }

  const unauthenticated = playersQuery.error instanceof ApiError && playersQuery.error.status === 401
  const needsPlayers = seasonQuery.data?.registration_open === true && seasonQuery.data.player_count === 0

  const season = seasonQuery.data

  const roster = playersQuery.data && playersQuery.data.length > 0 ? (
    <PlayerRoster
      players={playersQuery.data}
      divisions={divisionsQuery.data ?? []}
      canAssign={Boolean(season?.can_assign_players)}
      canDelete={Boolean(season?.registration_open)}
      onDelete={playerId => deletePlayerMutation.mutateAsync(playerId)}
      onAssign={(playerId, divisionId) => assignPlayerMutation.mutateAsync({ playerId, divisionId })}
      isDeleting={deletePlayerMutation.isPending}
      isAssigning={assignPlayerMutation.isPending}
    />
  ) : null

  return (
    <div className="admin-workspace">
      <section className={isAuthenticated ? 'hero-panel competition-hero competition-hero--league' : 'hero-panel competition-hero competition-hero--league competition-hero--compact'}>
        <div className="hero-copy">
          <span className="competition-kicker">Administration</span>
          <h1>Admin control</h1>
          {isAuthenticated ? (
            <>
              <p><strong>{season?.name ?? 'Active season'}</strong></p>
              <p>{season?.status === 'completed' ? 'Season completed' : season?.registration_open ? 'Registration open' : season?.status === 'started' ? 'Season started' : null}</p>
              {needsPlayers ? <p className="fixture-meta" id="season-start-help">No players registered yet. Register at least two players and assign them to the same division before starting the season.</p> : null}
              <div className="toolbar-actions admin-hero-actions">
                {season?.can_start_season ? (
                  <button className="season-start-button" type="button" aria-describedby={needsPlayers ? 'season-start-help' : undefined} onClick={() => seasonStartMutation.mutate()} disabled={seasonStartMutation.isPending || season.assigned_count < 2}>{seasonStartMutation.isPending ? 'Starting season...' : 'Start season'}</button>
                ) : null}
                <Link className="button-link secondary-button" to="/divisions">View public league</Link>
                <Link className="button-link secondary-button" to="/admin/player-identities">Player identity review</Link>
              </div>
            </>
          ) : <p>Sign in to manage league fixtures, tournament draws, and incoming results.</p>}
        </div>
        {isAuthenticated && season ? (
          <div className="hero-side">
            <div className="metric-card">
              <span className="section-eyebrow">Assigned players</span>
              <strong>{season.assigned_count}</strong>
            </div>
            <div className="metric-card">
              <span className="section-eyebrow">{season.status === 'started' ? 'Fixtures remaining' : 'Divisions'}</span>
              <strong>{season.status === 'started' ? season.remaining_fixtures : season.division_count}</strong>
            </div>
          </div>
        ) : null}
      </section>

      {!isAuthenticated ? (
        <section className="admin-grid admin-sign-in">
          <article className="admin-card">
            <h2>Login</h2>
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

        </section>
      ) : (
        <>
          {seasonStartMutation.error ? <StateNotice tone="error" message={readError(seasonStartMutation.error)} compact /> : null}
          {seasonQuery.data?.status === 'started' && !seasonQuery.data?.admin_locked ? <StateNotice message="Registration is locked. League settings and Slack channels stay editable until the first week is released." compact /> : null}
          {seasonQuery.data?.status === 'started' && seasonQuery.data?.admin_locked ? <StateNotice message="The first week is live. Central season setup is now locked. Division scoring and Autodarts mappings remain editable." compact /> : null}

          <section className="admin-grid admin-settings-grid">
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
                  <button className="save-config-button" type="submit" disabled={updateSeasonMutation.isPending || updateConfigMutation.isPending}>{updateSeasonMutation.isPending || updateConfigMutation.isPending ? 'Saving...' : 'Save config'}</button>
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
                    <div className="toolbar-actions division-actions">
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

            <article className="admin-card admin-roster-card">
              {season?.admin_locked ? (
                <details className="locked-roster" key={season.id}>
                  <summary>
                    <h2>Registered players</h2>
                    <span className="fixture-meta">{playersQuery.data?.length ?? 0} players &middot; Roster locked</span>
                    <span className="week-switcher-icon" aria-hidden="true">&#9662;</span>
                  </summary>
                  <div>{roster}</div>
                </details>
              ) : (
                <>
                  <h2>Registered players</h2>
                  <p>Assign each player to one division or leave them waitlisted and excluded from the season schedule.</p>
                  {roster}
                </>
              )}
              {deletePlayerMutation.error ? <StateNotice tone="error" message={readError(deletePlayerMutation.error)} compact /> : null}
              {assignPlayerMutation.error ? <StateNotice tone="error" message={readError(assignPlayerMutation.error)} compact /> : null}
            </article>
          </section>
          {seasonQuery.data ? <SeasonLifecycle key={seasonQuery.data.id} season={seasonQuery.data} /> : null}
          {seasonQuery.data && !seasonQuery.isError && <AutodartsMappings key={`mappings-${seasonQuery.data.id}`} season={seasonQuery.data} players={playersQuery.data ?? []} />}
        </>
      )}
    </div>
  )
}
