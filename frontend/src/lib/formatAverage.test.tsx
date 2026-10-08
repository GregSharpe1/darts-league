import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { formatAverage } from './api'
import { AdminFixtureCard } from '../pages/admin/AdminFixtureCard'
import { AuditEntryCard } from '../pages/admin/AuditEntryCard'
import { CompletedResults } from '../pages/home/CompletedResults'

afterEach(cleanup)

it.each([
  [undefined, ''],
  [null, ''],
  [0, '0.0'],
  [61.27, '61.3'],
] as const)('formats average %s without treating missing data as zero', (value, expected) => {
  expect(formatAverage(value)).toBe(expected)
})

const result = { player_one_legs: 3, player_two_legs: 1, player_one_average: null, player_two_average: 0, winner_id: 1 }
const fixture = { id: 1, player_one: 'Morgan', player_two: 'Casey', scheduled_at: '2026-06-15T09:00:00Z', game_variant: '501', legs_to_win: 3, status: 'played', result }

it.each([null, undefined])('keeps unknown average %s blank when editing and saving a score', async (average) => {
  const onSave = vi.fn().mockResolvedValue(undefined)
  render(<AdminFixtureCard fixture={{ ...fixture, result: { ...result, player_one_average: average } }} onSave={onSave} onUndo={vi.fn()} isSaving={false} isUndoing={false} />)
  expect(screen.getByLabelText('Morgan average')).toHaveValue('')
  expect(screen.getByLabelText('Casey average')).toHaveValue('0.0')
  fireEvent.click(screen.getByRole('button', { name: 'Save score' }))
  await waitFor(() => expect(onSave).toHaveBeenCalledWith({ fixtureId: 1, playerOneLegs: 3, playerTwoLegs: 1, playerOneAverage: undefined, playerTwoAverage: 0 }))
})

it('renders audit history with null, omitted and zero averages', () => {
  render(<AuditEntryCard entry={{ id: 1, fixture_id: 1, action: 'result_edited', actor: 'admin', created_at: fixture.scheduled_at, old_result: { player_one_legs: 3, player_two_legs: 0, winner_id: 1 }, new_result: result }} />)
  expect(screen.getByText('Fixture #1 - 3-0 -> 3-1 ( / 0.0)')).toBeVisible()
})

it('renders unknown public averages as placeholders rather than zero', () => {
  render(<CompletedResults weeks={[{ week_number: 1, status: 'unlocked', reveal_at: fixture.scheduled_at, fixtures: [fixture, { ...fixture, id: 2, result: { player_one_legs: 3, player_two_legs: 0, winner_id: 1 } }] }]} />)
  expect(screen.getByText('Averages: - / 0.0')).toBeVisible()
  expect(screen.getByText('Averages: - / -')).toBeVisible()
})
