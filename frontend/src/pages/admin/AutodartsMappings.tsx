import { useRef, useState } from 'react'
import { useAutodartsMappings, useSaveAutodartsMapping } from '../../lib/api'
import type { Player, SeasonSummary } from '../../lib/api'
import { readError } from '../../lib/utils'
import { StateNotice } from '../../components/StateNotice'

export function AutodartsMappings({ season, players }: { readonly season: SeasonSummary; readonly players: readonly Player[] }) {
  const query = useAutodartsMappings(season.id)
  const mutation = useSaveAutodartsMapping()
  const [playerId, setPlayerId] = useState('')
  const [username, setUsername] = useState('')
  const [message, setMessage] = useState('')
  const input = useRef<HTMLInputElement>(null)
  const current = query.data?.season_id === season.id
  const busy = mutation.isPending
  const blocked = busy || !current || query.isFetching || query.isError
  const trimmed = username.trim()
  const valid = trimmed.length > 0 && [...trimmed].length <= 80 && ![...trimmed].some(char => char.charCodeAt(0) < 32 || (char.charCodeAt(0) >= 127 && char.charCodeAt(0) <= 159))
  const save = (id: number, name: string | null) => {
    if (blocked) return
    setMessage('')
    mutation.mutate({ playerId: id, seasonId: season.id, username: name }, { onSuccess: () => {
      setMessage(name === null ? 'Mapping removed. Pending results still require review.' : 'Mapping saved. Pending results still require review.')
      if (name !== null || String(id) === playerId) { setPlayerId(''); setUsername('') }
    } })
  }
  return <section className="admin-card" style={{ marginTop: '1.5rem' }} aria-labelledby="autodarts-mappings-title">
    <span className="eyebrow">Source identities</span><h2 id="autodarts-mappings-title">Autodarts mappings</h2>
    <p>Current season: {season.name}. Automatic approval: {query.data ? query.data.auto_submit_enabled ? 'enabled' : 'disabled' : 'loading'}.</p>
    <p>Enter a unique Autodarts username before the first import. Use exact, case-sensitive spelling; surrounding spaces are trimmed. Mapping edits never publish pending results.</p>
    {query.isPending && <StateNotice message="Loading Autodarts mappings..." compact />}
    {query.error && <StateNotice tone="error" message={readError(query.error)} compact />}
    {query.data && !current && <p role="alert">The current season changed. Reload the admin page before saving.</p>}
    {query.isError && <button type="button" className="secondary-button" onClick={() => void query.refetch()}>Retry mappings</button>}
    <form className="admin-login" style={{ alignItems: 'start' }} onSubmit={event => { event.preventDefault(); if (valid && players.some(player => player.id === Number(playerId))) save(Number(playerId), trimmed) }}>
      <div className="field"><label htmlFor="mapping-player">League player</label>
        <select id="mapping-player" value={playerId} disabled={blocked} required onChange={event => {
          setPlayerId(event.target.value); setUsername(query.data?.mappings.find(item => item.player_id === Number(event.target.value))?.username ?? '')
        }}><option value="">Select league player...</option>{players.map(player => <option key={player.id} value={player.id}>{player.admin_label}</option>)}</select>
      </div>
      <div className="field"><label htmlFor="mapping-username">Autodarts username</label>
        <input ref={input} id="mapping-username" value={username} disabled={blocked} required autoComplete="off" aria-describedby="mapping-help" onChange={event => setUsername(event.target.value)} />
        <small id="mapping-help">1-80 characters, no control characters. For example: gsharpe.</small>
      </div>
      <button type="submit" disabled={blocked || !playerId || !valid}>{busy ? 'Saving...' : 'Save mapping'}</button>
    </form>
    {mutation.error && <p role="alert" className="form-error">{readError(mutation.error)}</p>}
    {message && <p role="status">{message}</p>}
    {current && query.data?.mappings.length === 0 && <p>No usernames mapped yet.</p>}
    {current && query.data?.mappings.map(mapping => <div className="admin-login" key={mapping.player_id}>
      <p style={{ overflowWrap: 'anywhere' }}><strong>{players.find(player => player.id === mapping.player_id)?.admin_label ?? `Player ${mapping.player_id}`}</strong>: <span>{mapping.username}</span></p>
      <div className="toolbar-actions">
        <button type="button" className="secondary-button" disabled={blocked} onClick={() => { setPlayerId(String(mapping.player_id)); setUsername(mapping.username); input.current?.focus() }}>Edit</button>
        <button type="button" className="secondary-button" disabled={blocked} onClick={() => save(mapping.player_id, null)}>Remove</button>
      </div>
    </div>)}
  </section>
}
