import { useState } from 'react'
import type { FormEvent } from 'react'
import { StateNotice } from '../../components/StateNotice'
import { useCorrectTournamentScore, useTournamentEntries } from '../../lib/tournamentsApi'
import type { BracketMatch, BracketSlot, Tournament, TournamentScoreCorrection } from '../../lib/tournamentsApi'
import { readError } from '../../lib/utils'

function scoredLaterMatches(matches: BracketMatch[], number: number) {
  const following: number[] = []
  let next = matches.find(match => match.slots.some(slot => slot.winner_of === number))
  while (next?.legs) {
    following.push(next.number)
    const previous = next.number
    next = matches.find(match => match.slots.some(slot => slot.winner_of === previous))
  }
  return following
}

export function TournamentScoreEditor({ event }: { event: Tournament }) {
  const [matchNumber, setMatchNumber] = useState('')
  const correction = useCorrectTournamentScore(event.slug)
  const entries = useTournamentEntries(event.slug)
  const entrantName = (slot: BracketSlot) => {
    const entrant = entries.data?.find(entry => entry.id === slot.entry_id)
    if (!entrant) return slot.player ?? ''
    return entrant.nickname && entrant.nickname !== entrant.display_name ?
      `${entrant.nickname} (${entrant.display_name})` : entrant.display_name
  }
  const matches = event.bracket?.flat() ?? []
  const scored = matches.filter(match => match.legs !== undefined)
  const match = scored.find(item => String(item.number) === matchNumber)

  return (
    <section className="admin-card content-panel" aria-labelledby="tournament-scores-heading">
      <h2 id="tournament-scores-heading">Correct tournament scores</h2>
      {scored.length === 0 ? <p>Score corrections are available after a match result is recorded.</p> : (
        <>
          <p>Correct a recorded result using the tournament's match format.</p>
          <div className="form-preview">
            <div className="field">
              <label htmlFor="tournament-score-match">Recorded match</label>
              <select id="tournament-score-match" value={match?.number ?? ''} disabled={correction.isPending} onChange={event => {
                correction.reset()
                setMatchNumber(event.target.value)
              }}>
                <option value="">Select a match...</option>
                {scored.map(item => (
                  <option key={item.number} value={item.number}>
                    Match {item.number} &middot; {entrantName(item.slots[0])} vs {entrantName(item.slots[1])} ({item.legs?.join('-')})
                  </option>
                ))}
              </select>
            </div>
          </div>
          {match?.legs && (
            <ScoreCorrectionForm
              key={JSON.stringify([match.number, match.legs, match.slots.map(slot => slot.entry_id)])}
              match={match}
              entrantNames={[entrantName(match.slots[0]), entrantName(match.slots[1])]}
              previousLegs={match.legs}
              legsToWin={event.legs_to_win}
              laterResults={scoredLaterMatches(matches, match.number)}
              disabled={correction.isPending}
              onSave={body => correction.mutate({ number: match.number, ...body })}
            />
          )}
          {correction.error ? <StateNotice tone="error" message={readError(correction.error)} compact /> : null}
          {correction.isSuccess ? <p role="status">Tournament score corrected.</p> : null}
        </>
      )}
    </section>
  )
}

function ScoreCorrectionForm({ match, entrantNames, previousLegs, legsToWin, laterResults, disabled, onSave }: {
  match: BracketMatch
  entrantNames: [string, string]
  previousLegs: [number, number]
  legsToWin: number
  laterResults: number[]
  disabled: boolean
  onSave: (correction: TournamentScoreCorrection) => void
}) {
  const [first, setFirst] = useState(String(previousLegs[0]))
  const [second, setSecond] = useState(String(previousLegs[1]))
  const [reason, setReason] = useState('')
  const legs: [number, number] = [Number(first), Number(second)]
  const [left, right] = match.slots
  const entrants: [number, number] = [left.entry_id ?? 0, right.entry_id ?? 0]
  const changed = legs[0] !== previousLegs[0] || legs[1] !== previousLegs[1]
  const valid = first !== '' && second !== '' && legs.every(score => Number.isInteger(score) && score >= 0) &&
    Math.max(...legs) === legsToWin && Math.min(...legs) < legsToWin
  const clearMatches = valid && (legs[0] > legs[1]) !== (previousLegs[0] > previousLegs[1]) ? laterResults : []
  const canSave = !disabled && changed && valid && Boolean(reason.trim())

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!canSave) return
    if (clearMatches.length > 0 && !window.confirm(`This changes the winner and clears the scores for later matches ${clearMatches.join(', ')}. Save this correction?`)) return
    onSave({ legs, previous_legs: previousLegs, entrants, clear_matches: clearMatches, reason: reason.trim() })
  }

  return (
    <form className="form-preview tournament-score-form" onSubmit={submit}>
      <div className="tournament-score-fields">
        <div className="field">
          <label htmlFor="tournament-first-score">{entrantNames[0]} legs</label>
          <input id="tournament-first-score" type="number" min="0" max={legsToWin} required value={first} disabled={disabled} onChange={event => setFirst(event.target.value)} />
        </div>
        <div className="field">
          <label htmlFor="tournament-second-score">{entrantNames[1]} legs</label>
          <input id="tournament-second-score" type="number" min="0" max={legsToWin} required value={second} disabled={disabled} onChange={event => setSecond(event.target.value)} />
        </div>
      </div>
      <div className="field">
        <label htmlFor="tournament-score-reason">Reason for correction</label>
        <input id="tournament-score-reason" value={reason} maxLength={500} required disabled={disabled} onChange={event => setReason(event.target.value)} />
      </div>
      {changed && !valid ? <p className="fixture-meta">The winner must reach {legsToWin} legs and the other player must have fewer.</p> : null}
      {clearMatches.length > 0 ? <p role="status">Changing the winner will clear the scores for later matches {clearMatches.join(', ')}.</p> : null}
      <button type="submit" disabled={!canSave}>{disabled ? 'Saving...' : 'Save corrected score'}</button>
    </form>
  )
}
