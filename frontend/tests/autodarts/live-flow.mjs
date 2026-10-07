import assert from 'node:assert/strict';
import { randomBytes } from 'node:crypto';
import { execFileSync, spawn } from 'node:child_process';
import { once } from 'node:events';
import { createServer } from 'node:http';
import { mkdir, mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium, expect } from '@playwright/test';
import { preview } from 'vite';
import { source } from '../../../score-scrape/test-source.mjs';

const root = fileURLToPath(new URL('../../../', import.meta.url));
const frontend = path.join(root, 'frontend');
const dsn = process.env.TEST_DATABASE_URL;
assert.ok(dsn, 'Set TEST_DATABASE_URL to a disposable loopback Postgres');
const database = new URL(dsn);
assert.ok(['127.0.0.1', 'localhost', '[::1]'].includes(database.hostname), 'This test only accepts a loopback database');
const schema = `browser_flow_${randomBytes(8).toString('hex')}`;
const runSQL = sql => execFileSync('psql', [dsn, '-v', 'ON_ERROR_STOP=1', '-q', '-c', sql], { stdio: 'pipe' });
const directory = await mkdtemp(path.join(tmpdir(), 'darts-flow-'));
const screenshots = path.join(root, 'docs/pr-screenshots/issue-48');
let backend;
let server;
let browser;
let relay;
let schemaCreated = false;
let queuedBody = null;
let capturedBody;
let acknowledged = 0;

const listen = async instance => {
  instance.listen(0, '127.0.0.1');
  await once(instance, 'listening');
  const address = instance.address();
  assert.ok(address && typeof address !== 'string');
  return address.port;
};
const stopBackend = async () => {
  if (backend && backend.exitCode === null && backend.signalCode === null) {
    const exit = once(backend, 'exit');
    backend.kill('SIGTERM');
    await exit;
  }
};

