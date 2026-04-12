import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import type { AdminFixture } from '../../lib/api'
import { formatAverage } from '../../lib/api'

export function AdminFixtureCard({ fixture, onSave, onUndo, isSaving, isUndoing, isLocked = false, isPastWeek = false }: {
  fixture: AdminFixture
  onSave: (payload: { fixtureId: number; playerOneLegs: number; playerTwoLegs: number; playerOneAverage?: number; playerTwoAverage?: number }) => Promise<unknown>
  onUndo: (fixtureId: number) => Promise<unknown>
  isSaving: boolean
  isUndoing: boolean
  isLocked?: boolean
  isPastWeek?: boolean
}) {
  const [playerOneLegs, setPlayerOneLegs] = useState(String(fixture.result?.player_one_legs ?? ''))
  const [playerTwoLegs, setPlayerTwoLegs] = useState(String(fixture.result?.player_two_legs ?? ''))
  const [playerOneAverage, setPlayerOneAverage] = useState(formatAverage(fixture.result?.player_one_average))
  const [playerTwoAverage, setPlayerTwoAverage] = useState(formatAverage(fixture.result?.player_two_average))
  const [statusMessage, setStatusMessage] = useState('')

  useEffect(() => {
    setPlayerOneLegs(String(fixture.result?.player_one_legs ?? ''))
    setPlayerTwoLegs(String(fixture.result?.player_two_legs ?? ''))
    setPlayerOneAverage(formatAverage(fixture.result?.player_one_average))
    setPlayerTwoAverage(formatAverage(fixture.result?.player_two_average))
  }, [fixture.id, fixture.result])

  const playerOneLegsValue = Number(playerOneLegs)
  const playerTwoLegsValue = Number(playerTwoLegs)
  const legsToWin = fixture.legs_to_win

  const parseLabel = (label: string) => {
    const match = label.match(/^(.+?)\s*\((.+)\)$/)  
    return match ? { nickname: match[1].trim(), realName: match[2].trim() } : { nickname: label, realName: null }
  }
  const playerOne = parseLabel(fixture.player_one)
  const playerTwo = parseLabel(fixture.player_two)
  const isValidScoreline = Number.isInteger(playerOneLegsValue) && Number.isInteger(playerTwoLegsValue) && (
    (playerOneLegsValue === legsToWin && playerTwoLegsValue >= 0 && playerTwoLegsValue < legsToWin) ||
    (playerTwoLegsValue === legsToWin && playerOneLegsValue >= 0 && playerOneLegsValue < legsToWin)
  )
  const scorelineHint = `Valid scores: ${legsToWin}-0 to ${legsToWin}-${Math.max(0, legsToWin - 1)} (or reversed).`

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setStatusMessage('')
    if (!isValidScoreline) {
      setStatusMessage(`Enter a valid first-to-${legsToWin} score. ${scorelineHint}`)
      return
    }
    await onSave({
      fixtureId: fixture.id,
      playerOneLegs: playerOneLegsValue,
      playerTwoLegs: playerTwoLegsValue,
      playerOneAverage: playerOneAverage === '' ? undefined : Number(playerOneAverage),
      playerTwoAverage: playerTwoAverage === '' ? undefined : Number(playerTwoAverage),
    })
    setStatusMessage('Score saved.')
  }

  const handleUndo = async () => {
    setStatusMessage('')
    await onUndo(fixture.id)
    setStatusMessage('Recorded result removed.')
  }

  const isOverdue = isPastWeek && !fixture.result

  return (
    <article className={`admin-fixture-card${fixture.result ? ' recorded' : ''}${isLocked ? ' locked-week' : ''}${isOverdue ? ' overdue' : ''}`}>
      <form className="score-form" onSubmit={handleSubmit}>
        <div className="score-table">
          <div className="score-col score-col-labels">
            <div className="score-col-status">
              {fixture.result ? <span className="week-progress-badge week-progress-badge--done">✓</span> : isOverdue ? <span className="week-progress-badge week-progress-badge--overdue">✗</span> : null}
            </div>
            <span className="score-row-label">Legs</span>
            <span className="score-row-label">Avg.</span>
          </div>
          <div className="score-col">
            <span className="score-player-name">
              {playerOne.nickname}
              {playerOne.realName ? <span className="score-player-realname">{playerOne.realName}</span> : null}
            </span>
            <input id={`p1-${fixture.id}`} type="number" min={0} max={legsToWin} step={1} value={playerOneLegs} onChange={(event) => setPlayerOneLegs(event.target.value)} inputMode="numeric" aria-label={`${fixture.player_one} legs`} />
            <input id={`a1-${fixture.id}`} value={playerOneAverage} onChange={(event) => setPlayerOneAverage(event.target.value)} inputMode="decimal" aria-label={`${fixture.player_one} average`} />
          </div>
          <div className="score-col">
            <span className="score-player-name">
              {playerTwo.nickname}
              {playerTwo.realName ? <span className="score-player-realname">{playerTwo.realName}</span> : null}
            </span>
            <input id={`p2-${fixture.id}`} type="number" min={0} max={legsToWin} step={1} value={playerTwoLegs} onChange={(event) => setPlayerTwoLegs(event.target.value)} inputMode="numeric" aria-label={`${fixture.player_two} legs`} />
            <input id={`a2-${fixture.id}`} value={playerTwoAverage} onChange={(event) => setPlayerTwoAverage(event.target.value)} inputMode="decimal" aria-label={`${fixture.player_two} average`} />
          </div>
        </div>
        <div className="score-actions">
          <button type="submit" disabled={isSaving || !isValidScoreline}>{isSaving ? 'Saving...' : 'Save score'}</button>
          {fixture.result ? <button className="secondary-button" type="button" onClick={handleUndo} disabled={isUndoing}>{isUndoing ? 'Undoing...' : 'Undo result'}</button> : null}
        </div>
      </form>
      <div className="score-feedback">
        {!isValidScoreline ? <p className="fixture-meta">{scorelineHint}</p> : statusMessage ? <p className="fixture-meta">{statusMessage}</p> : null}
      </div>
    </article>
  )
}
