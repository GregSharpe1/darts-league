import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App'

describe('App', () => {
  beforeEach(() => {
    const state = { authenticated: false, seasonStarted: false, seasonName: 'MVP Season' }

    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = typeof input === 'string' ? input : input.toString()
      const method = init?.method ?? 'GET'

      if (path === '/api/season') {
        return response({
          id: 1,
          instance_name: 'Cardiff Office - Darts League',
          name: state.seasonName,
          status: state.seasonStarted ? 'started' : 'registration_open',
          timezone: 'Europe/London',
          registration_open: !state.seasonStarted,
          player_count: 4,
          week_count: state.seasonStarted ? 3 : 0,
          game_variant: '501',
          legs_to_win: 3,
          games_per_week: 1,
          total_fixtures: state.seasonStarted ? 6 : 0,
        })
      }

      if (path === '/api/fixtures') {
        return response({
          current_week: 1,
          weeks: [
            {
              week_number: 1,
              status: 'unlocked',
              reveal_at: 'Mon, 23 Mar 2026 09:00:00 GMT',
              fixtures: [
                { id: 1, player_one: 'The Freeze', player_two: 'Bully Boy', scheduled_at: 'Mon, 23 Mar 2026 19:30:00 GMT', game_variant: '501', legs_to_win: 3, result: { player_one_legs: 3, player_two_legs: 1, player_one_average: 96.4, player_two_average: 89.3, winner_id: 1 } },
                { id: 4, player_one: 'The Asp', player_two: 'The Ferret', scheduled_at: 'Tue, 24 Mar 2026 19:30:00 GMT', game_variant: '501', legs_to_win: 3 },
              ],
            },
            {
              week_number: 2,
              status: 'unlocked',
              reveal_at: 'Mon, 30 Mar 2026 09:00:00 GMT',
              fixtures: [{ id: 3, player_one: 'Voltage', player_two: 'Snakebite', scheduled_at: 'Mon, 30 Mar 2026 19:30:00 GMT', game_variant: '501', legs_to_win: 3 }],
            },
            {
              week_number: 3,
              status: 'locked',
              reveal_at: 'Mon, 06 Apr 2026 09:00:00 GMT',
              fixtures: [{ id: 2, player_one: "I knew you'd look", player_two: 'Nothing to see here' }],
            },
          ],
        })
      }

      if (path === '/api/standings') {
        return response({ standings: [] })
      }

      if (path === '/api/version') {
        return response({ version: 'v0.0.6' })
      }

      if (path === '/api/admin/login' && method === 'POST') {
        state.authenticated = true
        return response({ authenticated: true, actor: 'admin' })
      }

      if (path === '/api/admin/logout' && method === 'POST') {
        state.authenticated = false
        return response({ authenticated: false })
      }

      if (path === '/api/admin/players') {
        if (!state.authenticated) {
          return response({ error: { code: 'unauthorized', message: 'Admin login is required.' } }, 401)
        }
        return response({
          players: [
            { id: 1, display_name: 'Luke Humphries', preferred_name: 'The Freeze', admin_label: 'The Freeze (Luke Humphries)', registered_at: 'Mon, 16 Mar 2026 19:00:00 GMT' },
            { id: 2, display_name: 'Michael Smith', preferred_name: 'Bully Boy', admin_label: 'Bully Boy (Michael Smith)', registered_at: 'Mon, 16 Mar 2026 19:05:00 GMT' },
          ],
        })
      }

      if (path === '/api/admin/fixtures') {
        return response({
          weeks: [
            {
              week_number: 1,
              reveal_at: 'Mon, 23 Mar 2026 09:00:00 GMT',
              fixtures: [
                {
                  id: 1,
                  player_one: 'The Freeze (Luke Humphries)',
                  player_two: 'Bully Boy (Michael Smith)',
                  scheduled_at: 'Mon, 23 Mar 2026 19:30:00 GMT',
                  game_variant: '501',
                  legs_to_win: 3,
                  status: 'scheduled',
                  result: { player_one_legs: 3, player_two_legs: 1, player_one_average: 96.4, player_two_average: 89.3, winner_id: 1 },
                },
              ],
            },
          ],
        })
      }

      if (path === '/api/admin/audit') {
        return response({
          entries: [
            { id: 1, fixture_id: 1, fixture_label: 'The Freeze (Luke Humphries) vs Bully Boy (Michael Smith)', action: 'result_edited', actor: 'admin', created_at: 'Mon, 23 Mar 2026 20:45:00 GMT', old_result: { player_one_legs: 3, player_two_legs: 0, player_one_average: 92.1, player_two_average: 80.4, winner_id: 1 }, new_result: { player_one_legs: 3, player_two_legs: 1, player_one_average: 96.4, player_two_average: 89.3, winner_id: 1 } },
          ],
        })
      }

      if (path === '/api/admin/season/start' && method === 'POST') {
        state.seasonStarted = true
        return response({ id: 1, instance_name: 'Cardiff Office - Darts League', name: state.seasonName, status: 'started', timezone: 'Europe/London', registration_open: false, player_count: 4, week_count: 3, game_variant: '501', legs_to_win: 3, games_per_week: 1, total_fixtures: 6 })
      }

      if (path === '/api/admin/season' && method === 'PUT') {
        const body = JSON.parse(String(init?.body ?? '{}'))
        state.seasonName = body.name
        return response({ id: 1, instance_name: 'Cardiff Office - Darts League', name: state.seasonName, status: state.seasonStarted ? 'started' : 'registration_open', timezone: 'Europe/London', registration_open: !state.seasonStarted, player_count: 4, week_count: state.seasonStarted ? 3 : 0, game_variant: '501', legs_to_win: 3, games_per_week: 1, total_fixtures: state.seasonStarted ? 6 : 0 })
      }

      if (path === '/api/admin/season/config' && method === 'PUT') {
        return response({ id: 1, instance_name: 'Cardiff Office - Darts League', name: state.seasonName, status: 'registration_open', timezone: 'Europe/London', registration_open: true, player_count: 4, week_count: 0, game_variant: '501', legs_to_win: 3, games_per_week: 1, total_fixtures: 0 })
      }

      if (path === '/api/admin/season/presets') {
        return response({ presets: [{ games_per_week: 1, week_count: 3 }, { games_per_week: 2, week_count: 2 }, { games_per_week: 3, week_count: 1 }] })
      }

      if (path === '/api/admin/season/preview') {
        return response({ player_count: 4, game_variant: '501', legs_to_win: 3, games_per_week: 1, week_count: 3, total_fixtures: 6 })
      }

      if (path.startsWith('/api/admin/players/') && method === 'DELETE') {
        return response({}, 204)
      }

      if (path.startsWith('/api/admin/fixtures/1/result') && method === 'PUT') {
        return response({ id: 1, fixture_id: 1, player_one_legs: 3, player_two_legs: 2, winner_id: 1 })
      }

      if (path.startsWith('/api/admin/fixtures/1/result') && method === 'DELETE') {
        return response({}, 204)
      }

      if (path.startsWith('/api/admin/fixtures/1/result') && method === 'POST') {
        return response({ error: { code: 'result_exists', message: 'This fixture already has a recorded result.' } }, 409)
      }

      return response({})
    }))
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('renders live fixtures from the API', async () => {
    renderApp('/')

    expect(screen.getByRole('heading', { name: /fixtures with a little theatre/i })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /^admin$/i })).not.toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /week 1/i })).toBeInTheDocument()
    })

    expect(screen.getByRole('button', { name: /week 1/i })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText(/the asp vs the ferret/i)).toBeInTheDocument()
    expect(screen.getByText(/week 1 - 501 - first to 3 legs/i)).toBeInTheDocument()
    expect(screen.getByText(/arrange within the week/i)).toBeInTheDocument()
    expect(screen.getByText(/i knew you'd look vs nothing to see here/i)).toBeInTheDocument()
    expect(screen.getByText(/players registered before the season start action/i)).toBeInTheDocument()
    expect(screen.getByText(/cardiff office - darts league/i)).toBeInTheDocument()
    expect(await screen.findByText(/backend v0.0.6/i)).toBeInTheDocument()
    expect(screen.getByText(/frontend dev/i)).toBeInTheDocument()
    await waitFor(() => {
      expect(document.title).toBe('Cardiff Office - Darts League')
    })
  })

  it('groups unfinished unlocked fixtures by week and lets the user switch between weeks', async () => {
    renderApp('/')

    const weekOneButton = await screen.findByRole('button', { name: /week 1/i })
    const weekTwoButton = await screen.findByRole('button', { name: /week 2/i })

    expect(weekOneButton).toHaveAttribute('aria-expanded', 'true')
    expect(weekTwoButton).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByText(/the asp vs the ferret/i)).toBeInTheDocument()
    expect(screen.queryByText(/voltage vs snakebite/i)).not.toBeInTheDocument()

    fireEvent.click(weekTwoButton)

    expect(weekOneButton).toHaveAttribute('aria-expanded', 'false')
    expect(weekTwoButton).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText(/voltage vs snakebite/i)).toBeInTheDocument()
    expect(screen.queryByText(/the asp vs the ferret/i)).not.toBeInTheDocument()
  })

  it('gates admin tools behind login and reveals live admin data after authentication', async () => {
    renderApp('/admin')

    expect(await screen.findByRole('heading', { name: /^login$/i })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /^admin$/i })).not.toBeInTheDocument()

    fireEvent.change(screen.getByLabelText(/username/i), { target: { value: 'admin' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /unlock admin tools/i }))

    expect(await screen.findByRole('heading', { name: /registered players/i })).toBeInTheDocument()
    expect(await screen.findByText(/the freeze \(luke humphries\)/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/the freeze \(luke humphries\) legs/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/the freeze \(luke humphries\) average/i)).toHaveValue('96.4')
    expect(screen.getByRole('button', { name: /undo result/i })).toBeInTheDocument()
    expect(screen.getByText(/result edited/i)).toBeInTheDocument()
  })

  it('prevents saving an invalid scoreline for the fixture format', async () => {
    renderApp('/admin')

    fireEvent.change(screen.getByLabelText(/username/i), { target: { value: 'admin' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /unlock admin tools/i }))

    await screen.findByRole('heading', { name: /registered players/i })

    fireEvent.change(screen.getByLabelText(/the freeze \(luke humphries\) legs/i), { target: { value: '1' } })
    fireEvent.change(screen.getByLabelText(/bully boy \(michael smith\) legs/i), { target: { value: '1' } })

    expect(screen.getByText(/valid scores: 3-0 to 3-2/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /save score/i })).not.toBeDisabled()
  })

  it('locks admin roster controls after the season starts', async () => {
    renderApp('/admin')

    fireEvent.change(screen.getByLabelText(/username/i), { target: { value: 'admin' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /unlock admin tools/i }))

    await screen.findByRole('heading', { name: /registered players/i })
    expect(screen.getAllByRole('button', { name: /^delete$/i }).length).toBeGreaterThan(0)

    // Click "Start season" to open the confirmation dialog.
    fireEvent.click(screen.getAllByRole('button', { name: /start season/i })[0])

    // Wait for the confirmation dialog to appear with the preview data.
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /start season\?/i })).toBeInTheDocument()
    })

    // Click "Start Season" in the confirmation dialog to actually start.
    const confirmButtons = screen.getAllByRole('button', { name: /^start season$/i })
    fireEvent.click(confirmButtons[confirmButtons.length - 1])

    await waitFor(() => {
      expect(screen.getByText(/registration is locked and player deletion is now disabled/i)).toBeInTheDocument()
    })
    await waitFor(() => {
      expect(screen.queryAllByRole('link', { name: /register/i })).toHaveLength(0)
    })

    expect(screen.getAllByText(/roster locked/i).length).toBeGreaterThan(0)
    expect(screen.getAllByRole('button', { name: /start season/i })[0]).toBeDisabled()
  })

  it('lets the admin rename the league before the season starts', async () => {
    renderApp('/admin')

    fireEvent.change(screen.getByLabelText(/username/i), { target: { value: 'admin' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /unlock admin tools/i }))

    expect(await screen.findByRole('heading', { name: /league settings/i })).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText(/league name/i), { target: { value: 'Cardiff Premier League' } })
    fireEvent.click(screen.getByRole('button', { name: /save league name/i }))

    await waitFor(() => {
      expect(screen.getAllByText(/cardiff premier league/i).length).toBeGreaterThan(0)
    })
  })
})

function renderApp(route: string) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={[route]}>
        <App />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function response(body: unknown, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => (status === 204 ? '' : JSON.stringify(body)),
  }
}
