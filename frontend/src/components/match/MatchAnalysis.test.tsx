import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { syntheticMatch, withoutGeometry } from '../../../tests/match/fixture'
import { MatchAnalysis } from './MatchAnalysis'
import type { MatchData } from './types'

afterEach(cleanup)

describe('match analysis', () => {
  it('uses exact source positions and segment counts when reference data is supplied', () => {
    const { container } = render(<MatchAnalysis status="ready" match={syntheticMatch} />)
    const dots = container.querySelectorAll('[data-dart]')
    expect(dots).toHaveLength(27)
    expect(dots[1]).toHaveAttribute('cx', '249.74626310007167')
    expect(dots[1]).toHaveAttribute('cy', '133.75238690385638')
    expect(screen.getByRole('list', { name: 'Segment totals' })).toHaveTextContent('T20: 21')
    expect(screen.getByRole('list', { name: 'Segment totals' })).toHaveTextContent('T19: 3')
    expect(screen.getByRole('list', { name: 'Segment totals' })).toHaveTextContent('D12: 3')
  })
  it('joins by ID when source order is reversed and filters each leg', () => {
    const match: MatchData = { ...syntheticMatch,
      players: [syntheticMatch.players[1], syntheticMatch.players[0]] }
    const { container } = render(<MatchAnalysis status="ready" match={match} />)
    fireEvent.click(screen.getByRole('button', { name: 'Casey Vale' }))
    for (const [leg, count] of [['1', 6], ['2', 9], ['3', 6]] as const) {
      fireEvent.change(screen.getByLabelText('Leg'), { target: { value: leg } })
      expect(container.querySelectorAll('[data-dart]')).toHaveLength(count)
      expect(screen.getByRole('list', { name: 'Segment totals' })).toHaveTextContent(`Miss: ${count}`)
    }
  })
  it('preserves counts without plotting when geometry is unknown', () => {
    const { container } = render(<MatchAnalysis status="ready" match={withoutGeometry(syntheticMatch)} />)
    expect(container.querySelectorAll('[data-dart], [data-heat]')).toHaveLength(0)
    expect(screen.getByText('Coordinates unavailable')).toBeVisible()
    expect(screen.queryByText('Overlapping recorded positions')).not.toBeInTheDocument()
    expect(screen.getByRole('list', { name: 'Segment totals' })).toHaveTextContent('T20: 21')
  })
  it('shows summary-only unknowns without inventing zeros', () => {
    const match: MatchData = { ...syntheticMatch, detail: null,
      players: [{ ...syntheticMatch.players[0], label: '<img onerror=alert(1)>' + 'Long'.repeat(15), stats: null }, syntheticMatch.players[1]] }
    const { container } = render(<MatchAnalysis status="ready" match={match} />)
    expect(screen.getByText('Throw detail unavailable')).toBeVisible()
    expect(container.querySelector('img')).toBeNull()
    expect(screen.getAllByText('Unknown').length).toBeGreaterThan(0)
  })
  it('shows no percentage when checkout attempts are zero', () => {
    render(<MatchAnalysis status="ready" match={syntheticMatch} />)
    const summary = screen.getByRole('region', { name: 'Match summary' })
    expect(within(summary).getByText('0 / 0 (not applicable)')).toBeVisible()
  })
  it('reports loading and retryable error without fetching', () => {
    const retry = vi.fn()
    const { rerender } = render(<MatchAnalysis status="loading" />)
    expect(screen.getByRole('status')).toHaveTextContent('Loading match analysis')
    rerender(<MatchAnalysis status="error" onRetry={retry} />)
    expect(screen.getByRole('alert')).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }))
    expect(retry).toHaveBeenCalledOnce()
  })
})
