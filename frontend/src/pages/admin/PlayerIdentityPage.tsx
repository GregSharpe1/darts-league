import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { StateNotice } from '../../components/StateNotice'
import { ApiError, request } from '../../lib/api'
import { readError } from '../../lib/utils'
import './PlayerIdentityPage.css'

type Entry = {
  identity_id: number
  kind: 'league' | 'tournament'
  entry_id: number
  display_name: string
  nickname: string
  competition: string
}

type Pair = { left: number; right: number }
type Suggestion = Pair & { signals: string[]; can_link: boolean }
type Review = { entries: Entry[]; suggestions: Suggestion[] }

const queryKey = ['admin-player-identities']

function entryLabel(entry: Entry) {
  return entry.nickname && entry.nickname !== entry.display_name
    ? `${entry.display_name} (${entry.nickname})`
    : entry.display_name
}

function IdentityEntries({ entries }: { entries: Entry[] }) {
  return <ul className="identity-entry-list">
    {entries.map(entry => <li key={`${entry.kind}-${entry.entry_id}`}>
      <strong>{entryLabel(entry)}</strong>
      <span>{entry.kind === 'league' ? 'League' : 'Tournament'} / {entry.competition}</span>
    </li>)}
  </ul>
}

export function PlayerIdentityPage() {
  const queryClient = useQueryClient()
  const review = useQuery({
    queryKey,
    queryFn: ({ signal }) => request<Review>('/api/admin/player-identities', { signal, cache: 'no-store' }),
    retry: false,
    gcTime: 0,
  })
  const decide = useMutation({
    mutationFn: (decision: Pair & { action: 'link' | 'separate' }) =>
      request<void>('/api/admin/player-identities/decisions', { method: 'POST', body: JSON.stringify(decision) }),
    onSuccess: async () => {
      setManualLeft('')
      setManualRight('')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey }),
        queryClient.invalidateQueries({ queryKey: ['public-player-directory'] }),
        queryClient.invalidateQueries({ queryKey: ['player-placements'] }),
      ])
    },
  })
  const [manualLeft, setManualLeft] = useState('')
  const [manualRight, setManualRight] = useState('')
  const grouped = new Map<number, Entry[]>()
  for (const entry of review.data?.entries ?? []) {
    const entries = grouped.get(entry.identity_id) ?? []
    entries.push(entry)
    grouped.set(entry.identity_id, entries)
  }
  const options = Array.from(grouped).map(([id, entries]) => ({
    id, label: `${entryLabel(entries[0])} / ${entries[0].competition}${entries.length > 1 ? ` (+${entries.length - 1} entries)` : ''}`,
  })).sort((a, b) => a.label.localeCompare(b.label))

  function link(left: number, right: number) {
    const first = grouped.get(left)?.[0]
    const second = grouped.get(right)?.[0]
    if (!first || !second) return
    if (window.confirm(`Link all entries for ${entryLabel(first)} (${first.competition}) and ${entryLabel(second)} (${second.competition})? Their placements will be calculated together.`)) {
      decide.mutate({ left, right, action: 'link' })
    }
  }

  return <div className="admin-workspace">
    <section className="page-intro">
      <h1>Player identity review</h1>
      <p>Review possible name matches before combining player stats.</p>
    </section>
    {review.isPending ? <p role="status">Loading player identities...</p> :
      review.isError ? <section className="admin-card">
        {review.error instanceof ApiError && review.error.status === 401
          ? <Link to="/admin">Sign in to review player identities</Link>
          : <><StateNotice tone="error" message={readError(review.error)} />
            <button type="button" onClick={() => { void review.refetch() }}>Try again</button></>}
      </section> : <>
        <div role="status">
          {decide.isSuccess ? <StateNotice message={decide.variables.action === 'link' ? 'Player profiles linked.' : 'Profiles marked as different people.'} /> : null}
        </div>
        {decide.isError ? <div role="alert"><StateNotice tone="error" message={readError(decide.error)} /></div> : null}
        <section className="admin-card" aria-labelledby="identity-suggestions">
          <h2 id="identity-suggestions">Possible matches</h2>
          {review.data.suggestions.length === 0 ? <p>No possible matches to review.</p> :
            <ul className="identity-suggestions">
              {review.data.suggestions.map(({ left, right, signals, can_link }) => <li key={`${left}-${right}`}>
                <p>{signals.join(' / ')}</p>
                <div className="identity-compare">
                  <IdentityEntries entries={grouped.get(left) ?? []} />
                  <IdentityEntries entries={grouped.get(right) ?? []} />
                </div>
                {!can_link ? <p>Both profiles already have an entry in the same competition.</p> : null}
                <div className="toolbar-actions">
                  <button type="button" disabled={!can_link || decide.isPending} onClick={() => link(left, right)}>Link profiles</button>
                  <button className="secondary-button" type="button" disabled={decide.isPending}
                    onClick={() => decide.mutate({ left, right, action: 'separate' })}>Keep separate</button>
                </div>
              </li>)}
            </ul>}
        </section>
        {options.length > 1 ? <section className="admin-card" aria-labelledby="identity-manual">
          <h2 id="identity-manual">Link profiles manually</h2>
          <p>Use this when a returning player registered under a different name.</p>
          <div className="identity-compare">
            <div className="field">
              <label htmlFor="identity-left">First profile</label>
              <select id="identity-left" value={manualLeft} onChange={e => setManualLeft(e.target.value)}>
                <option value="">Choose a profile</option>
                {options.map(option => <option value={option.id} key={option.id}>{option.label}</option>)}
              </select>
            </div>
            <div className="field">
              <label htmlFor="identity-right">Second profile</label>
              <select id="identity-right" value={manualRight} onChange={e => setManualRight(e.target.value)}>
                <option value="">Choose a profile</option>
                {options.map(option => <option value={option.id} key={option.id}>{option.label}</option>)}
              </select>
            </div>
          </div>
          <button type="button" disabled={!manualLeft || !manualRight || manualLeft === manualRight || decide.isPending}
            onClick={() => link(Number(manualLeft), Number(manualRight))}>Link selected profiles</button>
        </section> : null}
      </>}
  </div>
}
