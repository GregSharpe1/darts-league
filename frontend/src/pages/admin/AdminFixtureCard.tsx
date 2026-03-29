import { useState } from 'react'
import type { FormEvent } from 'react'
import type { AdminFixture } from '../../lib/api'
import { formatAverage } from '../../lib/api'

export function AdminFixtureCard({ fixture, onSave, onUndo, isSaving, isUndoing, isLocked = false }: {
  fixture: AdminFixture
  onSave: (payload: { fixtureId: number; playerOneLegs: number; playerTwoLegs: number; playerOneAverage?: number; playerTwoAverage?: number }) => Promise<unknown>
  onUndo: (fixtureId: number) => Promise<unknown>
  isSaving: boolean
  isUndoing: boolean
  isLocked?: boolean
}) {
  const [playerOneLegs, setPlayerOneLegs] = useState(String(fixture.result?.player_one_legs ?? ''))
  const [playerTwoLegs, setPlayerTwoLegs] = useState(String(fixture.result?.player_two_legs ?? ''))
  const [playerOneAverage, setPlayerOneAverage] = useState(formatAverage(fixture.result?.player_one_average))
  const [playerTwoAverage, setPlayerTwoAverage] = useState(formatAverage(fixture.result?.player_two_average))
  const [statusMessage, setStatusMessage] = useState('')

  const playerOneLegsValue = Number(playerOneLegs)
  const playerTwoLegsValue = Number(playerTwoLegs)
  const legsToWin = fixture.legs_to_win
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

  return (
    <article className={`admin-fixture-card${fixture.result ? ' recorded' : ''}${isLocked ? ' locked-week' : ''}`}>
      <form className="score-form" onSubmit={handleSubmit}>
        <div className="score-columns">
          <div className="score-player-col">
            <span className="score-player-name">{fixture.player_one}</span>
            <div className="score-field">
              <label htmlFor={`p1-${fixture.id}`}>Legs ({fixture.player_one})</label>
              <input id={`p1-${fixture.id}`} type="number" min={0} max={legsToWin} step={1} value={playerOneLegs} onChange={(event) => setPlayerOneLegs(event.target.value)} inputMode="numeric" />
            </div>
            <div className="score-field">
              <label htmlFor={`a1-${fixture.id}`}>Average ({fixture.player_one})</label>
              <input id={`a1-${fixture.id}`} value={playerOneAverage} onChange={(event) => setPlayerOneAverage(event.target.value)} inputMode="decimal" />
            </div>
          </div>
          <div className="score-player-col">
            <span className="score-player-name">{fixture.player_two}</span>
            <div className="score-field">
              <label htmlFor={`p2-${fixture.id}`}>Legs ({fixture.player_two})</label>
              <input id={`p2-${fixture.id}`} type="number" min={0} max={legsToWin} step={1} value={playerTwoLegs} onChange={(event) => setPlayerTwoLegs(event.target.value)} inputMode="numeric" />
            </div>
            <div className="score-field">
              <label htmlFor={`a2-${fixture.id}`}>Average ({fixture.player_two})</label>
              <input id={`a2-${fixture.id}`} value={playerTwoAverage} onChange={(event) => setPlayerTwoAverage(event.target.value)} inputMode="decimal" aria-label={`${fixture.player_two} average`} />
            </div>
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
