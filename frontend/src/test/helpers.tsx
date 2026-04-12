import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { vi } from 'vitest'
import App from '../App'

export function renderApp(route: string) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={[route]}>
        <App />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

export function response(body: unknown, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => (status === 204 ? '' : JSON.stringify(body)),
  }
}

export type AppState = {
  authenticated: boolean
  seasonStarted: boolean
  seasonName: string
}

export function createMockFetch(state: AppState) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
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
  })
}
