import { useState } from 'react'
import { useSeasonLifecycle } from '../../lib/api'
import type { SeasonSummary } from '../../lib/api'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'

export function SeasonLifecycle({ season }: { readonly season: SeasonSummary }) {
  const close = useSeasonLifecycle('close')
  const next = useSeasonLifecycle('next')
  const [name, setName] = useState('')

  if (season.status === 'registration_open') return null

  return (
    <section className="admin-card season-lifecycle" aria-label="Season lifecycle">
      {season.status === 'started' ? (
        <div className="card-header">
          <div className="card-copy">
            <h2>Finish the league</h2>
            <p id="close-league-help">{season.remaining_fixtures} match{season.remaining_fixtures === 1 ? '' : 'es'} remaining across all divisions. {!season.can_close_season ? 'Record every match result to enable closure. ' : ''}Closing makes results read-only and keeps the division boards visible.</p>
          </div>
          <div className="toolbar-actions season-close-actions">
            <button className="season-close-button" type="button" aria-describedby="close-league-help" disabled={!season.can_close_season || close.isPending} onClick={() => {
              if (window.confirm('Close this league? All divisions and results become read-only. Results stay visible until the next season registration opens. This cannot be undone.')) close.mutate({ season_id: season.id })
            }}>{close.isPending ? 'Closing league...' : 'Close league'}</button>
          </div>
        </div>
      ) : (
        <>
          <h2>League completed</h2>
          <p>All divisions and results are read-only. Final standings and results remain visible until you open the next season registration.</p>
          {season.can_create_next_season ? (
            <form className="admin-login" onSubmit={(event) => {
              event.preventDefault()
              if (window.confirm('Open registration for a new season? Public boards will switch immediately to the new league. Old results stay stored, but will no longer appear on the boards. The new roster and divisions start empty.')) next.mutate({ season_id: season.id, name: name.trim() })
            }}>
              <div className="field">
                <label htmlFor="next-season-name">Next league name</label>
                <input id="next-season-name" value={name} onChange={(event) => setName(event.target.value)} required minLength={2} maxLength={60} disabled={next.isPending} />
              </div>
              <button type="submit" disabled={next.isPending || name.trim().length < 2}>{next.isPending ? 'Opening registration...' : 'Open next season registration'}</button>
            </form>
          ) : null}
        </>
      )}
      {close.error || next.error ? <StateNotice tone="error" message={readError(close.error ?? next.error)} compact /> : null}
    </section>
  )
}
