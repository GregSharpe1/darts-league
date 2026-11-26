import { test, expect } from '@playwright/test'
import { mkdir } from 'node:fs/promises'
import path from 'node:path'

const players = ['Morgan Ember', 'Casey Vale'].map((name, index) => ({ id: index + 1, display_name: name, preferred_name: name, admin_label: name, status: 'assigned', division_id: 1 }))
const divisions = [{ id: 1, name: 'Premier Division', slug: 'premier', position: 1 }]
const result = { player_one_legs: 3, player_two_legs: 1, player_one_average: null, player_two_average: 0, winner_id: 1 }
const fixture = { id: 1, player_one: players[0].display_name, player_two: players[1].display_name, scheduled_at: '2026-06-15T09:00:00Z', game_variant: '501', legs_to_win: 3, result }
const pending = { ...result, id: 1, external_match_id: 'mock-match', competition_type: 'league', player_one_name: 'Morgan', player_two_name: 'Casey', status: 'pending', received_at: fixture.scheduled_at }
const pendingDetail = {
  pending_result: pending, source: 'autodarts', digest: 'mock', changed_import: false,
  settings_evidence: 'source_reported', review_reason: '', played_at: fixture.scheduled_at,
  original_payload: {}, source_active: false,
  players: [
    { match_player_id: 'morgan', display_name: 'Morgan', legs_won: 3, stats: null },
    { match_player_id: 'casey', display_name: 'Casey', legs_won: 1, stats: { match_average: 0 } },
  ],
}

for (const width of [375, 768, 1280]) {
  test(`existing navigation and shared controls at ${width}px`, async ({ page }, testInfo) => {
    await page.setViewportSize({ width, height: 960 })
    const errors = []
    let reviewing = false
    page.on('pageerror', error => errors.push(error.message))
    await page.route('**/api/**', route => {
      const url = new URL(route.request().url()).pathname
      if (url === '/api/season') return route.fulfill({ json: { id: 1, instance_name: 'Cardiff Darts League', name: 'Summer League', status: reviewing ? 'started' : 'completed', timezone: 'Europe/London', registration_open: false, season_started: true, admin_locked: true, player_count: 2, division_count: 1, assigned_count: 2, waitlist_count: 0, week_count: 1, game_variant: '501', legs_to_win: 3, games_per_week: 1, total_fixtures: 1, remaining_fixtures: reviewing ? 1 : 0 } })
      if (url === '/api/version') return route.fulfill({ json: { version: 'ui-test' } })
      if (url === '/api/tournaments') return route.fulfill({ json: [] })
      if (url === '/api/divisions' || url === '/api/admin/divisions') return route.fulfill({ json: { divisions } })
      if (url === '/api/admin/players') return route.fulfill({ json: { players } })
      if (url === '/api/divisions/premier/fixtures') return route.fulfill({ json: { current_week: 1, weeks: [{ week_number: 1, reveal_at: fixture.scheduled_at, status: 'unlocked', fixtures: [fixture] }] } })
      if (url === '/api/divisions/premier/standings') return route.fulfill({ json: { standings: players.map(player => ({ player: player.preferred_name, display_name: player.display_name, played: 1, won: player.id === 1 ? 1 : 0, lost: player.id === 1 ? 0 : 1, legs_for: player.id === 1 ? 3 : 1, legs_against: player.id === 1 ? 1 : 3, leg_difference: player.id === 1 ? 2 : -2, average: player.id === 1 ? null : 0, points: player.id === 1 ? 2 : 0 })) } })
      if (url === '/api/admin/pending-results') return route.fulfill({ json: { pending_results: [pending] } })
      if (url === '/api/admin/pending-results/1') return route.fulfill({ json: pendingDetail })
      if (url === '/api/admin/divisions/premier/fixtures') return route.fulfill({ json: { weeks: [{ week_number: 1, fixtures: [
        { ...fixture, result: undefined, player_one_id: 1, player_two_id: 2, expected_result: null },
      ] }] } })
      errors.push(`Unexpected API request: ${url}`)
      return route.fulfill({ status: 404, json: { error: { code: 'not_found', message: 'No real API used.' } } })
    })
    const capture = async name => {
      await page.evaluate(() => document.fonts.ready)
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      const output = process.env.CAPTURE_UI_SCREENSHOTS === '1'
        ? path.resolve('../docs/pr-screenshots/feat-autodarts-01-ui-foundations')
        : testInfo.outputPath('screenshots')
      await mkdir(output, { recursive: true })
      await page.screenshot({ path: path.join(output, `${name}-${width}.png`), fullPage: true })
    }

    await page.goto('/')
    await page.getByRole('link', { name: 'Browse divisions', exact: true }).click()
    await page.getByRole('link', { name: 'View fixtures', exact: true }).click()
    await expect(page).toHaveURL(/\/divisions\/premier$/)
    await expect(page.getByRole('heading', { name: 'Premier Division', level: 1 })).toBeVisible()
    await expect(page.getByText('Averages: - / 0.0', { exact: true })).toBeVisible()
    if (width === 1280) await capture('division')

    await page.getByRole('link', { name: 'Standings', exact: true }).click()
    await expect(page.locator('tbody tr')).toHaveCount(2)
    await expect(page.locator('tbody tr').first().getByRole('cell', { name: '-', exact: true })).toBeVisible()
    await expect(page.locator('tbody tr').last().getByRole('cell', { name: '0.00', exact: true })).toBeVisible()

    reviewing = true
    await page.goto('/admin/pending-results')
    const select = page.getByLabel('League', { exact: true })
    const confirm = page.getByRole('button', { name: 'Approve result', exact: true })
    await expect(confirm).toBeDisabled()
    await expect(confirm).toHaveCSS('background-image', 'none')
    await expect(confirm).toHaveCSS('cursor', 'not-allowed')
    const averages = page.getByText('Match average (source)', { exact: true }).locator('xpath=following-sibling::dd[1]')
    await expect(averages).toHaveText(['Unknown', '0.0'])
    await page.keyboard.press('Tab')
    await select.focus()
    await expect(select).toHaveCSS('outline-style', 'solid')
    await expect(select).toHaveCSS('font-family', /Barlow/)
    await expect(select).toHaveCSS('border-radius', '10px')
    if (width === 375) await capture('pending-controls')
    await page.getByLabel('Season', { exact: true }).selectOption('1')
    await select.selectOption('1')
    await page.getByLabel('Player 1: Morgan', { exact: true }).selectOption('1')
    await page.getByLabel('Player 2: Casey', { exact: true }).selectOption('2')
    await page.getByLabel('Fixture', { exact: true }).selectOption('1')
    await expect(confirm).toBeEnabled()
    await expect(confirm).toHaveCSS('background-image', /linear-gradient/)
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await expect(page.locator('.brand-badge')).toHaveCSS('transition-duration', '0s')
    expect(errors).toEqual([])
  })
}
