import type { Player } from '../../lib/api'
import { formatWhen } from '../../lib/api'

export function PlayerRoster({ players, registrationOpen, onDelete, isDeleting }: {
  players: Player[]
  registrationOpen: boolean
  onDelete: (playerId: number) => Promise<unknown>
  isDeleting: boolean
}) {
  return (
    <ul className="check-list">
      {players.map((player) => (
        <li key={player.id}>
          <div>
            <strong>{player.admin_label}</strong>
            <div className="fixture-meta">Registered {player.registered_at ? formatWhen(player.registered_at) : 'recently'}</div>
          </div>
          {registrationOpen
            ? <button className="ghost-button" type="button" onClick={() => onDelete(player.id)} disabled={isDeleting}>{isDeleting ? 'Deleting...' : 'Delete'}</button>
            : <span className="fixture-meta">Roster locked</span>
          }
        </li>
      ))}
    </ul>
  )
}
