import { useState } from 'react'
import type { Division, PendingResult, Player } from '../../lib/api'
import { formatAverage, formatWhen } from '../../lib/api'

export function PendingResultCard({
  pending,
  players,
  divisions,
  onConfirm,
  onReject,
  isConfirming,
  isRejecting,
}: {
  pending: PendingResult
  players: Player[]
  divisions: Division[]
  onConfirm: (payload: { pendingId: number; playerOneId: number; playerTwoId: number }) => Promise<unknown>
  onReject: (pendingId: number) => Promise<unknown>
  isConfirming: boolean
  isRejecting: boolean
}) {
  const [divisionId, setDivisionId] = useState('')
  const [playerOneId, setPlayerOneId] = useState('')
  const [playerTwoId, setPlayerTwoId] = useState('')
  const [statusMessage, setStatusMessage] = useState('')

  const canConfirm = Boolean(playerOneId) && Boolean(playerTwoId) && playerOneId !== playerTwoId
  const filteredPlayers = players.filter((player) => String(player.division_id ?? '') === divisionId)
  const controlsDisabled = isConfirming || isRejecting

  const handleConfirm = async () => {
    setStatusMessage('')
    if (!canConfirm) {
      setStatusMessage('Select a matching player for each side before confirming.')
      return
    }
    try {
      await onConfirm({ pendingId: pending.id, playerOneId: Number(playerOneId), playerTwoId: Number(playerTwoId) })
      setStatusMessage('Result recorded.')
    } catch (error) {
      setStatusMessage(error instanceof Error ? error.message : 'Result could not be recorded.')
    }
  }

  const handleReject = async () => {
    setStatusMessage('')
    try {
      await onReject(pending.id)
    } catch (error) {
      setStatusMessage(error instanceof Error ? error.message : 'Result could not be rejected.')
    }
  }

  return (
    <article className="admin-fixture-card">
      <div className="fixture-meta">Received {formatWhen(pending.received_at)}</div>
      <div className="score-table">
        <div className="score-col score-col-labels">
          <div className="score-col-status" />
          <span className="score-row-label">Legs</span>
          <span className="score-row-label">Avg.</span>
        </div>
        <div className="score-col">
          <span className="score-player-name">
            {pending.player_one_name}
            <span className="score-player-realname">Reported by scoring app</span>
          </span>
          <input readOnly value={pending.player_one_legs} aria-label={`${pending.player_one_name} legs`} />
          <input readOnly value={formatAverage(pending.player_one_average)} aria-label={`${pending.player_one_name} average`} />
        </div>
        <div className="score-col">
          <span className="score-player-name">
            {pending.player_two_name}
            <span className="score-player-realname">Reported by scoring app</span>
          </span>
          <input readOnly value={pending.player_two_legs} aria-label={`${pending.player_two_name} legs`} />
          <input readOnly value={formatAverage(pending.player_two_average)} aria-label={`${pending.player_two_name} average`} />
        </div>
      </div>

      <div className="pending-mapping">
        <div className="field">
          <label htmlFor={`division-${pending.id}`}>League</label>
          <select
            id={`division-${pending.id}`}
            value={divisionId}
            onChange={(event) => {
              setDivisionId(event.target.value)
              setPlayerOneId('')
              setPlayerTwoId('')
            }}
            disabled={controlsDisabled}
          >
            <option value="">Select league...</option>
            {divisions.map((division) => (
              <option key={division.id} value={division.id}>{division.name}</option>
            ))}
          </select>
        </div>
        <div className="pending-player-mapping">
          <div className="field">
            <label htmlFor={`player-one-${pending.id}`}>Player one</label>
            <select id={`player-one-${pending.id}`} value={playerOneId} onChange={(event) => setPlayerOneId(event.target.value)} disabled={controlsDisabled || !divisionId}>
              <option value="">Select player...</option>
              {filteredPlayers.map((player) => <option key={player.id} value={player.id}>{player.admin_label}</option>)}
            </select>
          </div>
          <div className="field">
            <label htmlFor={`player-two-${pending.id}`}>Player two</label>
            <select id={`player-two-${pending.id}`} value={playerTwoId} onChange={(event) => setPlayerTwoId(event.target.value)} disabled={controlsDisabled || !divisionId}>
              <option value="">Select player...</option>
              {filteredPlayers.map((player) => <option key={player.id} value={player.id}>{player.admin_label}</option>)}
            </select>
          </div>
        </div>
      </div>

      <div className="score-actions">
        <button type="button" onClick={handleConfirm} disabled={controlsDisabled || !canConfirm}>{isConfirming ? 'Confirming...' : 'Confirm Score'}</button>
        <button className="secondary-button" type="button" onClick={handleReject} disabled={controlsDisabled}>{isRejecting ? 'Rejecting...' : 'Reject Result'}</button>
      </div>
      {statusMessage ? <p className="fixture-meta" role="status">{statusMessage}</p> : null}
    </article>
  )
}
