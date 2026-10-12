import { test, expect } from '@playwright/test'
import { mockReview, season, source, fixture, expectedResult } from './pending-review-fixtures'

const suggestion = { season_id: 4, division_id: 1, mapping: Object.fromEntries(source.players.map((player, index) => [player.match_player_id, index + 11])), fixture_id: 90 }

for (const width of [375, 768, 1280]) {
  test(`mapping keyboard CRUD after roster lock at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 960 })
    await mockReview(page)
    let mappings = []
    const writes = []
    await page.route('**/api/admin/autodarts-mappings**', route => {
      const request = route.request()
      if (request.method() === 'GET') return route.fulfill({ json: { season_id: 4, auto_submit_enabled: false, mappings } })
      writes.push(request.postDataJSON())
      if (request.method() === 'DELETE') { mappings = []; return route.fulfill({ status: 204 }) }
      mappings = [{ player_id: 11, username: request.postDataJSON().username }]
      return route.fulfill({ json: mappings[0] })
    })
    await page.goto('/admin')
    const section = page.getByRole('region', { name: 'Autodarts mappings' })
    await section.getByLabel('League player', { exact: true }).selectOption('11')
    await section.getByLabel('Autodarts username', { exact: true }).fill('  gsharpe  ')
    await section.getByRole('button', { name: 'Save mapping', exact: true }).focus()
    await page.keyboard.press('Enter')
    await expect(section.getByText('gsharpe', { exact: true })).toBeVisible()
    expect(writes[0]).toEqual({ season_id: 4, username: 'gsharpe' })
    await section.getByRole('button', { name: 'Edit', exact: true }).click()
    await section.getByLabel('Autodarts username', { exact: true }).fill('GSharpe')
    await section.getByRole('button', { name: 'Save mapping', exact: true }).click()
    await expect(section.getByText('GSharpe', { exact: true })).toBeVisible()
    await page.evaluate(() => document.fonts.ready)
    await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }))
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    await page.screenshot({ path: `test-results/autodarts-mappings/mappings-${width}.png`, fullPage: true })
    await section.getByRole('button', { name: 'Remove', exact: true }).click()
    await expect(section.getByText('No usernames mapped yet.')).toBeVisible()
    expect(writes.at(-1)).toEqual({ season_id: 4 })
  })
  test(`suggested review preserves reversed scores and consent at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 960 })
    const { requests } = await mockReview(page, { source: { ...source, suggestion }, fixture: { ...fixture, expected_result: expectedResult, result: expectedResult } })
    await page.goto('/admin/pending-results')
    await expect(page.getByLabel('Fixture', { exact: true })).toHaveValue('90')
    await expect(page.getByText('0 - 3', { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Approve result', exact: true })).toBeDisabled()
    await page.getByLabel('Replace the existing result or changed source import').check()
    await page.getByLabel('Replacement reason').fill('Checked suggested match')
    await page.evaluate(() => document.fonts.ready)
    await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }))
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    await page.screenshot({ path: `test-results/autodarts-mappings/prefill-${width}.png`, fullPage: true })
    await page.getByRole('button', { name: 'Approve result', exact: true }).click()
    await expect(page.getByText('Result approved. Fixture and standings refreshed.')).toBeVisible()
    expect(requests.find(request => request.url.endsWith('/confirm')).body.expected_result).toEqual(expectedResult)
  })
}

test('mapping conflict retains input and completed seasons allow mappings', async ({ page }) => {
  await mockReview(page)
  await page.route('**/api/season', route => route.fulfill({ json: { ...season, status: 'completed' } }))
  await page.route('**/api/admin/autodarts-mappings**', route => route.fulfill(route.request().method() === 'GET'
    ? { json: { season_id: 4, auto_submit_enabled: true, mappings: [] } }
    : { status: 409, json: { error: { code: 'conflict', message: 'Username already mapped.' } } }))
  await page.goto('/admin')
  await page.getByLabel('League player', { exact: true }).selectOption('11')
  await page.getByLabel('Autodarts username', { exact: true }).fill('gsharpe')
  await page.getByRole('button', { name: 'Save mapping', exact: true }).click()
  await expect(page.getByText('Username already mapped.')).toBeVisible()
  await expect(page.getByLabel('Autodarts username', { exact: true })).toHaveValue('gsharpe')
})

test('background refetch preserves manual review edits and explicit reload resets them', async ({ page }) => {
  await mockReview(page, { source: { ...source, suggestion } })
  await page.goto('/admin/pending-results')
  await expect(page.getByLabel('Fixture', { exact: true })).toHaveValue('90')
  await page.getByLabel('Player 2:', { exact: false }).selectOption('13')
  await page.getByLabel('Review note', { exact: true }).fill('Manual review in progress')
  await page.getByRole('button', { name: 'Fetch new results', exact: true }).click()
  await expect(page.getByText('Fetch complete. Inbox refreshed.')).toBeVisible()
  await expect(page.getByLabel('Player 2:', { exact: false })).toHaveValue('13')
  await expect(page.getByLabel('Review note', { exact: true })).toHaveValue('Manual review in progress')
  await page.getByRole('button', { name: 'Reload source and fixture' }).click()
  await expect(page.getByLabel('Fixture', { exact: true })).toHaveValue('90')
})

