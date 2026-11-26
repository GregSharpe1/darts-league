import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { afterEach, expect, it, vi } from 'vitest'
import { PendingResultsPage } from './PendingResultsPage'

afterEach(() => { cleanup(); vi.unstubAllGlobals() })

const mapping = { 'source-a': 31, 'source-b': 32 }
const candidate = {
  tournament_slug: 'east-cup', tournament_name: 'East Cup', match_number: 2,
  players: ['Alice (Flight)', 'Bob (Flight)'], mappings: [mapping],
}

function renderInbox(
  candidates: typeof candidate[],
  competition = 'tournament',
  playersResponse = () => Response.json({ error: { code: 'season_not_found', message: 'League data unavailable' } }, { status: 404 }),
) {
  const pending = {
    id: 7, external_match_id: 'demo', player_one_name: 'Alice', player_two_name: 'Bob',
    player_one_legs: 3, player_two_legs: 1, status: 'pending', received_at: '2026-06-16T12:00:00Z',
    competition_type: competition, pending_reason: candidates.length ? 'Auto-submit is disabled.' : 'No eligible unplayed tournament match.',
  }
  const source = {
    pending_result: pending, source: 'autodarts', digest: 'demo', changed_import: false,
    settings_evidence: 'source_reported', review_reason: '', played_at: '2026-06-16T11:00:00Z',
    players: [
      { match_player_id: 'source-a', display_name: 'Alice', legs_won: 3 },
      { match_player_id: 'source-b', display_name: 'Bob', legs_won: 1 },
    ],
    original_payload: {}, source_active: false, tournament_candidates: candidates,
  }
  const requests: unknown[] = []
  const fetch = vi.fn(async (url: string, options?: RequestInit) => {
    if (url === '/api/admin/players') return playersResponse()
    if (url === '/api/admin/pending-results') return Response.json({ pending_results: requests.length ? [] : [pending] })
    if (url === '/api/admin/pending-results/7') return Response.json(source)
    if (url.endsWith('/confirm')) { requests.push(JSON.parse(String(options?.body))); return Response.json({ winner_id: 31 }) }
    if (url === '/api/season') return Response.json({ status: 'completed', season_started: true })
    return Response.json({ error: { message: 'League data unavailable' } }, { status: 404 })
  })
  vi.stubGlobal('fetch', fetch)
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(<QueryClientProvider client={client}><PendingResultsPage /></QueryClientProvider>)
  return requests
}

it('preselects the only tournament match and submits its entrant IDs while the league is closed', async () => {
  const requests = renderInbox([candidate])
  const match = await screen.findByLabelText('Tournament and bracket match')
  expect(match).toHaveValue('east-cup/2')
  expect(screen.getByText('Auto-submit is disabled.')).toBeVisible()
  fireEvent.click(screen.getByRole('button', { name: 'Approve result' }))
  expect(await screen.findByText('Result approved. Tournament bracket refreshed.')).toBeVisible()
  expect(requests).toEqual([{
    tournament_slug: 'east-cup', match_number: 2, mapping, expected_result: null,
    replace: false, reason: '', attest_format: false, missing_date_reason: '',
  }])
})

it('requires an explicit destination for legacy receipts and a choice among matching brackets', async () => {
  renderInbox([candidate, { ...candidate, tournament_slug: 'west-cup', tournament_name: 'West Cup' }], '')
  const type = await screen.findByLabelText('Competition type')
  const approve = screen.getByRole('button', { name: 'Approve result' })
  expect(approve).toBeDisabled()
  fireEvent.change(type, { target: { value: 'tournament' } })
  const bracket = screen.getByLabelText('Tournament and bracket match')
  expect(bracket).toHaveValue('')
  expect(approve).toBeDisabled()
  fireEvent.change(bracket, { target: { value: 'west-cup/2' } })
  expect(approve).toBeEnabled()
})

it('explains when no tournament match can be approved', async () => {
  renderInbox([])
  expect(await screen.findByText('No eligible unplayed tournament match.')).toBeVisible()
  expect(screen.getByRole('button', { name: 'Approve result' })).toBeDisabled()
})

it('loads tournament review after an authenticated league server error', async () => {
  renderInbox([candidate], 'tournament', () => Response.json({
    error: { code: 'internal_error', message: 'League data unavailable' },
  }, { status: 500 }))
  expect(await screen.findByLabelText('Tournament and bracket match')).toHaveValue('east-cup/2')
})

it.each([
  { status: 401, message: 'Log in from the admin page to review pending results.' },
  { status: 403, message: 'Log in from the admin page to review pending results.' },
  { status: 502, message: 'Gateway unavailable' },
  { status: undefined, message: 'Network unavailable' },
])('keeps the inbox gated when the access check fails ($status)', async ({ status, message }) => {
  const access = () => {
    if (status === undefined) throw new TypeError(message)
    return Response.json({ error: { message } }, { status })
  }
  renderInbox([candidate], 'tournament', access)
  expect(await screen.findByText(message)).toBeVisible()
  expect(vi.mocked(fetch).mock.calls.some(([url]) => String(url).startsWith('/api/admin/pending-results'))).toBe(false)
})
