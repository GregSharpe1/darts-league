import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import fixture from '../../lib/playerStatistics.fixture.json'
import { PlayerPage } from './PlayerPage'
import { MatchPage } from '../matches/MatchPage'

afterEach(() => { cleanup(); vi.unstubAllGlobals() })
function show(admin = false) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={client}><MemoryRouter initialEntries={[`${admin ? '/admin' : ''}/seasons/2/players/7`]}>
    <Routes><Route path={`${admin ? '/admin' : ''}/seasons/:seasonId/players/:playerId`} element={<PlayerPage admin={admin} />} /></Routes>
  </MemoryRouter></QueryClientProvider>)
}
function respond(value: unknown) { vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(value)))) }

it('renders partial metrics with known zero and scoped history links', async () => {
  respond(fixture)
  show()
  expect(await screen.findByRole('heading', { name: 'Arrow' })).toBeInTheDocument()
  expect(within(screen.getByRole('group', { name: '180s' })).getByText('0')).toBeInTheDocument()
  expect(within(screen.getByRole('group', { name: 'Dart-weighted average' })).getByText('Unavailable')).toBeInTheDocument()
  expect(within(screen.getByRole('group', { name: 'First-nine match average mean' })).getByText('1 of 2 eligible matches')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Match 12' })).toHaveAttribute('href', '/matches/12')
  expect(screen.getByRole('link', { name: 'Match 12' })).toHaveClass('button-link')
  expect(screen.getByRole('link', { name: 'The Comet' })).toHaveAttribute('href', '/seasons/2/players/9')
  expect(screen.queryByText(/locked/i)).not.toBeInTheDocument()
})

it('filters recorded entry types without plotting unverified automatic positions', async () => {
  respond(fixture)
  const { container } = show()
  await screen.findByRole('heading', { name: 'Arrow' })
  expect(container.querySelectorAll('[data-dart]')).toHaveLength(1)
  fireEvent.click(screen.getByRole('button', { name: 'Automatic' }))
  expect(container.querySelectorAll('[data-dart]')).toHaveLength(0)
  expect(screen.getByText('Coordinates unavailable')).toBeInTheDocument()
  expect(screen.getByText(/1 recorded throws; 1 known positions; 0 plottable/)).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Manual' }))
  expect(container.querySelectorAll('[data-dart]')).toHaveLength(1)
  fireEvent.click(screen.getByRole('button', { name: 'Unknown' }))
  expect(container.querySelectorAll('[data-dart]')).toHaveLength(0)
})

it('shows manual-only summaries and no fabricated detail', async () => {
  respond({ ...fixture, first_nine_match_average_mean: null, checkout_hits: null, checkout_attempts: null, checkout_percentage: null, total_180: null, highest_finish: null,
    coverage: { ...fixture.coverage, matches_with_detail: 0, first_nine_match_average_mean: 0, checkout: 0, total_180: 0, highest_finish: 0,
      matches_with_recorded_throws: 0, matches_with_positions: 0, recorded_throws: 0, known_positions: 0, plottable_positions: 0, position_fraction: null }, throws: [] })
  show()
  expect(await screen.findByText('42.50')).toBeInTheDocument()
  expect(screen.getByText('No recorded throws for this selection.')).toBeInTheDocument()
})

it('distinguishes zero attempts from a zero percentage', async () => {
  respond({ ...fixture, checkout_attempts: 0, checkout_percentage: null })
  show()
  expect(await screen.findByText('0 / 0 - no attempts')).toBeInTheDocument()
})

it.each([401, 404, 503])('sanitizes %s failures and offers appropriate recovery', async status => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('private-account', { status })))
  show(status === 401)
  if (status === 401) expect(await screen.findByRole('link', { name: 'Admin login' })).toHaveAttribute('href', '/admin')
  if (status === 404) expect(await screen.findByText('Player not found in this season.')).toBeInTheDocument()
  if (status === 503) expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument()
  expect(screen.queryByText('private-account')).not.toBeInTheDocument()
})

