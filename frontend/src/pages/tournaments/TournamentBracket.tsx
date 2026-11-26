import type { BracketMatch, BracketSlot } from '../../lib/tournamentsApi'
import './tournamentBracket.css'

function slotLabel(slot: BracketSlot) {
  if (slot.bye) return 'Bye'
  if (slot.winner_of) return `Winner of match ${slot.winner_of}`
  return slot.player
}

function roundLabel(matchCount: number) {
  if (matchCount === 1) return 'Final'
  if (matchCount === 2) return 'Semi-finals'
  if (matchCount === 4) return 'Quarter-finals'
  return `Round of ${matchCount * 2}`
}

export function TournamentBracket({ rounds }: { rounds: BracketMatch[][] }) {
  return (
    <section className="content-panel" aria-labelledby="knockout-bracket">
      <h2 id="knockout-bracket">Knockout bracket</h2>
      <div className="tournament-bracket" role="region" aria-label="Knockout rounds" tabIndex={0}>
        {rounds.map((round) => (
          <section className="tournament-bracket-round" key={round[0].number}>
            <h3>{roundLabel(round.length)}</h3>
            <ol>
              {round.map((match) => (
                <li key={match.number}>
                  <span className="fixture-meta">{match.slots.some((slot) => slot.bye) ? 'Walkover' : `Match ${match.number}`}</span>
                  {match.slots.map((slot, index) => (
                    <div className={slot.player ? 'tournament-bracket-player' : 'fixture-meta'} key={index}>{slotLabel(slot)}</div>
                  ))}
                </li>
              ))}
            </ol>
          </section>
        ))}
      </div>
    </section>
  )
}
