import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import type { BracketMatch, Tournament } from '../../lib/tournamentsApi'
import { TournamentScoreEditor } from './TournamentScoreEditor'

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

it('requires confirmation before correcting a winner and clearing a later score', async () => {
  const alice = { player: 'Alice', entry_id: 1 }
  const bob = { player: 'Bob', entry_id: 2 }
  const cara = { player: 'Cara', entry_id: 3 }
  const semi: BracketMatch = { number: 1, slots: [alice, bob], legs: [3, 1], winner_id: 1 }
  const final: BracketMatch = { number: 2, slots: [{ ...alice, winner_of: 1 }, cara], legs: [3, 0], winner_id: 1 }
  const event: Tournament = {
    slug: 'winter-cup', name: 'Winter Cup', starts_on: '2027-01-10',
    registration_open: false, game_variant: '501', legs_to_win: 3,
    bracket: [[semi], [final]],
  }
  const requests: unknown[] = []
  vi.stubGlobal('fetch', vi.fn(async (url: string, options?: RequestInit) => {
    if (url.endsWith('/entries')) return Response.json([])
    requests.push({ url, method: options?.method, body: JSON.parse(String(options?.body)) })
    return Response.json(event)
  }))
  const confirm = vi.fn().mockReturnValueOnce(false).mockReturnValueOnce(true)
  vi.stubGlobal('confirm', confirm)
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(<QueryClientProvider client={client}><TournamentScoreEditor event={event} /></QueryClientProvider>)

  fireEvent.change(screen.getByLabelText('Recorded match'), { target: { value: '1' } })
  fireEvent.change(screen.getByLabelText('Alice legs'), { target: { value: '1' } })
  fireEvent.change(screen.getByLabelText('Bob legs'), { target: { value: '3' } })
  const save = screen.getByRole('button', { name: 'Save corrected score' })
  expect(save).toBeDisabled()
  fireEvent.change(screen.getByLabelText('Reason for correction'), { target: { value: 'Checked with both players' } })
  expect(screen.getByText('Changing the winner will clear the scores for later matches 2.')).toBeVisible()
  fireEvent.click(save)
  expect(requests).toEqual([])
  fireEvent.click(save)
  expect(await screen.findByText('Tournament score corrected.')).toBeVisible()
  expect(requests).toEqual([{
    url: '/api/admin/tournaments/winter-cup/matches/1/score',
    method: 'PUT',
    body: { legs: [1, 3], previous_legs: [3, 1], entrants: [1, 2], clear_matches: [2], reason: 'Checked with both players' },
  }])
})
