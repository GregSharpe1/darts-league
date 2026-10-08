import { afterEach, expect, it } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { CompletedResults } from '../home/CompletedResults'
import { AdminFixtureCard } from '../admin/AdminFixtureCard'

afterEach(cleanup)

const fixture = { id: 7, player_one: 'Arrow', player_two: 'Bob', scheduled_at: '2026-03-30T08:00:00Z', game_variant: '501', legs_to_win: 3, status: 'recorded', result: { player_one_legs: 3, player_two_legs: 1, winner_id: 11 } }

it('links only recorded revealed public results, including completed seasons', () => {
  render(<MemoryRouter><CompletedResults weeks={[
    { week_number: 1, status: 'unlocked', reveal_at: fixture.scheduled_at, fixtures: [fixture, { ...fixture, id: 8, result: undefined }] },
    { week_number: 2, status: 'locked', reveal_at: fixture.scheduled_at, fixtures: [{ ...fixture, id: 9 }] },
  ]} /></MemoryRouter>)
  expect(screen.getAllByRole('link')).toHaveLength(1)
  expect(screen.getByRole('link')).toHaveAttribute('href', '/matches/7')
})

it.each([true, false])('links recorded admin scores even when read-only is %s', readOnly => {
  render(<MemoryRouter><AdminFixtureCard fixture={fixture} onSave={async () => {}} onUndo={async () => {}} isSaving={false} isUndoing={false} isLocked readOnly={readOnly} /></MemoryRouter>)
  expect(screen.getByRole('link', { name: /view match/i })).toHaveAttribute('href', '/admin/matches/7')
})

it('does not link an unrecorded admin fixture', () => {
  render(<MemoryRouter><AdminFixtureCard fixture={{ ...fixture, result: undefined }} onSave={async () => {}} onUndo={async () => {}} isSaving={false} isUndoing={false} /></MemoryRouter>)
  expect(screen.queryByRole('link')).not.toBeInTheDocument()
})
