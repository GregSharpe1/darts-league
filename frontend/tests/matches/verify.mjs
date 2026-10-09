import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { chromium } from '@playwright/test'
import { preview } from 'vite'
import { fixtureMatch, matchState } from './fixture.mjs'

const root = fileURLToPath(new URL('../../', import.meta.url))
const screenshots = fileURLToPath(new URL('./screenshots/', import.meta.url))
const fontRoot = new URL('../../../docs/mockups/autodarts/fonts/', import.meta.url)
const server = await preview({ root, preview: { host: '127.0.0.1', port: 0 } })
const address = server.httpServer.address()
assert(address && typeof address === 'object')
const base = `http://127.0.0.1:${address.port}`
const browser = await chromium.launch()
const errors = [], artifacts = []
const fixture = { id: 7, player_one: 'Morgan Ember', player_two: 'Casey Vale', scheduled_at: '2026-03-30T08:00:00Z', game_variant: '501', legs_to_win: 3, status: 'recorded', result: { player_one_legs: 3, player_two_legs: 0, winner_id: 101 } }
const weeks = [{ week_number: 1, reveal_at: fixture.scheduled_at, status: 'unlocked', fixtures: [fixture, { ...fixture, id: 8, result: undefined }] },
  { week_number: 2, reveal_at: '2026-04-06T08:00:00Z', status: 'locked', fixtures: [{ ...fixture, id: 9, result: undefined }] }]
