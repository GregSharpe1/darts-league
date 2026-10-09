import { test, expect } from '@playwright/test'
import { mkdir } from 'node:fs/promises'
import path from 'node:path'
import { expectedResult, fixture, mapReview, mockReview, source } from './pending-review-fixtures'

const output = path.resolve('test-results/pending-review')
async function capture(page, name) {
  await mkdir(output, { recursive: true })
  await page.evaluate(() => document.fonts.ready)
  expect(await page.evaluate(() => document.fonts.check('16px Barlow') && document.fonts.check('700 24px Rajdhani'))).toBe(true)
  await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }))
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  await page.screenshot({ path: path.join(output, `${name}-${page.viewportSize().width}.png`), fullPage: true })
}

for (const width of [375, 768, 1280]) {
  test(`selected review, mapping and approval at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 960 })
    const { requests } = await mockReview(page)
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    await page.goto('/admin/pending-results')
    await expect(page.getByRole('region', { name: 'Match summary' })).toBeVisible()
    expect(requests.filter(request => /\/pending-results\/\d+$/.test(request.url)).map(request => request.url)).toEqual(['/api/admin/pending-results/7'])
    await expect(page.getByRole('button', { name: 'Approve result', exact: true })).toBeDisabled()
    const summary = page.getByRole('button', { name: 'Summary', exact: true })
    const analysis = page.getByRole('button', { name: 'Analysis & heatmap', exact: true })
    const fetch = page.getByRole('button', { name: 'Fetch new results', exact: true })
    for (const button of [summary, analysis, fetch]) {
      await expect(button).toHaveCSS('border-radius', '999px')
      expect(await button.evaluate(el => el.getBoundingClientRect().height)).toBeGreaterThanOrEqual(44)
    }
    await expect(summary).toHaveCSS('background-image', /linear-gradient/)
    await expect(analysis).toHaveCSS('background-image', 'none')
    await expect(fetch).toHaveCSS('background-image', /linear-gradient/)
    await capture(page, 'unmapped')
    await page.getByRole('button', { name: 'Analysis & heatmap' }).focus()
    await page.keyboard.press('Enter')
    await expect(page.locator('[data-dart]')).toHaveCount(27)
    await expect(analysis).toHaveCSS('background-image', /linear-gradient/)
    await expect(summary).toHaveCSS('background-image', 'none')
    await page.getByRole('button', { name: 'Casey Vale', exact: true }).click()
    await page.getByLabel('Leg', { exact: true }).selectOption('3')
    await expect(page.locator('[data-dart]')).toHaveCount(6)
    await capture(page, 'analysis')
    await page.getByRole('button', { name: 'Summary', exact: true }).click()
    await mapReview(page)
    await expect(page.getByText('0 - 3', { exact: true })).toBeVisible()
    await page.getByLabel('Review note', { exact: true }).fill('Source clock and fixture checked')
    await capture(page, 'mapped')
    await page.getByRole('button', { name: 'Approve result', exact: true }).focus()
    await page.keyboard.press('Enter')
    await expect(page.getByText('Result approved. Fixture and standings refreshed.')).toBeVisible()
    const posts = requests.filter(request => request.url.endsWith('/confirm'))
    expect(posts).toHaveLength(1)
    expect(posts[0].body).toEqual({ season_id: 4, fixture_id: 90, mapping: Object.fromEntries(source.players.map((player, index) => [player.match_player_id, index + 11])), expected_result: null, replace: false, reason: 'Source clock and fixture checked', attest_format: false, missing_date_reason: '' })
    expect(requests.filter(request => request.url === '/api/season').length).toBeGreaterThan(1)
    expect(errors).toEqual([])
  })

  test(`replacement and stale snapshot at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 960 })
    const { state, requests } = await mockReview(page, { fixture: { ...fixture, result: expectedResult, expected_result: expectedResult }, conflict: true })
    await page.goto('/admin/pending-results')
    await mapReview(page)
    const approve = page.getByRole('button', { name: 'Approve result', exact: true })
    await expect(approve).toBeDisabled()
    await page.getByLabel('Replace the existing result or changed source import').check()
    await expect(approve).toBeDisabled()
    await page.getByLabel('Replacement reason').fill('Corrected source capture')
    await capture(page, 'replacement')
    await approve.click()
    await expect(page.getByText('The result changed while you reviewed it.')).toBeVisible()
    await expect(approve).toBeDisabled()
    await capture(page, 'conflict')
    expect(requests.find(request => request.url.endsWith('/confirm')).body.expected_result).toEqual(expectedResult)
    state.fixture.expected_result = { ...expectedResult, updated_at: '2026-06-15T12:00:00Z', player_two_legs: 1 }
    state.fixture.result = state.fixture.expected_result
    state.conflict = false
    await page.getByRole('button', { name: 'Reload source and fixture' }).click()
    await expect(page.getByLabel('Season', { exact: true })).toHaveValue('')
    await mapReview(page)
    await expect(page.getByLabel('Replace the existing result or changed source import')).not.toBeChecked()
    await page.getByLabel('Replace the existing result or changed source import').check()
    await page.getByLabel('Replacement reason').fill('Reviewed refreshed result')
    await approve.click()
    await expect(page.getByText('Result approved. Fixture and standings refreshed.')).toBeVisible()
    const body = requests.filter(request => request.url.endsWith('/confirm')).at(-1).body
    expect(body.expected_result).toEqual(state.fixture.expected_result)
    expect(body.replace).toBe(true)
    expect(body.reason).toBe('Reviewed refreshed result')
  })
}

