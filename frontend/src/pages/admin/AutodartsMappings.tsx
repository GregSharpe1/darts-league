import { useEffect, useRef, useState } from 'react'
import { useAutodartsMappings, useSaveAutodartsMapping } from '../../lib/api'
import type { Player, SeasonSummary } from '../../lib/api'
import { readError } from '../../lib/utils'
import { StateNotice } from '../../components/StateNotice'

export function AutodartsMappings({ season, players }: { readonly season: SeasonSummary; readonly players: readonly Player[] }) {
  const query = useAutodartsMappings(season.id)
  const mutation = useSaveAutodartsMapping()
  const [playerId, setPlayerId] = useState('')
  const [editingId, setEditingId] = useState<number>()
  const [username, setUsername] = useState('')
  const [message, setMessage] = useState('')
  const input = useRef<HTMLInputElement>(null)
  const data = query.data
  const current = data !== undefined && data.season_id === season.id
  const mappings = current ? data.mappings : []
  const mappedIds = new Set(mappings.map(mapping => mapping.player_id))
  const availablePlayers = players.filter(player => !mappedIds.has(player.id))
  const busy = mutation.isPending
  const blocked = busy || !current || query.isFetching || query.isError
  const trimmed = username.trim()
  const valid = trimmed.length > 0 && [...trimmed].length <= 80 && ![...trimmed].some(char => char.charCodeAt(0) < 32 || (char.charCodeAt(0) >= 127 && char.charCodeAt(0) <= 159))
  const selectedId = editingId ?? availablePlayers.find(player => String(player.id) === playerId)?.id
  const playerLabel = (id: number) => players.find(player => player.id === id)?.admin_label ?? `Player ${id}`
  useEffect(() => {
    if (editingId !== undefined) input.current?.focus()
  }, [editingId])

  const resetForm = () => {
    setPlayerId('')
    setEditingId(undefined)
    setUsername('')
  }
  const save = (id: number, name: string | null) => {
    if (blocked) return
    setMessage('')
    mutation.mutate({ playerId: id, seasonId: season.id, username: name }, { onSuccess: () => {
      setMessage(name === null ? 'Mapping removed. Pending results still require review.' : 'Mapping saved. Pending results still require review.')
      if (name !== null || editingId === id) resetForm()
    } })
  }
  return <section className="admin-card autodarts-mappings" aria-labelledby="autodarts-mappings-title">
    <span className="eyebrow">Source identities</span><h2 id="autodarts-mappings-title">Autodarts mappings</h2>
    <p>Current season: {season.name}. Automatic approval: {query.data ? query.data.auto_submit_enabled ? 'enabled' : 'disabled' : 'loading'}.</p>
    <p>Enter a unique Autodarts username before the first import. Use exact, case-sensitive spelling; surrounding spaces are trimmed. Mapping edits never publish pending results.</p>
    {query.isPending && <StateNotice message="Loading Autodarts mappings..." compact />}
    {query.error && <StateNotice tone="error" message={readError(query.error)} compact />}
    {query.data && !current && <p role="alert">The current season changed. Reload the admin page before saving.</p>}
    {query.isError && <button type="button" className="secondary-button" onClick={() => void query.refetch()}>Retry mappings</button>}
    <form className="admin-login" onSubmit={event => { event.preventDefault(); if (valid && selectedId !== undefined) save(selectedId, trimmed) }}>
      <div className="field">
        <label htmlFor="mapping-player">League player</label>
        {editingId !== undefined ? (
          <input id="mapping-player" value={playerLabel(editingId)} readOnly />
        ) : (
          <select id="mapping-player" value={selectedId === undefined ? '' : playerId} disabled={blocked || availablePlayers.length === 0} required onChange={event => setPlayerId(event.target.value)}>
            <option value="">Select league player...</option>
            {availablePlayers.map(player => <option key={player.id} value={player.id}>{player.admin_label}</option>)}
          </select>
        )}
      </div>
      <div className="field">
        <label htmlFor="mapping-username">Autodarts username</label>
        <input ref={input} id="mapping-username" value={username} disabled={blocked || (editingId === undefined && availablePlayers.length === 0)} required autoComplete="off" aria-describedby="mapping-help" onChange={event => setUsername(event.target.value)} />
        <small id="mapping-help">1-80 characters, no control characters.</small>
      </div>
      <button type="submit" disabled={blocked || selectedId === undefined || !valid}>{busy ? 'Saving...' : 'Save mapping'}</button>
      {editingId !== undefined ? <div className="toolbar-actions"><button type="button" className="secondary-button" disabled={busy} onClick={resetForm}>Cancel edit</button></div> : null}
    </form>
    {mutation.error && <p role="alert" className="form-error">{readError(mutation.error)}</p>}
    {message && <p role="status">{message}</p>}
    {current && mappings.length === 0 && <p>No usernames mapped yet.</p>}
    {current && players.length > 0 && availablePlayers.length === 0 && editingId === undefined && <p>All league players have an Autodarts mapping.</p>}
    {mappings.length > 0 && <ul className="check-list autodarts-mapping-list" aria-label="Mapped players">
      {mappings.map(mapping => {
        const label = playerLabel(mapping.player_id)
        return <li key={mapping.player_id}>
          <div className="tournament-list-label"><strong>{label}</strong><div className="fixture-meta">{mapping.username}</div></div>
          <div className="toolbar-actions">
            <button type="button" className="secondary-button" aria-label={`Edit mapping for ${label}`} disabled={blocked} onClick={() => {
              setMessage('')
              setEditingId(mapping.player_id)
              setUsername(mapping.username)
            }}>Edit</button>
            <button type="button" className="secondary-button" aria-label={`Remove mapping for ${label}`} disabled={blocked} onClick={() => save(mapping.player_id, null)}>Remove</button>
          </div>
        </li>
      })}
    </ul>}
  </section>
}
