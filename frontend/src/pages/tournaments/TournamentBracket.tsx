import type { BracketMatch, BracketSlot } from '../../lib/tournamentsApi'
import './tournamentBracket.css'

function slotLabel(slot: BracketSlot) {
  if (slot.bye) return 'Bye'
  if (slot.player) return slot.player
  if (slot.winner_of) return `Winner of match ${slot.winner_of}`
  return undefined
}

function roundLabel(matchCount: number) {
  if (matchCount === 1) return 'Final'
  if (matchCount === 2) return 'Semi-finals'
  if (matchCount === 4) return 'Quarter-finals'
  return `Round of ${matchCount * 2}`
}

export function TournamentBracket({ rounds }: { rounds: BracketMatch[][] }) {
  const final = rounds.at(-1)?.[0]
  const champion = final?.winner_id ? final.slots.find(slot => slot.entry_id === final.winner_id)?.player : undefined
  return (
    <section className="content-panel tournament-bracket-panel" aria-labelledby="knockout-bracket">
      <h2 id="knockout-bracket">Knockout bracket</h2>
      {champion && <p><strong>Winner:</strong> {champion}</p>}
      {rounds.length > 1 ? <p className="tournament-bracket-scroll-note">Scroll across to see later rounds.</p> : null}
      <div className="tournament-bracket" role="region" aria-label="Knockout rounds" tabIndex={0}>
        {rounds.map((round) => (
          <section className="tournament-bracket-round" key={round[0].number}>
            <h3>{roundLabel(round.length)}</h3>
            <ol>
              {round.map((match) => (
                <li key={match.number}>
                  <span className="tournament-bracket-match-label">{match.slots.some((slot) => slot.bye) ? 'Automatic advance' : `Match ${match.number}`}</span>
                  {match.slots.map((slot, index) => (
                    <div className={slot.player ? 'tournament-bracket-player' : 'fixture-meta'} key={index}><span>{slotLabel(slot)}</span>{match.legs && <b>{match.legs[index]}</b>}</div>
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