test('blocks conflicting source, escapes labels and rejects with reason using keyboard', async ({ page }) => {
  const blocked = { ...source, pending_result: { ...source.pending_result, status: 'review_blocked', player_one_name: '<img src=x onerror=alert(1)>' }, review_reason: 'Leg totals conflict' }
  const { requests } = await mockReview(page, { source: blocked })
  await page.goto('/admin/pending-results')
  await expect(page.getByText('Source conflict - approval blocked')).toBeVisible()
  await expect(page.getByRole('region', { name: 'Match summary' })).toHaveCount(0)
  await expect(page.locator('img')).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Approve result', exact: true })).toBeDisabled()
  await capture(page, 'blocked')
  await page.getByText('Reject this import', { exact: true }).focus()
  await page.keyboard.press('Enter')
  await page.getByLabel('Rejection reason').fill('Conflicting capture; request a corrected import')
  await page.getByRole('button', { name: 'Reject result', exact: true }).focus()
  await page.keyboard.press('Enter')
  await expect(page.getByText('Result rejected. The league score was not changed.')).toBeVisible()
  expect(requests.find(request => request.url.endsWith('/reject')).body).toEqual({ reason: 'Conflicting capture; request a corrected import' })
})

test('requires legacy attestation and date reason; prevents invalid pair', async ({ page }) => {
  const legacy = { ...source, detail: null, played_at: null, settings_evidence: 'legacy_stored_summary', players: source.players.map(player => ({ ...player, stats: null })) }
  const { requests } = await mockReview(page, { source: legacy })
  await page.goto('/admin/pending-results')
  await mapReview(page)
  const approve = page.getByRole('button', { name: 'Approve result', exact: true })
  await expect(approve).toBeDisabled()
  await page.getByLabel('I verified this legacy match').check()
  await expect(approve).toBeDisabled()
  await page.getByLabel('Missing source date reason').fill('Both players verified the scheduled match')
  await expect(approve).toBeEnabled()
  await page.getByLabel('Player 2:', { exact: false }).selectOption('13')
  await expect(approve).toBeDisabled()
  await expect(page.getByText('No fixture matches this exact pair')).toBeVisible()
  await expect(page.getByLabel('Player 2:', { exact: false }).locator('option[value="11"]')).toBeDisabled()
  await page.getByLabel('Player 2:', { exact: false }).selectOption('12')
  await page.getByLabel('Fixture', { exact: true }).selectOption('90')
  await capture(page, 'legacy')
  await approve.click()
  await expect(page.getByText('Result approved. Fixture and standings refreshed.')).toBeVisible()
  expect(requests.find(request => request.url.endsWith('/confirm')).body).toMatchObject({ attest_format: true, missing_date_reason: 'Both players verified the scheduled match' })
})

