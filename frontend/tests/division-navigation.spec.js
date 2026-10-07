import { test, expect } from '@playwright/test'

for (const status of ['started', 'completed']) {
  test(`home opens a ${status} division with score-only results`, async ({ page }) => {
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    await page.route('**/api/**', route => {
      const url = new URL(route.request().url()).pathname
      if (url === '/api/season') return route.fulfill({ json: { id: 4, name: 'Mock league', status, instance_name: 'Mock navigation', registration_open: false, season_started: true, game_variant: '501', legs_to_win: 3, games_per_week: 1, total_fixtures: 2, remaining_fixtures: 0 } })
      if (url === '/api/version') return route.fulfill({ json: { version: 'mock' } })
      if (url === '/api/divisions') return route.fulfill({ json: { divisions: [{ id: 1, slug: 'premier', name: 'Premier Division', position: 1 }] } })
      if (url.endsWith('/standings')) return route.fulfill({ json: { standings: [] } })
      if (url.endsWith('/fixtures')) return route.fulfill({ json: { current_week: 1, weeks: [{ week_number: 1, status: 'unlocked', reveal_at: '2026-06-15T08:00:00Z', fixtures: [
        { id: 91, player_one: 'Morgan', player_two: 'Casey', result: { player_one_legs: 3, player_two_legs: 1, player_one_average: null, player_two_average: null, winner_id: 1 } },
        { id: 92, player_one: 'Casey', player_two: 'Rowan', result: { player_one_legs: 3, player_two_legs: 0, player_one_average: 0, winner_id: 2 } },
      ] }] } })
      return route.fulfill({ status: 404, json: { error: { code: 'mock_not_found', message: 'No real API used.' } } })
    })
    await page.goto('/')
    await page.getByRole('link', { name: 'View fixtures', exact: true }).click()
    await expect(page).toHaveURL(/\/divisions\/premier$/)
    await expect(page.getByRole('heading', { name: 'Premier Division', level: 1 })).toBeVisible()
    if (status === 'completed') {
      await expect(page.getByText('Averages: - / -', { exact: true })).toBeVisible()
      await expect(page.getByText('Averages: 0.0 / -', { exact: true })).toBeVisible()
    } else {
      await expect(page.getByText(/Every unlocked fixture in this division has been played/)).toBeVisible()
    }
    expect(errors).toEqual([])
  })
}
