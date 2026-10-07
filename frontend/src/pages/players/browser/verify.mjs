import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { chromium } from '@playwright/test'
import { preview } from 'vite'

const root = fileURLToPath(new URL('../../../../', import.meta.url))
const screenshots = fileURLToPath(new URL('./screenshots/', import.meta.url))
const fonts = new URL('../../../../../docs/mockups/autodarts/fonts/', import.meta.url)
const fixture = JSON.parse(await readFile(new URL('../../../lib/playerStatistics.fixture.json', import.meta.url), 'utf8'))
const manual = { ...fixture, first_nine_match_average_mean: null, checkout_hits: null, checkout_attempts: null, checkout_percentage: null, total_180: null, highest_finish: null,
  coverage: { ...fixture.coverage, matches_with_detail: 0, first_nine_match_average_mean: 0, checkout: 0, total_180: 0, highest_finish: 0,
    matches_with_recorded_throws: 0, matches_with_positions: 0, recorded_throws: 0, known_positions: 0, plottable_positions: 0, position_fraction: null },
  history: fixture.history.map(h => ({ ...h, detail_coverage: 'none' })), throws: [] }
const empty = { ...manual, played: 0, won: 0, lost: 0, legs_for: 0, legs_against: 0, points: 0, match_average_mean: null,
  coverage: { ...manual.coverage, eligible_matches: 0, match_average_mean: 0 }, history: [] }
const server = await preview({ root, preview: { host: '127.0.0.1', port: 0 } })
const address = server.httpServer.address()
assert(address && typeof address === 'object')
const base = `http://127.0.0.1:${address.port}`
const browser = await chromium.launch()
const artifacts = [], errors = []
try {
  await mkdir(screenshots, { recursive: true })
  for (const width of [375, 768, 1280]) {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    let state = 'partial', release
    page.on('pageerror', error => errors.push(error.message))
    await page.route('**/*', async route => {
      const url = new URL(route.request().url())
      if (url.hostname === 'fonts.googleapis.com') {
        const css = await readFile(new URL('fonts.css', fonts), 'utf8')
        return route.fulfill({ contentType: 'text/css', body: css.replaceAll('url(', `url(${base}/qa-fonts/`) })
      }
      if (url.pathname.startsWith('/qa-fonts/')) return route.fulfill({ contentType: 'font/ttf', body: await readFile(new URL(url.pathname.split('/').at(-1), fonts)) })
      if (url.origin !== base) { errors.push(`External request: ${url.origin}`); return route.abort() }
      if (!url.pathname.startsWith('/api/')) return route.continue()
      let body, status = 200
      if (url.pathname.endsWith('/statistics')) {
        if (state === 'loading') await new Promise(resolve => { release = resolve })
        status = state === 'missing' ? 404 : state === 'unauthorized' ? 401 : state === 'error' ? 503 : 200
        body = status !== 200 ? { error: { message: 'private-source-secret' } } : state === 'manual' ? manual : state === 'empty' ? empty : state === 'zero-attempts' ? { ...fixture, checkout_attempts: 0, checkout_percentage: null } : fixture
      } else if (url.pathname === '/api/season') body = { id: 2, instance_name: 'Offline player QA', name: 'Season two', status: 'started', registration_open: false }
      else if (url.pathname === '/api/version') body = { version: 'offline-qa' }
      else if (url.pathname.endsWith('/divisions')) body = { divisions: [{ id: 1, name: 'Test division', slug: 'test', position: 1 }] }
      else if (url.pathname.endsWith('/players')) body = { players: [] }
      else if (url.pathname.endsWith('/standings')) body = { standings: [{ player_id: 7, player: 'Arrow', display_name: 'Alice', played: 2, won: 1, lost: 1, legs_for: 4, legs_against: 4, leg_difference: 0, points: 2 }] }
      else { errors.push(`Unexpected API: ${url.pathname}`); status = 404; body = {} }
      return route.fulfill({ status, contentType: 'application/json', headers: { 'Cache-Control': 'no-store' }, body: JSON.stringify(body) })
    })
    const capture = async name => {
      await page.evaluate(() => document.fonts.ready)
      assert(await page.evaluate(() => document.fonts.check('16px Barlow') && document.fonts.check('700 24px Rajdhani')))
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `${name}/${width}: overflow`)
      assert.equal(await page.getByText('private-source-secret').count(), 0)
      const file = `${name}-${width}.png`
      await page.screenshot({ path: `${screenshots}/${file}`, fullPage: true })
      artifacts.push({ file, sha256: createHash('sha256').update(await readFile(`${screenshots}/${file}`)).digest('hex') })
    }
    for (const admin of [false, true]) {
      for (state of admin ? ['partial', 'unauthorized'] : ['partial', 'manual', 'zero-attempts', 'empty', 'missing', 'error', 'loading']) {
        await page.goto(`${base}${admin ? '/admin' : ''}/seasons/2/players/7`)
        if (state === 'loading') await page.getByText('Loading player statistics...').waitFor()
        else if (state === 'missing') await page.getByText('Player not found in this season.').waitFor()
        else if (state === 'unauthorized') await page.getByRole('link', { name: 'Admin login' }).waitFor()
        else if (state === 'error') await page.getByRole('button', { name: 'Try again' }).waitFor()
        else await page.getByRole('heading', { name: 'Arrow', exact: true }).waitFor()
        if (state === 'partial') {
          assert.equal(await page.locator('[data-dart]').count(), 1)
          assert.equal(await page.getByRole('link', { name: 'Match 12', exact: true }).getAttribute('href'), `${admin ? '/admin' : ''}/matches/12`)
          assert.equal(await page.getByRole('link', { name: 'The Comet', exact: true }).getAttribute('href'), `${admin ? '/admin' : ''}/seasons/2/players/9`)
        }
        await capture(`${admin ? 'admin' : 'public'}-${state}`)
        if (release) { release(); release = undefined }
        if (state === 'error') {
          state = 'partial'
          await page.getByRole('button', { name: 'Try again' }).click()
          await page.getByRole('heading', { name: 'Arrow', exact: true }).waitFor()
        }
      }
    }
    state = 'partial'
    await page.goto(`${base}/seasons/2/players/7`)
    await page.getByRole('heading', { name: 'Arrow', exact: true }).waitFor()
    for (const [entry, points] of [['Manual', 1], ['Automatic', 0], ['Unknown', 0], ['All', 1]]) {
      const button = page.getByRole('button', { name: entry, exact: true })
      await button.focus(); await page.keyboard.press('Enter')
      assert.equal(await button.getAttribute('aria-pressed'), 'true')
      assert.equal(await page.locator('[data-dart]').count(), points)
      await capture(`public-filter-${entry.toLowerCase()}`)
    }
    await page.goto(`${base}/divisions/test/standings`)
    const link = page.getByRole('link', { name: 'Arrow', exact: true })
    await link.waitFor()
    assert.equal(await link.getAttribute('href'), '/seasons/2/players/7')
    await link.click()
    await page.getByRole('heading', { name: 'Arrow', exact: true }).waitFor()
    await page.close()
  }
  assert.deepEqual(errors, [])
  await writeFile(`${screenshots}/hashes.json`, `${JSON.stringify(artifacts, null, 2)}\n`)
  console.log(JSON.stringify({ status: 'passed', widths: [375, 768, 1280], screenshots: artifacts.length, errors }, null, 2))
} finally {
  await browser.close()
  await new Promise((resolve, reject) => server.httpServer.close(error => error ? reject(error) : resolve()))
}
