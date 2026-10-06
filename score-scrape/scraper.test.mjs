import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { test } from 'node:test';

const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || '../frontend/node_modules/playwright');
const script = await readFile(new URL('./scoreScrape.js', import.meta.url), 'utf8');
const fixture = JSON.parse(await readFile(new URL('../docs/autodarts-fixtures/manual.json', import.meta.url)));
const source = () => {
  const state = {
    id: 'offline-match', createdAt: '2026-06-15T10:00:00+01:00', finishedAt: '2026-06-15T11:00:00+01:00',
    variant: 'X01', targetLegs: 3, targetSets: null, winner: 0,
    settings: { baseScore: 501, outMode: 'Double' },
    players: fixture.players.map(p => ({ id: p.id, name: p.name, userId: null, user: { email: 'private@example.invalid' } })),
    scores: fixture.players.map(p => ({ legs: p.legsWon })), matchStats: fixture.matchStats,
    host: { secret: 'do-not-copy' }, games: []
  };
  for (const dart of fixture.darts) {
    let game = state.games.find(g => g.leg === dart.leg - 1);
    if (!game) state.games.push(game = { set: 0, leg: dart.leg - 1, finishedAt: state.finishedAt, winnerPlayerId: fixture.winnerId, turns: [] });
    let turn = game.turns.find(t => t.turn === dart.visit);
    if (!turn) game.turns.push(turn = { turn: dart.visit, playerId: dart.playerId, score: 501, points: 0, busted: false, throws: [] });
    const prefix = dart.segment[0];
    const number = Number(dart.segment.slice(1));
    turn.throws.push({ throw: dart.dart, segment: { bed: { T: 'Triple', D: 'Double', M: 'Outside', S: 'Single' }[prefix], number }, entry: 'manual_coords', coords: { x: dart.x, y: dart.y } });
    turn.points += ({ T: 3, D: 2, M: 0, S: 1 })[prefix] * number;
  }
  for (const game of state.games) {
    const remaining = new Map(state.players.map(p => [p.id, 501]));
    for (const turn of game.turns) { turn.score = remaining.get(turn.playerId) - turn.points; remaining.set(turn.playerId, turn.score); }
  }
  return state;
};
const expected = state => ({
  schema_version: 'autodarts.import.v1', source: 'autodarts', external_match_id: state.id,
  playedAt: state.createdAt, settings: { base_score: 501, legs_to_win: 3, out: 'double' }, completed: true,
  players: state.players.map((p, i) => {
    const s = state.matchStats?.find(s => s.playerId === p.id);
    return { match_player_id: p.id, account_id: p.userId, display_name: p.name, legs_won: state.scores[i].legs,
      stats: s ? { match_average: s.average ?? null, points_scored: null, darts_thrown: s.dartsThrown ?? null, checkout_hits: s.checkoutsHit ?? null, checkout_attempts: s.checkouts ?? null } : null };
  }),
  detail: { coverage: 'partial', legs: state.games.map(g => ({ number: g.leg + 1, completed: true, winner_id: g.winnerPlayerId,
    visits: g.turns.map(t => ({ number: t.turn + 1, player_id: t.playerId, start_remaining: t.score + t.points, end_remaining: t.score, bust: false,
      throws: t.throws.map(d => ({ number: d.throw + 1, segment: { bed: { Triple: 'triple', Double: 'double', Outside: 'miss', Single: 'single' }[d.segment.bed], number: d.segment.bed === 'Outside' ? 0 : d.segment.number }, entry_type: 'manual',
        position: d.coords ? { x: d.coords.x, y: d.coords.y, units: 'board-radius', origin: 'bull', axis_orientation: 'x-right-y-up', provenance: 'manual' } : null })) })) })) }
});

