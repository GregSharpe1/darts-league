import type { Division, Player } from '../../lib/api'
import { formatWhen } from '../../lib/api'

export function PlayerRoster({
  players,
  divisions,
  registrationOpen,
  onDelete,
  onAssign,
  isDeleting,
  isAssigning,
}: {
  players: Player[]
  divisions: Division[]
  registrationOpen: boolean
  onDelete: (playerId: number) => Promise<unknown>
  onAssign: (playerId: number, divisionId?: number) => Promise<unknown>
  isDeleting: boolean
  isAssigning: boolean
}) {
  const divisionNameByID = new Map(divisions.map((division) => [division.id, division.name]))

  return (
    <ul className="check-list">
      {players.map((player) => (
        <li key={player.id}>
          <div>
            <strong>{player.admin_label}</strong>
            <div className="fixture-meta">Registered {player.registered_at ? formatWhen(player.registered_at) : 'recently'}</div>
            <div className="fixture-meta">
              {player.division_id ? `Assigned to ${divisionNameByID.get(player.division_id) ?? 'division'}` : 'Waitlist / inactive'}
            </div>
          </div>
          {registrationOpen ? (
            <div className="toolbar-actions">
              <select
                aria-label={`${player.admin_label} division`}
                value={player.division_id ?? ''}
                onChange={(event) => onAssign(player.id, event.target.value ? Number(event.target.value) : undefined)}
                disabled={isAssigning}
              >
                <option value="">Waitlist / inactive</option>
                {divisions.map((division) => (
                  <option key={division.id} value={division.id}>{division.name}</option>
                ))}
              </select>
              <button className="ghost-button" type="button" onClick={() => onDelete(player.id)} disabled={isDeleting}>{isDeleting ? 'Deleting...' : 'Delete'}</button>
            </div>
          ) : (
            <span className="fixture-meta">Roster locked</span>
          )}
        </li>
      ))}
    </ul>
  )
}
