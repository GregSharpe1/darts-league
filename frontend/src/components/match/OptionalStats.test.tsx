import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'
import { syntheticMatch } from '../../../tests/match/fixture'
import { MatchAnalysis } from './MatchAnalysis'
import type { MatchData } from './types'

afterEach(cleanup)

it('shows source match statistics, preserves zeros and never uses them for a leg', () => {
  const match: MatchData = { ...syntheticMatch,
    detail: { coverage: 'partial', legs: syntheticMatch.detail!.legs.slice(0, 1) },
    players: [{ ...syntheticMatch.players[0], stats: {
      match_average: 77, points_scored: null, darts_thrown: null, checkout_hits: null, checkout_attempts: null,
      first_nine_average: 81.25, total_180: 0, highest_finish: 120,
    } }, syntheticMatch.players[1]],
  }
  render(<MatchAnalysis status="ready" match={match} />)
  const analysis = within(screen.getByRole('region', { name: 'Throw analysis' }))
  const value = (name: string) => analysis.getByText(name).nextElementSibling
  expect(value('First-nine average (source)')).toHaveTextContent('81.3')
  expect(value('180s (source)')).toHaveTextContent('0')
  expect(value('Highest finish (source)')).toHaveTextContent('120')
  const summary = within(screen.getByRole('region', { name: 'Match summary' }))
  expect(summary.getAllByText('Points scored')[0].nextElementSibling).toHaveTextContent('Unknown')
  fireEvent.change(screen.getByLabelText('Leg'), { target: { value: '1' } })
  for (const label of ['First-nine average (source)', '180s (source)', 'Highest finish (source)']) {
    expect(value(label)).toHaveTextContent('Not available')
  }
  expect(summary.getByText('81.3')).toBeVisible()
  fireEvent.change(screen.getByLabelText('Leg'), { target: { value: 'all' } })
  fireEvent.click(screen.getByRole('button', { name: syntheticMatch.players[1].label }))
  expect(value('180s (source)')).toHaveTextContent('Not available')
})

it.each([undefined, null, 0])('handles summary-only optional values: %s', value => {
  const match: MatchData = { ...syntheticMatch, detail: null,
    players: [{ ...syntheticMatch.players[0], stats: {
      match_average: null, points_scored: null, darts_thrown: null, checkout_hits: null, checkout_attempts: null,
      first_nine_average: value, total_180: value, highest_finish: value,
    } }, syntheticMatch.players[1]],
  }
  render(<MatchAnalysis status="ready" match={match} />)
  for (const label of ['First-nine average (source)', '180s (source)', 'Highest finish (source)']) {
    expect(screen.getAllByText(label)[0].nextElementSibling).toHaveTextContent(value == null ? 'Not available' : '0')
  }
})