test('offline intercepted XHR: exact detail, privacy, retries and filtering', async () => {
  const browser = await chromium.launch({ headless: true });
  try {
    const context = await browser.newContext();
    const page = await context.newPage();
    let response = source();
    let fail = false;
    const posts = [];
    await context.route('**/*', async route => {
      const request = route.request();
      if (request.url() === 'https://relay.invalid/results') {
        posts.push(JSON.parse(request.postData()));
        return route.fulfill({ status: fail ? 503 : 202, headers: { 'access-control-allow-origin': '*' }, body: '{}' });
      }
      if (request.url().includes('/stats')) return route.fulfill({ contentType: 'application/json', headers: { 'access-control-allow-origin': '*' }, body: JSON.stringify(response) });
      return route.fulfill({ contentType: 'text/html', body: '<html><body>Offline Autodarts</body></html>' });
    });
    const install = async (live = true, old = false) => {
      await page.goto('https://play.autodarts.com/');
      await page.evaluate(({ live, old }) => {
        document.cookie = `autodarts_score_scrape_settings=${encodeURIComponent(JSON.stringify({ endpoint: 'https://relay.invalid/results', baseScore: 501, winningLegs: old ? 2 : 3 }))}; path=/`;
        if (live) window.__scoreScrapeSandbox = false;
      }, { live, old });
      await page.evaluate(script);
    };
    const emit = async (url = `https://api.autodarts.com/as/v0/matches/${response.id}/stats`, json = false) => page.evaluate(async ({ url, json }) => {
      await new Promise(resolve => { const xhr = new XMLHttpRequest(); xhr.open('GET', url); if (json) xhr.responseType = 'json'; xhr.addEventListener('loadend', resolve); xhr.send(); });
    }, { url, json });
    const button = name => page.getByRole('button', { name, exact: true });
    const confirm = async () => { await button('Submit').click(); };
    const close = async () => { await button('Close').click(); };
    await install(true, true);
    assert.equal(await page.locator('[name=winningLegs]').inputValue(), '3');
    await button('Save').click();
    assert.equal(await page.locator('#autodarts-wrapper-installed-indicator').count(), 1);
    await emit(); await emit();
    assert.equal(await button('Submit').count(), 1);
    await button('Cancel').click();
    await emit();
    await button('Submit').evaluate(el => { el.click(); el.click(); });
    await page.getByText('Accepted for delivery.', { exact: false }).waitFor();
    assert.deepEqual(posts, [expected(response)]);
    await close(); await emit();
    assert.equal(await button('Submit').count(), 0);
    response.createdAt = '2026-06-16T10:00:00+01:00';
    fail = true; await emit(); await confirm();
    await button('Retry').waitFor();
    assert.equal(posts.length, 2);
    fail = false; await button('Retry').click(); await close();
    assert.equal(posts.length, 3); assert.deepEqual(posts[1], posts[2]);
    response.createdAt = '2026-06-17T10:00:00+01:00';
    fail = true; await emit(); await confirm(); await close();
    fail = false; await emit(); await confirm(); await close();
    assert.equal(posts.length, 5);
    response = source(); response.id = 'reversed'; response.players.reverse(); response.scores.reverse(); response.winner = 1; response.matchStats.reverse();
    delete response.matchStats[0].average;
    response.games[0].turns[0].throws[0].coords = null;
    await emit(undefined, true); await confirm(); await close();
    assert.deepEqual(posts.at(-1), expected(response));
    response = source(); response.id = 'changed-while-pending';
    await emit();
    response.createdAt = null;
    await emit(); await confirm(); await close(); await confirm(); await close();
    assert.notEqual(posts.at(-2).playedAt, posts.at(-1).playedAt);
    response = source(); response.id = 'unknown-and-auto';
    const darts = response.games[0].turns[0].throws;
    darts[0].entry = 'unrecognized'; darts[1].entry = 'auto';
    darts[2].coords = { x: null, y: 1 };
    await emit(); await confirm(); await close();
    const resultDarts = posts.at(-1).detail.legs[0].visits[0].throws;
    assert.equal(resultDarts[0].entry_type, 'unknown');
    assert.equal(resultDarts[0].position.provenance, 'unknown');
    assert.equal(resultDarts[1].entry_type, 'automatic');
    assert.equal(resultDarts[1].position.units, null);
    assert.equal(resultDarts[2].position, null);
    response = source(); response.id = 'summary'; delete response.games; delete response.matchStats;
    await emit('https://api.autodarts.io/as/v0/matches/summary/stats'); await confirm(); await close();
    assert.equal(posts.at(-1).detail, null); assert.equal(posts.at(-1).players[0].stats, null);
    for (const change of [s => { s.targetLegs = 2; }, s => { s.finishedAt = null; }, s => { s.variant = 'Cricket'; }, s => { s.scores[1].legs = 3; }, s => { s.players[1].id = s.players[0].id; }, s => { s.settings.outMode = 'Straight'; }]) {
      response = source(); change(response); await emit(); assert.equal(await button('Submit').count(), 0);
    }
    response = source();
    for (const url of ['https://evil.invalid/as/v0/matches/offline-match/stats', 'https://api.autodarts.com/as/v0/matches/wrong/stats', 'https://api.autodarts.com/as/v0/matches/offline-match/stats/extra']) {
      await emit(url); assert.equal(await button('Submit').count(), 0);
    }
    const before = posts.length;
    await install(false); await emit(); await confirm(); await close(); assert.equal(posts.length, before);
    await install(); response.synthetic = true; await emit(); await confirm(); await close(); assert.equal(posts.length, before);
    await context.close();
  } finally { await browser.close(); }
});
