import assert from 'node:assert/strict';
import { mkdir, readFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';

const require = createRequire(new URL('../../../frontend/package.json', import.meta.url));
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fixture = JSON.parse(await readFile(new URL('../../autodarts-fixtures/randomized.json', import.meta.url), 'utf8'));
const browser = await chromium.launch({ headless: true });
const directory = fileURLToPath(new URL('./screenshots/', import.meta.url));
await mkdir(directory, { recursive: true });
const errors = [];
const base = process.env.MOCKUP_BASE_URL || 'http://127.0.0.1:8765';
try {
  for (const width of [1280, 768, 375]) {
    const page = await browser.newPage({ viewport: { width, height: 960 } });
    page.on('pageerror', error => errors.push(error.message));
    await page.route('**/*', route => new URL(route.request().url()).origin === new URL(base).origin ? route.continue() : route.abort());
    await page.goto(`${base}/docs/mockups/autodarts/`);
    await page.waitForFunction(() => document.body.dataset.captureLoaded === 'true');
    assert.equal(await page.locator('#match-title').innerText(), 'MORGAN EMBER');
    await page.evaluate(() => document.fonts.ready);
    assert.equal(await page.evaluate(() => document.fonts.check('700 24px Rajdhani')), true);
    assert.equal(await page.evaluate(() => document.fonts.check('400 16px Barlow')), true);
    for (const view of ['review', 'analysis', 'unmatched']) {
      if (view === 'analysis') await page.locator('[data-view="analysis"]').click();
      if (view === 'unmatched') await page.locator('#mapping-demo').click();
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, `Overflow at ${width}/${view}`);
      await page.screenshot({ path: `${directory}/${view}-${width}.png`, fullPage: true });
    }
    assert.equal(await page.locator('#confirm').isDisabled(), true);
    await page.locator('#player-two').selectOption('morgan');
    assert.equal(await page.locator('#confirm').isDisabled(), true, 'Same player cannot map to both sides');
    await page.locator('#player-two').selectOption('casey');
    assert.equal(await page.locator('#confirm').isEnabled(), true);
    await page.locator('#confirm').click();
    assert.match(await page.locator('#action-status').innerText(), /Nothing was saved/);
    await page.screenshot({ path: `${directory}/confirmed-${width}.png`, fullPage: true });
    await page.locator('#reject').click();
    assert.match(await page.locator('#action-status').innerText(), /rejected/);
    await page.screenshot({ path: `${directory}/rejected-${width}.png`, fullPage: true });
    await page.locator('#open-analysis').click();
    assert.equal(await page.locator('#analysis-view').isVisible(), true);
    assert.match(await page.locator('#board-description').textContent(), /21 at T20, 3 at T19, 3 at D12/);
    for (const player of ['winner', 'opponent']) {
      await page.locator(`[data-player="${player}"]`).click();
      for (const leg of ['all', '1', '2', '3']) {
        await page.locator('#leg-filter').selectOption(leg);
        const count = player === 'winner' ? (leg === 'all' ? 27 : 9) : (leg === 'all' ? 21 : leg === '2' ? 9 : 6);
        assert.equal((await page.locator('#dart-count').innerText()).trim(), `${count} darts`);
        assert.match(await page.locator('#board-description').textContent(), new RegExp(`^${count} synthetic darts`));
        const darts = fixture.darts.filter(d => d.playerId === (player === 'winner' ? 'demo-player-1' : 'demo-player-2') && (leg === 'all' || d.leg === Number(leg)));
        const segments = Object.fromEntries([...new Set(darts.map(d => d.segment))].map(segment => [segment, darts.filter(d => d.segment === segment).length]));
        const displayed = await page.locator('#distribution .segment-row').evaluateAll(rows => Object.fromEntries(rows.map(row => [row.querySelector('span').textContent, Number(row.querySelector('b').textContent)])));
        assert.deepEqual(displayed, segments, 'Textual chart equivalent matches actual dart segments');
        const rendered = await page.locator('#full-board [data-dart]').evaluateAll(nodes => nodes.map(node => ({ x: Number(node.getAttribute('cx')), y: Number(node.getAttribute('cy')) })));
        assert.deepEqual(rendered, darts.map(d => ({ x: 250 + d.x * 191, y: 250 - d.y * 191 })));
      }
    }
    await page.screenshot({ path: `${directory}/opponent-leg3-${width}.png`, fullPage: true });
    await page.keyboard.press('Tab');
    await page.locator('.table-wrap').focus();
    assert.equal(await page.locator('.table-wrap').evaluate(el => getComputedStyle(el).outlineStyle), 'solid');
    if (width === 375) {
      await page.keyboard.press('ArrowRight');
      await page.waitForFunction(() => document.querySelector('.table-wrap').scrollLeft > 0);
    }
    await page.locator('[data-view="review"]').click();
    await page.locator('#division').selectOption('');
    assert.equal(await page.locator('#confirm').isDisabled(), true);
    assert.equal(await page.locator('#player-one').isDisabled(), true);
    await page.keyboard.press('Tab');
    await page.locator('#mapping-demo').focus();
    assert.equal(await page.locator('#mapping-demo').evaluate(el => getComputedStyle(el).outlineStyle), 'solid');
    await page.goto(`${base}/docs/mockups/autodarts/?view=analysis&coordinates=missing`);
    await page.waitForFunction(() => document.body.dataset.captureLoaded === 'true');
    assert.equal(await page.locator('#full-board [data-dart]').count(), 0);
    assert.match(await page.locator('#board-description').textContent(), /no positions are invented/);
    assert.match(await page.locator('#distribution').innerText(), /T20/);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: `${directory}/no-coordinate-${width}.png`, fullPage: true });
    for (const player of ['winner', 'opponent']) {
      await page.locator(`[data-player="${player}"]`).click();
      for (const leg of ['all', '1', '2', '3']) {
        await page.locator('#leg-filter').selectOption(leg);
        assert.equal(await page.locator('#full-board [data-dart]').count(), 0);
        assert.match(await page.locator('#board-title').textContent(), /Coordinates unavailable/);
      }
    }
    console.log(`PASS ${width}px: layouts, player/leg filters, mapping guards, preview actions, keyboard focus`);
    await page.close();
  }
  assert.deepEqual(errors, []);
} finally {
  await browser.close();
}