test('late fixture response cannot overwrite manual pair selection', async ({ page }) => {
  await mockReview(page, { source: { ...source, suggestion } })
  let release
  const gate = new Promise(resolve => { release = resolve })
  await page.route('**/api/admin/divisions/premier/fixtures', async route => {
    await gate
    await route.fulfill({ json: { weeks: [{ week_number: 1, fixtures: [fixture] }] } })
  })
  await page.goto('/admin/pending-results')
  await page.getByLabel('Player 2:', { exact: false }).selectOption('13')
  release()
  await expect(page.getByText('Loading fixtures...')).toHaveCount(0)
  await expect(page.getByLabel('Fixture', { exact: true })).toHaveValue('')
  await expect(page.getByLabel('Player 2:', { exact: false })).toHaveValue('13')
})

test('partial suggestion leaves unknown player and fixture unselected', async ({ page }) => {
  await mockReview(page, { source: { ...source, suggestion: { ...suggestion, mapping: { [source.players[0].match_player_id]: 11 }, fixture_id: null } } })
  await page.goto('/admin/pending-results')
  await expect(page.getByLabel('Player 1:', { exact: false })).toHaveValue('11')
  await expect(page.getByLabel('Player 2:', { exact: false })).toHaveValue('')
  await expect(page.getByLabel('Fixture', { exact: true })).toHaveValue('')
  await expect(page.getByRole('button', { name: 'Approve result', exact: true })).toBeDisabled()
})

test('old-season suggestions are ignored and incompatible fixtures are not selected', async ({ page }) => {
  const { state } = await mockReview(page, { source: { ...source, suggestion: { ...suggestion, season_id: 3 } } })
  await page.goto('/admin/pending-results')
  await expect(page.getByLabel('Season', { exact: true })).toHaveValue('')
  await expect(page.getByLabel('Player 1:', { exact: false })).toHaveValue('')
  state.source.suggestion.season_id = 4
  state.fixture.legs_to_win = 2
  await page.getByRole('button', { name: 'Reload source and fixture' }).click()
  await expect(page.getByLabel('Season', { exact: true })).toHaveValue('4')
  await expect(page.getByLabel('Fixture', { exact: true })).toHaveValue('')
})

test('stale season and unauthorized mapping writes preserve the draft', async ({ page }) => {
  await mockReview(page)
  let status = 409
  await page.route('**/api/admin/autodarts-mappings**', route => route.fulfill(route.request().method() === 'GET'
    ? { json: { season_id: 4, auto_submit_enabled: false, mappings: [] } }
    : { status, json: { error: { code: 'stale_or_unauthorized', message: status === 409 ? 'Season changed. Reload the admin page.' : 'Log in to continue.' } } }))
  await page.goto('/admin')
  await page.getByLabel('League player', { exact: true }).selectOption('11')
  await page.getByLabel('Autodarts username', { exact: true }).fill('gsharpe')
  await page.getByRole('button', { name: 'Save mapping', exact: true }).click()
  await expect(page.getByText('Season changed. Reload the admin page.')).toBeVisible()
  await expect(page.getByLabel('Autodarts username', { exact: true })).toHaveValue('gsharpe')
  status = 401
  await page.getByRole('button', { name: 'Save mapping', exact: true }).click()
  await expect(page.getByText('Log in to continue.')).toBeVisible()
  await expect(page.getByLabel('Autodarts username', { exact: true })).toHaveValue('gsharpe')
})

test('mapping controls stay disabled during requests and reject invalid names', async ({ page }) => {
  await mockReview(page)
  let release
  const gate = new Promise(resolve => { release = resolve })
  await page.route('**/api/admin/autodarts-mappings**', async route => {
    if (route.request().method() === 'GET') return route.fulfill({ json: { season_id: 4, auto_submit_enabled: false, mappings: [] } })
    await gate
    await route.fulfill({ json: { player_id: 11, username: 'gsharpe' } })
  })
  await page.goto('/admin')
  await page.getByLabel('League player', { exact: true }).selectOption('11')
  const input = page.getByLabel('Autodarts username', { exact: true })
  const save = page.getByRole('button', { name: 'Save mapping', exact: true })
  for (const name of ['   ', 'x'.repeat(81), 'x\u007fy']) {
    await input.fill(name)
    await expect(save).toBeDisabled()
  }
  await input.fill('gsharpe')
  await save.click()
  await expect(input).toBeDisabled()
  await expect(page.getByLabel('League player', { exact: true })).toBeDisabled()
  release()
  await expect(input).toBeEnabled()
})

test('changed source blocks approval without erasing the review draft', async ({ page }) => {
  const { state } = await mockReview(page, { source: { ...source, suggestion } })
  await page.goto('/admin/pending-results')
  await expect(page.getByLabel('Fixture', { exact: true })).toHaveValue('90')
  await page.getByLabel('Review note', { exact: true }).fill('Keep this draft')
  state.source.digest = 'changed-source'
  await page.getByRole('button', { name: 'Fetch new results', exact: true }).click()
  await expect(page.getByText('Fetch complete. Inbox refreshed.')).toBeVisible()
  await expect(page.getByLabel('Review note', { exact: true })).toHaveValue('Keep this draft')
  await expect(page.getByRole('button', { name: 'Approve result', exact: true })).toBeDisabled()
  await expect(page.getByText('The review is out of date. Reload source and fixture, then review again.')).toBeVisible()
})

test('mapping data for another season disables writes', async ({ page }) => {
  await mockReview(page)
  await page.route('**/api/admin/autodarts-mappings', route => route.fulfill({ json: { season_id: 5, auto_submit_enabled: false, mappings: [] } }))
  await page.goto('/admin')
  await expect(page.getByText('The current season changed. Reload the admin page before saving.')).toBeVisible()
  await expect(page.getByLabel('Autodarts username', { exact: true })).toBeDisabled()
  await expect(page.getByRole('button', { name: 'Save mapping', exact: true })).toBeDisabled()
})