try {
  runSQL(`CREATE SCHEMA "${schema}"`);
  schemaCreated = true;
  database.searchParams.set('search_path', schema);
  execFileSync('go', ['build', '-o', path.join(directory, 'api'), './cmd/api'], { cwd: path.join(root, 'backend'), stdio: 'pipe' });
  relay = createServer(async (request, response) => {
    response.setHeader('Content-Type', 'application/json');
    if (request.method === 'GET') {
      response.end(JSON.stringify({ messages: queuedBody ? [{ messageId: 'test-delivery', receiptHandle: 'test-receipt', body: queuedBody }] : [] }));
      return;
    }
    let body = '';
    for await (const chunk of request) body += chunk;
    const ack = JSON.parse(body);
    assert.equal(ack.messages[0].messageId, 'test-delivery');
    acknowledged++;
    queuedBody = null;
    response.end(JSON.stringify({ acknowledged: ['test-delivery'], failed: [] }));
  });
  const relayPort = await listen(relay);
  const reservation = createServer();
  const backendPort = await listen(reservation);
  await new Promise(resolve => reservation.close(resolve));
  const backendURL = `http://127.0.0.1:${backendPort}`;
  const sessionSecret = randomBytes(32).toString('hex');
  const startBackend = async now => {
    let stderr = '';
    backend = spawn(path.join(directory, 'api'), [], { env: {
      ...process.env, DATABASE_URL: database.href, HTTP_ADDRESS: `127.0.0.1:${backendPort}`, APP_NOW: now,
      ADMIN_USERNAME: 'admin', ADMIN_PASSWORD: 'offline-only', ADMIN_SESSION_SECRET: sessionSecret,
      SLACK_BOT_TOKEN: '', SLACK_PUBLIC_CHANNEL_ID: '', SLACK_ADMIN_CHANNEL_ID: '',
      RESULTS_ENDPOINT: `http://127.0.0.1:${relayPort}/results`, RESULTS_POLL_INTERVAL: '24h',
    }, stdio: ['ignore', 'ignore', 'pipe'] });
    backend.stderr.on('data', chunk => { stderr += chunk; });
    await expect.poll(async () => {
      if (backend.exitCode !== null) throw new Error(`Test backend exited: ${stderr}`);
      try { return (await fetch(`${backendURL}/healthz`)).status; } catch { return 0; }
    }, { timeout: 15_000 }).toBe(200);
    assert.ok(!stderr.includes('falling back to in-memory'), 'Flow must use durable Postgres');
  };
  await startBackend('2026-06-15T10:00:00Z');
  server = await preview({ root: frontend, preview: { host: '127.0.0.1', port: 0, proxy: { '/api': backendURL } } });
  const address = server.httpServer.address();
  assert.ok(address && typeof address !== 'string');
  const appURL = `http://127.0.0.1:${address.port}`;
  browser = await chromium.launch({ headless: true });

  const capture = await browser.newContext();
  const upstream = source();
  upstream.players[0].name = 'Untrusted source one';
  upstream.players[1].name = 'Untrusted source two';
  await capture.route('**/*', async route => {
    if (route.request().url() === 'https://relay.invalid/results') {
      capturedBody = route.request().postData();
      queuedBody = capturedBody;
      return route.fulfill({ status: 202, headers: { 'access-control-allow-origin': '*' }, body: '{}' });
    }
    if (route.request().url().endsWith('/stats')) return route.fulfill({ json: upstream, headers: { 'access-control-allow-origin': '*' } });
    return route.fulfill({ contentType: 'text/html', body: '<html><body>Isolated source fixture</body></html>' });
  });
  await capture.addCookies([{ url: 'https://play.autodarts.com', name: 'autodarts_score_scrape_settings', value: encodeURIComponent(JSON.stringify({ endpoint: 'https://relay.invalid/results', baseScore: 501, winningLegs: 3 })) }]);
  const sourcePage = await capture.newPage();
  await sourcePage.goto('https://play.autodarts.com');
  await sourcePage.evaluate(() => { window.__scoreScrapeSandbox = false; });
  await sourcePage.evaluate(await readFile(path.join(root, 'score-scrape/scoreScrape.js'), 'utf8'));
  await sourcePage.evaluate(() => new Promise(resolve => {
    const request = new XMLHttpRequest();
    request.open('GET', 'https://api.autodarts.com/as/v0/matches/offline-match/stats');
    request.addEventListener('loadend', resolve);
    request.send();
  }));
  await sourcePage.getByRole('button', { name: 'Submit', exact: true }).click();
  await sourcePage.getByText('Accepted for delivery.', { exact: false }).waitFor();
  assert.ok(capturedBody);
  await capture.close();

  const context = await browser.newContext({ viewport: { width: 1280, height: 960 } });
  await context.route('**/*', route => new URL(route.request().url()).origin === appURL ? route.continue() : route.abort());
  const page = await context.newPage();
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  const api = async (method, endpoint, data) => {
    const response = await context.request.fetch(appURL + endpoint, { method, data });
    assert.ok(response.ok(), `${method} ${endpoint}: ${response.status()} ${await response.text()}`);
    return response.status() === 204 ? null : response.json();
  };
  const first = await api('POST', '/api/players/register', { display_name: 'Morgan Ember', nickname: 'Ember' });
  const second = await api('POST', '/api/players/register', { display_name: 'Casey Vale', nickname: 'Vale' });
  await page.goto(appURL + '/admin');
  await page.getByLabel('Username').fill('admin');
  await page.getByLabel('Password').fill('offline-only');
  await page.getByRole('button', { name: 'Unlock admin tools' }).click();
  await expect(page.getByRole('heading', { name: 'League settings' })).toBeVisible();
  const provisioned = await api('POST', '/api/admin/divisions/provision', { count: 1 });
  const division = provisioned.divisions[0];
  for (const player of [first, second]) await api('PUT', `/api/admin/players/${player.id}/assignment`, { division_id: division.id });
  const season = await api('POST', '/api/admin/season/start');
  const schedule = await api('GET', `/api/admin/divisions/${division.slug}/fixtures`);
  const fixture = schedule.weeks[0].fixtures[0];
  await page.goto(appURL + '/admin/pending-results');
  await page.getByRole('button', { name: 'Fetch new results', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Verify the match' })).toBeVisible();
  await page.locator('#review-season').selectOption(String(season.id));
  await page.locator('#review-division').selectOption(String(division.id));
  await page.locator('#review-player-0').selectOption(String(first.id));
  await page.locator('#review-player-1').selectOption(String(second.id));
  await page.locator('#review-fixture').selectOption(String(fixture.id));
  await page.getByLabel('Review note', { exact: true }).fill('Source clock and intended fixture checked in isolated test');
  await page.getByRole('button', { name: 'Analysis & heatmap' }).click();
  await expect(page.locator('[data-dart]')).toHaveCount(27);
  await mkdir(screenshots, { recursive: true });
  await page.screenshot({ path: path.join(screenshots, 'real-review-1280.png'), fullPage: true });
  await page.getByRole('button', { name: 'Approve result', exact: true }).click();
  await expect(page.getByText('Result approved. Fixture and standings refreshed.')).toBeVisible();
  assert.equal(acknowledged, 1);
  assert.equal((await context.request.get(appURL + `/api/fixtures/${fixture.id}/autodarts`)).status(), 404);
  queuedBody = capturedBody;
  await api('POST', '/api/admin/results/poll');
  assert.equal((await api('GET', '/api/admin/pending-results')).pending_results.length, 0);
  assert.equal(acknowledged, 2);
  await stopBackend();
  await startBackend(new Date(new Date(fixture.scheduled_at).getTime() + 1000).toISOString());
  const match = await api('GET', `/api/fixtures/${fixture.id}/autodarts`);
  assert.equal(match.detail.legs.length, 3);
  assert.equal(match.players.find(player => player.league_player_id === first.id).stats.total_180, 6);
  assert.equal(JSON.stringify(match).includes('Untrusted source'), false);
  assert.equal(JSON.stringify(match).includes('demo-player'), false);
  await page.goto(appURL + `/matches/${fixture.id}`);
  await expect(page.locator('[data-dart]')).toHaveCount(27);
  await page.screenshot({ path: path.join(screenshots, 'real-public-analysis-1280.png'), fullPage: true });
  await api('PUT', `/api/admin/fixtures/${fixture.id}/result`, { player_one_legs: 3, player_two_legs: 1 });
  assert.equal((await api('GET', `/api/fixtures/${fixture.id}/autodarts`)).detail, null);
  await api('DELETE', `/api/admin/fixtures/${fixture.id}/result`);
  assert.equal((await context.request.get(appURL + `/api/fixtures/${fixture.id}/autodarts`)).status(), 404);
  assert.deepEqual(errors, []);
  console.log('PASS real scraper -> HTTP relay -> Postgres -> browser approval -> reveal -> public heatmap -> edit/undo; duplicate import remains reviewed');
} finally {
  if (browser) await browser.close();
  if (server) await server.close();
  await stopBackend();
  if (relay) await new Promise(resolve => relay.close(resolve));
  if (schemaCreated) runSQL(`DROP SCHEMA "${schema}" CASCADE`);
  await rm(directory, { recursive: true, force: true });
}
