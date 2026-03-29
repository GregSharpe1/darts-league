import { useMemo } from 'react'
import type { AuditEntry } from '../../lib/api'
import { formatWhen, formatAverage } from '../../lib/api'

export function AuditEntryCard({ entry }: { entry: AuditEntry }) {
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