test('retries selected detail failures and only fetches selected imports', async ({ page }) => {
  const { state, requests } = await mockReview(page, { detailStatus: 503 })
  await page.goto('/admin/pending-results')
  await expect(page.getByText('Source temporarily unavailable.')).toBeVisible()
  await capture(page, 'detail-error')
  state.detailStatus = 200
  await page.getByRole('button', { name: 'Retry selected result' }).click()
  await expect(page.getByRole('region', { name: 'Match summary' })).toBeVisible()
  await page.getByRole('button', { name: /Other source/ }).click()
  await expect(page.getByRole('button', { name: /Other source/ })).toHaveAttribute('aria-pressed', 'true')
  await expect.poll(() => requests.filter(request => request.url === '/api/admin/pending-results/8').length).toBe(1)
  await page.getByRole('button', { name: 'Fetch new results' }).click()
  await expect(page.getByText('Fetch complete. Inbox refreshed.')).toBeVisible()
  expect(requests.filter(request => request.url.endsWith('/poll'))).toHaveLength(1)
})

test('missing expected snapshot cannot be approved', async ({ page }) => {
  const { expected_result, ...oldFixture } = fixture
  expect(expected_result).toBeNull()
  await mockReview(page, { fixture: oldFixture })
  await page.goto('/admin/pending-results')
  await mapReview(page)
  await expect(page.getByText('Expected result snapshot is unavailable. Reload before approving.')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Approve result', exact: true })).toBeDisabled()
})

test('unauthorized users do not request imports', async ({ page }) => {
  const { requests } = await mockReview(page, { authenticated: false })
  await page.goto('/admin/pending-results')
  await expect(page.getByText('Log in from the admin page to review pending results.')).toBeVisible()
  expect(requests.some(request => request.url.includes('/pending-results'))).toBe(false)
})

test('blocks repeated activation while a confirm request is in flight', async ({ page }) => {
  const { requests } = await mockReview(page)
  let release
  const response = new Promise(resolve => { release = resolve })
  let confirms = 0
  await page.route('**/pending-results/7/confirm', async route => {
    confirms++
    await response
    await route.fallback()
  })
  await page.goto('/admin/pending-results')
  await mapReview(page)
  const approve = page.getByRole('button', { name: 'Approve result', exact: true })
  await approve.evaluate(button => { button.click(); button.click() })
  await expect.poll(() => confirms).toBe(1)
  await expect(page.getByRole('button', { name: 'Approving...' })).toBeDisabled()
  await expect(page.getByRole('button', { name: 'Fetch new results' })).toBeDisabled()
  await expect(page.getByRole('button', { name: /Other source/ })).toBeDisabled()
  release()
  await expect(page.getByText('Result approved. Fixture and standings refreshed.')).toBeVisible()
  expect(requests.filter(request => request.url.endsWith('/confirm'))).toHaveLength(1)
})

for (const width of [375, 768, 1280]) {
  test(`honest missing data states at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 960 })
    const { state } = await mockReview(page)
    state.source.detail.legs.forEach(leg => leg.visits.forEach(visit => visit.throws.forEach(dart => { dart.position = null })))
    state.source.players.forEach(player => { player.stats = null })
    await page.goto('/admin/pending-results')
    await page.getByRole('button', { name: 'Analysis & heatmap' }).click()
    await expect(page.getByText('Coordinates unavailable')).toBeVisible()
    await expect(page.locator('[data-dart], [data-heat]')).toHaveCount(0)
    await capture(page, 'missing-coordinates')
    state.source = { ...state.source, settings_evidence: 'legacy_stored_summary', played_at: null, detail: null }
    await page.reload()
    await expect(page.getByLabel('Missing source date reason')).toBeVisible()
    await capture(page, 'legacy')
    state.source.review_reason = 'Source totals conflict'
    state.source.pending_result = { ...state.source.pending_result, status: 'review_blocked' }
    await page.reload()
    await expect(page.getByText('Source conflict - approval blocked')).toBeVisible()
    await capture(page, 'blocked')
  })
}
