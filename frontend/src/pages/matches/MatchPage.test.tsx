import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { MatchPage } from './MatchPage'

afterEach(() => { cleanup(); vi.unstubAllGlobals() })

function show(admin = false) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={client}><MemoryRouter initialEntries={[admin ? '/admin/matches/7' : '/matches/7']}>
    <Routes><Route path={admin ? '/admin/matches/:fixtureId' : '/matches/:fixtureId'} element={<MatchPage admin={admin} />} /></Routes>
  </MemoryRouter></QueryClientProvider>)
}

it('shows loading while evidence is being fetched', () => {
  vi.stubGlobal('fetch', vi.fn(() => new Promise(() => {})))
  show()
  expect(screen.getByRole('status')).toBeInTheDocument()
})

it('renders the real shared summary and missing evidence state', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
    schema_version: 'autodarts.api.v1', fixture_id: 7, season_id: 2, playedAt: null,
    players: [{ league_player_id: 1, preferred_name: 'Arrow', legs_won: 3, stats: null }, { league_player_id: 2, preferred_name: 'Bob', legs_won: 1, stats: null }],
    detail: null, coverage: { recorded_throws: 0, known_positions: 0, position_fraction: null },
  }))))
  show()
  expect(await screen.findByRole('heading', { name: 'Throw detail unavailable' })).toBeInTheDocument()
  expect(screen.getAllByText('Arrow').length).toBeGreaterThan(0)
  expect(screen.getAllByText('Bob').length).toBeGreaterThan(0)
})

it.each([404, 401, 503])('renders sanitized %s without stale data or source errors', async status => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('private-source-secret', { status })))
  show(status === 401)
  if (status === 404) expect(await screen.findByText('Match not found.')).toBeInTheDocument()
  if (status === 401) expect(await screen.findByRole('link', { name: 'Admin login' })).toHaveAttribute('href', '/admin')
  if (status === 503) expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument()
  expect(screen.queryByText('private-source-secret')).not.toBeInTheDocument()
})
