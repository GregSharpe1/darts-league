import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { StandingsPage } from '../standings/StandingsPage'

const state = vi.hoisted((): { seasonId: number | undefined; playerId: number | undefined } => ({ seasonId: 2, playerId: 7 }))
vi.mock('../../lib/api', () => ({
  useSeasonSummary: () => ({ data: { id: state.seasonId, name: 'Season two' } }),
  useDivisions: () => ({ data: [] }),
  useDivisionStandings: () => ({ data: [{ player_id: state.playerId, player: 'Arrow', display_name: 'Alice', played: 1, won: 1, lost: 0, legs_for: 3, legs_against: 1, leg_difference: 2, points: 2 }] }),
}))
afterEach(() => { cleanup(); state.seasonId = 2; state.playerId = 7 })

it('links stable player and season IDs without changing ranking or columns', () => {
  render(<MemoryRouter><StandingsPage /></MemoryRouter>)
  expect(screen.getByRole('link', { name: 'Arrow' })).toHaveAttribute('href', '/seasons/2/players/7')
  expect(screen.getByLabelText('Position 1')).toHaveTextContent('1')
  expect(screen.getAllByRole('columnheader')).toHaveLength(9)
})

it.each(['season', 'player'])('does not resolve names into links when %s ID is absent', missing => {
  if (missing === 'season') state.seasonId = undefined
  else state.playerId = undefined
  render(<MemoryRouter><StandingsPage /></MemoryRouter>)
  expect(screen.queryByRole('link')).not.toBeInTheDocument()
  expect(screen.getByText('Arrow')).toBeInTheDocument()
})
