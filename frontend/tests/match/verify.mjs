import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { chromium } from '@playwright/test'
import { build, preview } from 'vite'

const root = fileURLToPath(new URL('../../', import.meta.url))
const screenshots = fileURLToPath(new URL('./screenshots/', import.meta.url))
const outDir = 'node_modules/.cache/match-harness'
await build({ root, build: { outDir, emptyOutDir: true, rollupOptions: {
  input: fileURLToPath(new URL('./index.html', import.meta.url)),
} } })
const server = await preview({ root, build: { outDir }, preview: { host: '127.0.0.1', port: 0 } })
const address = server.httpServer.address()
assert(address && typeof address === 'object')
const base = `http://127.0.0.1:${address.port}/tests/match/index.html`
const browser = await chromium.launch()
const errors = []
const artifacts = []
try {
  await mkdir(screenshots, { recursive: true })
  for (const width of [375, 768, 1280]) {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    page.on('pageerror', error => errors.push(error.message))
    page.on('console', message => { if (message.type() === 'error') errors.push(message.text()) })
    page.on('response', response => { if (response.status() >= 400) errors.push(`${response.status()} ${response.url()}`) })
    await page.route('**/*', route => new URL(route.request().url()).hostname === '127.0.0.1' ? route.continue() : route.abort())
    for (const state of ['analysis', 'unknown', 'summary', 'partial', 'long', 'loading', 'error']) {
      await page.goto(`${base}?state=${state}`)
      await page.getByRole('heading', { name: 'Match analysis.' }).waitFor()
      await page.evaluate(() => document.fonts.ready)
      assert(await page.evaluate(() => document.fonts.check('16px Barlow') && document.fonts.check('700 24px Rajdhani')), 'Local fonts loaded')
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `${state}/${width} page overflow`)
      if (state === 'analysis') {
        assert.equal(await page.locator('[data-dart]').count(), 27)
        assert.equal(await page.locator('[data-dart]').nth(1).getAttribute('cx'), '249.74626310007167')
      }
      if (state === 'unknown') {
        assert.equal(await page.locator('[data-dart], [data-heat]').count(), 0)
        assert(await page.getByText('Coordinates unavailable').isVisible())
      }
      const file = `${state}-${width}.png`
      await page.screenshot({ path: `${screenshots}/${file}`, fullPage: true })
      artifacts.push({ file, sha256: createHash('sha256').update(await readFile(`${screenshots}/${file}`)).digest('hex') })
    }
    await page.goto(base)
    const opponent = page.getByRole('button', { name: 'Casey Vale' })
    await opponent.focus()
    await page.keyboard.press('Enter')
    assert.equal(await opponent.getAttribute('aria-pressed'), 'true')
    assert.equal(await page.locator('[data-dart]').count(), 21)
    const leg = page.getByLabel('Leg', { exact: true })
    await leg.focus()
    await page.keyboard.press('End')
    await page.keyboard.press('Enter')
    assert.equal(await leg.inputValue(), '3')
    assert.equal(await page.locator('[data-dart]').count(), 6)
    const table = page.getByRole('region', { name: 'Recorded visits and checkout routes' })
    await table.focus()
    assert(await table.evaluate(el => el === document.activeElement))
    if (width === 375) {
      await page.keyboard.press('ArrowRight')
      await page.waitForFunction(() => (document.activeElement?.scrollLeft ?? 0) > 0)
    }
    const file = `opponent-leg3-${width}.png`
    await page.screenshot({ path: `${screenshots}/${file}`, fullPage: true })
    artifacts.push({ file, sha256: createHash('sha256').update(await readFile(`${screenshots}/${file}`)).digest('hex') })
    const coordinates = page.getByText('Recorded coordinates & provenance (text)', { exact: true })
    await coordinates.focus()
    await page.keyboard.press('Enter')
    assert.equal(await page.locator('details[open] li').count(), 6)
    assert(await page.locator('details[open]').getByText(/units=board-radius, origin=bull, axes=x-right-y-up/).first().isVisible())
    await page.goto(`${base}?state=error`)
    await page.getByRole('button', { name: 'Try again' }).click()
    await page.getByRole('region', { name: 'Match summary' }).waitFor()
    await page.close()
  }
  assert.deepEqual(errors, [])
  await writeFile(`${screenshots}/hashes.json`, `${JSON.stringify(artifacts, null, 2)}\n`)
  console.log(JSON.stringify({ status: 'passed', widths: [375, 768, 1280], screenshots: artifacts.length, browserErrors: errors }, null, 2))
} finally {
  await browser.close()
  await new Promise((resolve, reject) => server.httpServer.close(error => error ? reject(error) : resolve()))
}