try {
  await mkdir(screenshots, { recursive: true })
  for (const width of [375, 768, 1280]) {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    let state = 'analysis', release
    page.on('pageerror', error => errors.push(error.message))
    page.on('console', message => { if (message.type() === 'error' && !message.text().includes('Failed to load resource')) errors.push(message.text()) })
    await page.route('**/*', async route => {
      const url = new URL(route.request().url())
      if (url.hostname === 'fonts.googleapis.com') {
        const css = await readFile(new URL('fonts.css', fontRoot), 'utf8')
        return route.fulfill({ contentType: 'text/css', body: css.replaceAll('url(', `url(${base}/qa-fonts/`) })
      }
      if (url.pathname.startsWith('/qa-fonts/')) return route.fulfill({ contentType: 'font/ttf', body: await readFile(new URL(url.pathname.split('/').at(-1), fontRoot)) })
      if (url.origin !== base) { errors.push(`Unexpected external request: ${url.origin}`); return route.abort() }
      if (!url.pathname.startsWith('/api/')) return route.continue()
      let body, status = 200
      if (url.pathname.endsWith('/autodarts')) {
        if (state === 'loading') await new Promise(resolve => { release = resolve })
        status = state === 'missing' ? 404 : state === 'unauthorized' ? 401 : state === 'error' ? 503 : 200
        body = status === 200 ? matchState(state) : { error: { code: 'not_found', message: 'private-source-must-not-render' } }
      } else if (url.pathname === '/api/season') body = {
        id: 2, instance_name: 'Offline synthetic QA league', name: 'Synthetic season', status: state === 'completed-list' ? 'completed' : 'started', registration_open: false, season_started: true,
        game_variant: '501', legs_to_win: 3, games_per_week: 1, remaining_fixtures: 1, player_count: 2, week_count: 2, division_count: 1,
      }
      else if (url.pathname === '/api/version') body = { version: 'offline-qa' }
      else if (url.pathname.endsWith('/divisions')) body = { divisions: [{ id: 1, name: 'Test division', slug: 'test', position: 1 }] }
      else if (url.pathname.endsWith('/players')) body = { players: [] }
      else if (url.pathname.endsWith('/fixtures')) body = { current_week: 1, weeks }
      else if (url.pathname.endsWith('/standings')) body = { standings: [] }
      else if (url.pathname.endsWith('/audit')) body = { entries: [] }
      else { errors.push(`Unexpected API: ${url.pathname}`); status = 404; body = {} }
      return route.fulfill({ status, contentType: 'application/json', headers: { 'Cache-Control': 'no-store' }, body: JSON.stringify(body) })
    })
    const capture = async name => {
      await page.evaluate(() => document.fonts.ready)
      assert(await page.evaluate(() => document.fonts.check('16px Barlow') && document.fonts.check('700 24px Rajdhani')), 'Offline fonts loaded')
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `${name}/${width} overflow`)
      assert.equal(await page.getByText('private-source-must-not-render').count(), 0)
      const file = `${name}-${width}.png`
      await page.screenshot({ path: `${screenshots}/${file}`, fullPage: true })
      artifacts.push({ file, sha256: createHash('sha256').update(await readFile(`${screenshots}/${file}`)).digest('hex') })
    }
    for (const admin of [false, true]) {
      for (state of admin ? ['analysis', 'unauthorized'] : ['analysis', 'unknown', 'summary', 'missing', 'error', 'loading']) {
        await page.goto(`${base}${admin ? '/admin' : ''}/matches/7`)
        await page.getByRole('heading', { name: 'Match analysis', exact: true }).waitFor()
        if (state === 'loading') await page.getByRole('status').waitFor()
        else if (state === 'missing') await page.getByRole('heading', { name: 'Match not found.' }).waitFor()
        else if (state === 'unauthorized') await page.getByRole('link', { name: 'Admin login' }).waitFor()
        else if (state === 'error') await page.getByRole('button', { name: 'Try again' }).waitFor()
        else await page.getByRole('region', { name: 'Match summary' }).waitFor()
        if (admin) assert.equal(await page.getByRole('navigation').getByRole('link', { name: 'Divisions', exact: true }).count(), 0)
        if (state === 'analysis') assert.equal(await page.locator('[data-dart]').count(), 27)
        if (state === 'unknown') assert.equal(await page.locator('[data-dart]').count(), 0)
        assert.equal(await page.getByRole('link', { name: 'All divisions', exact: true }).count(), 0)
        assert.equal(await page.getByRole('navigation', { name: 'Player season statistics' }).count(), 0)
        if (state === 'summary') {
          assert.deepEqual(await page.locator('.ma-score-number strong').allTextContents(), ['3', '0'])
          assert.equal(await page.locator('[data-dart]').count(), 0)
          assert.equal(await page.getByLabel('Leg', { exact: true }).count(), 0)
        }
        await capture(`${admin ? 'admin' : 'public'}-${state}`)
        if (release) { release(); release = undefined }
        if (state === 'error') {
          state = 'analysis'
          await page.getByRole('button', { name: 'Try again' }).click()
          await page.getByRole('region', { name: 'Match summary' }).waitFor()
        }
      }
    }
    state = 'analysis'
    await page.goto(`${base}/matches/7`)
    const opponent = page.getByRole('button', { name: fixtureMatch.players[1].preferred_name })
    await opponent.focus(); await page.keyboard.press('Enter')
    assert.equal(await opponent.getAttribute('aria-pressed'), 'true')
    await page.getByLabel('Leg', { exact: true }).selectOption('3')
    assert.equal(await page.locator('[data-dart]').count(), 6)
    await capture('public-opponent-leg3')
    for (state of ['active-list', 'completed-list', 'admin-list']) {
      await page.goto(`${base}/${state === 'admin-list' ? 'admin/' : ''}divisions/test`)
      const link = page.getByRole('link', { name: 'View match: Morgan Ember vs Casey Vale', exact: true })
      await link.waitFor()
      assert.equal(await page.locator('a[href$="/matches/8"], a[href$="/matches/9"]').count(), 0)
      await capture(state)
      await link.click()
      await page.getByRole('region', { name: 'Match summary' }).waitFor()
      assert.equal(new URL(page.url()).pathname, `${state === 'admin-list' ? '/admin' : ''}/matches/7`)
    }
    await page.close()
  }
  assert.deepEqual(errors, [])
  await writeFile(`${screenshots}/hashes.json`, `${JSON.stringify(artifacts, null, 2)}\n`)
  console.log(JSON.stringify({ status: 'passed', widths: [375, 768, 1280], screenshots: artifacts.length, offline: true, errors }, null, 2))
} finally {
  await browser.close()
  await new Promise((resolve, reject) => server.httpServer.close(error => error ? reject(error) : resolve()))
}