it('keeps admin history links in the authenticated scope', async () => {
  respond(fixture)
  show(true)
  expect(await screen.findByRole('link', { name: 'Match 12' })).toHaveAttribute('href', '/admin/matches/12')
  expect(screen.getByRole('link', { name: 'Match 12' })).toHaveClass('button-link')
})

it('shows an empty season without inventing metric values', async () => {
  const coverage = Object.fromEntries(Object.keys(fixture.coverage).map(key => [key, key === 'position_fraction' ? null : 0]))
  respond({ ...fixture, played: 0, won: 0, lost: 0, legs_for: 0, legs_against: 0, points: 0,
    match_average_mean: null, first_nine_match_average_mean: null, checkout_hits: null, checkout_attempts: null,
    checkout_percentage: null, total_180: null, highest_finish: null, coverage, history: [], throws: [] })
  show()
  expect(await screen.findByText(/No eligible matches yet/)).toBeInTheDocument()
  expect(screen.queryByRole('link', { name: /Match \d/ })).not.toBeInTheDocument()
  expect(screen.getAllByText('Unavailable')).toHaveLength(7)
})

it('shows loading then retries without exposing the failed payload', async () => {
  const fetcher = vi.fn().mockResolvedValueOnce(new Response('private', { status: 503 }))
    .mockResolvedValueOnce(new Response(JSON.stringify(fixture)))
  vi.stubGlobal('fetch', fetcher)
  show()
  expect(screen.getByText('Loading player statistics...')).toBeInTheDocument()
  fireEvent.click(await screen.findByRole('button', { name: 'Try again' }))
  expect(await screen.findByRole('heading', { name: 'Arrow' })).toBeInTheDocument()
  expect(fetcher).toHaveBeenCalledTimes(2)
})

it.each([false, true])('links returned match player IDs in the matching admin=%s scope', async admin => {
  respond({ schema_version: 'autodarts.api.v1', fixture_id: 12, season_id: 2, playedAt: null,
    players: [{ league_player_id: 7, preferred_name: 'Arrow', legs_won: 3, stats: null }, { league_player_id: 9, preferred_name: 'The Comet', legs_won: 1, stats: null }],
    detail: null, coverage: { recorded_throws: 0, known_positions: 0, position_fraction: null } })
  const client = new QueryClient()
  render(<QueryClientProvider client={client}><MemoryRouter initialEntries={['/matches/12']}>
    <Routes><Route path="/matches/:fixtureId" element={<MatchPage admin={admin} />} /></Routes>
  </MemoryRouter></QueryClientProvider>)
  expect(await screen.findByRole('link', { name: 'Arrow statistics' })).toHaveAttribute('href', `${admin ? '/admin' : ''}/seasons/2/players/7`)
  expect(screen.getByRole('link', { name: 'Arrow statistics' })).toHaveClass('button-link')
  expect(screen.queryByRole('link', { name: 'All divisions' })).not.toBeInTheDocument()
  if (admin) expect(screen.getByRole('link', { name: 'Admin home' })).toBeInTheDocument()
})

it('cancels an in-flight request when the page unmounts', () => {
  let signal: AbortSignal | null | undefined
  vi.stubGlobal('fetch', vi.fn((_url: string, options: RequestInit) => {
    signal = options.signal
    return new Promise(() => {})
  }))
  const view = show()
  view.unmount()
  expect(signal?.aborted).toBe(true)
})

it('refetches current values when revisiting after a result edit', async () => {
  const fetcher = vi.fn().mockResolvedValueOnce(new Response(JSON.stringify(fixture)))
    .mockResolvedValueOnce(new Response(JSON.stringify({ ...fixture, match_average_mean: 55 })))
  vi.stubGlobal('fetch', fetcher)
  const first = show()
  await screen.findByText('42.50')
  first.unmount()
  show()
  expect(await screen.findByText('55.00')).toBeInTheDocument()
  expect(screen.queryByText('42.50')).not.toBeInTheDocument()
  expect(fetcher).toHaveBeenCalledTimes(2)
})
