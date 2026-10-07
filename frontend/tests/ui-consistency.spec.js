import { test, expect } from '@playwright/test'
import { mkdir } from 'node:fs/promises'
import path from 'node:path'

const output = path.resolve('../docs/pr-screenshots/issue-47')
const players = ['Morgan Ember', 'Casey Vale'].map((name, index) => ({ id: index + 1, display_name: name, preferred_name: name, admin_label: name, status: 'assigned', division_id: 1 }))
const divisions = [{ id: 1, name: 'Premier Division', slug: 'premier', position: 1 }]
const fixture = { id: 1, player_one: players[0].display_name, player_two: players[1].display_name, scheduled_at: '2026-06-15T09:00:00Z', game_variant: '501', legs_to_win: 3, status: 'unplayed' }
const weeks = [{ week_number: 1, reveal_at: '2026-06-15T09:00:00Z', status: 'unlocked', fixtures: [fixture] }]
const pending = { id: 1, player_one_name: 'Morgan', player_two_name: 'Casey', player_one_legs: 3, player_two_legs: 1, player_one_average: 60, player_two_average: 52, received_at: '2026-06-15T10:00:00Z', status: 'pending' }

for (const width of [375, 768, 1280]) {
  test(`shared presentation at ${width}px`, async ({ page }) => {
    test.setTimeout(120_000)
    await mkdir(output, { recursive: true })
    await page.setViewportSize({ width, height: 960 })
    let authenticated = false
    let started = false
    let closed = false
    let pendingState = 'ready'
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    await page.route('**/api/**', async route => {
      const url = new URL(route.request().url()).pathname
      if (url.startsWith('/api/admin/') && !authenticated) return route.fulfill({ status: 401, json: { error: { code: 'unauthorized', message: 'Log in to continue.' } } })
      if (url === '/api/season') return route.fulfill({ json: { id: 1, instance_name: 'Cardiff Darts League', name: 'Summer League', status: closed ? 'completed' : started ? 'started' : 'registration', timezone: 'Europe/London', registration_open: !started, season_started: started, admin_locked: started, can_start_season: !started, can_close_season: false, can_create_next_season: closed, remaining_fixtures: 1, can_edit_settings: !started, can_edit_division_channel: !started, can_edit_divisions: !started, can_assign_players: !started, player_count: 2, week_count: 1, game_variant: '501', legs_to_win: 3, games_per_week: 1, total_fixtures: 1, division_count: 1, assigned_count: 2, waitlist_count: 0 } })
      if (url === '/api/version') return route.fulfill({ json: { version: 'ui-test' } })
      if (url.endsWith('/divisions')) return route.fulfill({ json: { divisions } })
      if (url.endsWith('/players')) return route.fulfill({ json: { players } })
      if (url.endsWith('/presets')) return route.fulfill({ json: { presets: [{ games_per_week: 1, week_count: 1 }] } })
      if (url.endsWith('/preview')) return route.fulfill({ json: { player_count: 2, game_variant: '501', legs_to_win: 3, games_per_week: 1, week_count: 1, total_fixtures: 1 } })
      if (url.endsWith('/fixtures')) return route.fulfill({ json: { current_week: 1, weeks } })
      if (url.endsWith('/standings')) return route.fulfill({ json: { standings: players.map(player => ({ player: player.preferred_name, display_name: player.display_name, played: 1, won: player.id === 1 ? 1 : 0, lost: player.id === 1 ? 0 : 1, legs_for: player.id === 1 ? 3 : 1, legs_against: player.id === 1 ? 1 : 3, leg_difference: player.id === 1 ? 2 : -2, average: 60, points: player.id === 1 ? 2 : 0 })) } })
      if (url.endsWith('/audit')) return route.fulfill({ json: { entries: [{ id: 1, fixture_id: 1, fixture_label: 'Morgan Ember vs Casey Vale', action: 'result_edited', actor: 'admin', created_at: '2026-06-15T11:00:00Z', old_result: { player_one_legs: 3, player_two_legs: 0 }, new_result: { player_one_legs: 3, player_two_legs: 1 } }] } })
      if (url.endsWith('/pending-results')) {
        if (pendingState === 'loading') await new Promise(resolve => setTimeout(resolve, 1500))
        if (pendingState === 'error') return route.fulfill({ status: 500, json: { error: { code: 'unavailable', message: 'Results temporarily unavailable.' } } })
        return route.fulfill({ json: { pending_results: pendingState === 'empty' ? [] : [pending] } })
      }
      return route.fulfill({ status: 404, json: { error: { code: 'not_found', message: 'Unknown test route.' } } })
    })
    const capture = async name => {
      await page.evaluate(() => document.fonts.ready)
      await expect(page.locator('h1')).toBeVisible()
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await page.screenshot({ path: path.join(output, `${name}-${width}.png`), fullPage: true })
    }
    for (const [url, name] of [['/', 'home'], ['/register', 'registration'], ['/admin', 'login']]) {
      await page.goto(url)
      await capture(name)
    }
    authenticated = true
    await page.goto('/admin')
    await expect(page.getByRole('heading', { name: 'League settings' })).toBeVisible()
    await capture('admin-setup')
    await page.goto('/admin/pending-results')
    await expect(page.getByLabel('League', { exact: true })).toBeVisible()
    await capture('pending-ready')
    const select = page.getByLabel('League', { exact: true })
    await page.keyboard.press('Tab')
    await select.focus()
    await expect(select).toHaveCSS('outline-style', 'solid')
    await expect(select).toHaveCSS('font-family', /Barlow/)
    expect(await page.locator(':root').evaluate(el => getComputedStyle(el).getPropertyValue('--radius-panel').trim())).toBe('20px')
    await expect(page.getByRole('button', { name: 'Confirm Score', exact: true })).toHaveCSS('background-image', 'none')
    for (const state of ['loading', 'empty', 'error']) {
      pendingState = state
      await page.goto('/admin/pending-results')
      await expect(page.getByText(state === 'loading' ? 'Loading pending results...' : state === 'empty' ? 'No results are waiting for confirmation.' : 'Results temporarily unavailable.')).toBeVisible({ timeout: 15_000 })
      await capture(`pending-${state}`)
    }
    started = true
    for (const [url, name] of [['/admin', 'admin-locked'], ['/admin/divisions/premier', 'scores-audit'], ['/divisions/premier', 'fixtures'], ['/divisions/premier/standings', 'standings'], ['/register', 'registration-closed']]) {
      await page.goto(url)
      await capture(name)
    }
    closed = true
    await page.goto('/admin')
    await capture('admin-completed')
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await expect(page.locator('.brand-badge')).toHaveCSS('transition-duration', '0s')
    expect(errors).toEqual([])
  })
}
