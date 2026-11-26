import type { TournamentMatchFormat } from '../../lib/tournamentsApi'

export function TournamentFormat({ format }: { format: TournamentMatchFormat }) {
  return <p>{format.game_variant}, first to {format.legs_to_win} {format.legs_to_win === 1 ? 'leg' : 'legs'}</p>
}
