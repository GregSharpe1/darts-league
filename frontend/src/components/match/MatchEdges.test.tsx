import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import reference from '../../../../docs/autodarts-fixtures/randomized.json'
import { syntheticMatch } from '../../../tests/match/fixture'
import { MatchAnalysis } from './MatchAnalysis'
import { Heatmap } from './Heatmap'
import type { MatchData, MatchThrow } from './types'

afterEach(cleanup)

const dart: MatchThrow = { number: 1, segment: { bed: 'double', number: 8 }, entry_type: 'unknown', position: null }

describe('recorded data boundaries', () => {
  it.each([0, 1, 2])('renders arbitrary 3-%i partial matches without nine-darter assumptions', loserLegs => {
    const match: MatchData = { ...syntheticMatch,
      players: [{ ...syntheticMatch.players[0], stats: null }, { ...syntheticMatch.players[1], legs_won: loserLegs, stats: null }],
      detail: { coverage: 'partial', legs: Array.from({ length: 3 + loserLegs }, (_, index) => ({
        number: index + 1, completed: true, winner_id: index < loserLegs * 2 && index % 2 === 1 ? 'demo-player-2' : 'demo-player-1',
        visits: [{ number: 17, player_id: index < loserLegs * 2 && index % 2 === 1 ? 'demo-player-2' : 'demo-player-1',
          start_remaining: 16, end_remaining: 0, bust: false, throws: [dart] }],
      })) } }
    render(<MatchAnalysis status="ready" match={match} />)
    expect(screen.getAllByRole('option')).toHaveLength(4 + loserLegs)
    expect(screen.getByRole('list', { name: 'Segment totals' })).toHaveTextContent('D8: 3')
    expect(screen.getAllByText(/- Checkout/)).toHaveLength(3)
    expect(screen.getByText(/Partial detail: these counts/)).toBeVisible()
    expect(screen.queryByText('167.0')).not.toBeInTheDocument()
  })
  it('preserves all exact reference coordinates after player and leg filtering', () => {
    const { container } = render(<MatchAnalysis status="ready" match={syntheticMatch} />)
    for (const player of reference.players) {
      fireEvent.click(screen.getByRole('button', { name: player.name }))
      for (const leg of [1, 2, 3]) {
        fireEvent.change(screen.getByLabelText('Leg'), { target: { value: String(leg) } })
        const expected = reference.darts.filter(d => d.playerId === player.id && d.leg === leg)
        const dots = container.querySelectorAll('[data-dart]')
        expect(dots).toHaveLength(expected.length)
        dots.forEach((dot, index) => {
          expect(dot).toHaveAttribute('cx', String(250 + expected[index].x * 191))
          expect(dot).toHaveAttribute('cy', String(250 - expected[index].y * 191))
        })
      }
    }
  })
  it('plots only supported recorded positions in mixed data', () => {
    const position = { x: 0, y: 0, units: 'board-radius', origin: 'bull', axis_orientation: 'x-right-y-up', provenance: 'manual' } as const
    const { container } = render(<Heatmap synthetic={false} label="Mixed" throws={[
      dart, { ...dart, position }, { ...dart, position: { ...position, units: 'pixels' } },
    ]} />)
    expect(container.querySelectorAll('[data-dart]')).toHaveLength(1)
    expect(container.querySelector('[data-dart]')).toHaveAttribute('cx', '250')
    expect(screen.getByText(/2 \/ 3 recorded positions; 1 board-plottable/)).toBeVisible()
    expect(screen.getByText(/1 throws have no position/)).toBeVisible()
  })
  it.each([{ throws: [] }, { throws: [dart] }])('does not draw dots or density legend for absent coordinates (%j)', ({ throws }) => {
    const { container } = render(<Heatmap synthetic label="Missing" throws={throws} />)
    expect(container.querySelector('svg')).toBeNull()
    expect(screen.getByText('Coordinates unavailable')).toBeVisible()
  })
  it('scores a bust as zero while retaining dart segments and unfinished leg', () => {
    const match: MatchData = { ...syntheticMatch, detail: { coverage: 'partial', legs: [{
      number: 4, completed: false, winner_id: null,
      visits: [{ number: 9, player_id: 'demo-player-1', start_remaining: 10, end_remaining: 10, bust: true, throws: [dart] }],
    }] } }
    render(<MatchAnalysis status="ready" match={match} />)
    expect(screen.getByRole('option', { name: 'Leg 4 (unfinished)' })).toBeVisible()
    expect(screen.getByText('0 (bust)')).toBeVisible()
    expect(screen.queryByText(/- Checkout/)).not.toBeInTheDocument()
    expect(screen.getByRole('list', { name: 'Segment totals' })).toHaveTextContent('D8: 1')
  })
})
