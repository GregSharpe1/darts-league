import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';

const examples = JSON.parse(readFileSync(new URL('./examples-v1.json', import.meta.url), 'utf8'));
let checks = 0;
function check(name, run) {
  run();
  checks++;
  console.log(`ok ${checks} - ${name}`);
}
function keys(value, required, optional = []) {
  assert(value && typeof value === 'object' && !Array.isArray(value));
  assert(required.every(key => Object.hasOwn(value, key)), 'missing field');
  assert(Object.keys(value).every(key => [...required, ...optional].includes(key)), 'unknown field');
}
const integer = (n, min, max) => assert(Number.isSafeInteger(n) && n >= min && n <= max);
const id = value => assert(typeof value === 'string' && /^[A-Za-z0-9_-]{1,128}$/.test(value));
const label = value => assert(typeof value === 'string' && [...value].length >= 1 && [...value].length <= 80 && !/[\u0000-\u001f\u007f-\u009f]/.test(value));
const decimal = (n, min, max) => assert(typeof n === 'number' && Number.isFinite(n) && n >= min && n <= max);
const timestamp = value => assert(value === null || (typeof value === 'string' && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/.test(value) && Number.isFinite(Date.parse(value))));
const score = scores => assert(scores.length === 2 && scores.every(Number.isInteger) && ((scores[0] === 3 && scores[1] >= 0 && scores[1] < 3) || (scores[1] === 3 && scores[0] >= 0 && scores[0] < 3)));
function canonical(value) {
  if (Array.isArray(value)) return `[${value.map(canonical).join(',')}]`;
  if (value !== null && typeof value === 'object') return `{${Object.keys(value).sort().map(key => `${JSON.stringify(key)}:${canonical(value[key])}`).join(',')}}`;
  return JSON.stringify(value);
}
const digest = value => createHash('sha256').update(canonical(value), 'utf8').digest('hex');
function stats(value) {
  if (value === null) return;
  keys(value, ['match_average', 'points_scored', 'darts_thrown', 'checkout_hits', 'checkout_attempts']);
  if (value.match_average !== null) decimal(value.match_average, 0, 180);
  for (const field of ['points_scored', 'darts_thrown', 'checkout_hits', 'checkout_attempts']) {
    if (value[field] !== null) integer(value[field], 0, field === 'points_scored' ? 5010 : field === 'checkout_hits' ? 3 : 3000);
  }
  if (value.checkout_hits !== null && value.checkout_attempts !== null) assert(value.checkout_hits <= value.checkout_attempts);
  if (value.checkout_attempts !== null && value.darts_thrown !== null) assert(value.checkout_attempts <= value.darts_thrown);
  if (value.darts_thrown === 0) assert((value.points_scored === null || value.points_scored === 0) && value.match_average === null);
}
function throwPoints(dart, index) {
  keys(dart, ['number', 'segment', 'entry_type', 'position']);
  assert.equal(dart.number, index + 1);
  assert(['automatic', 'manual', 'unknown'].includes(dart.entry_type));
  keys(dart.segment, ['bed', 'number']);
  const { bed, number } = dart.segment;
  const multipliers = { single: 1, double: 2, triple: 3 };
  let points;
  if (Object.hasOwn(multipliers, bed)) {
    integer(number, 1, 20);
    points = number * multipliers[bed];
  } else {
    assert(['miss', 'outer_bull', 'inner_bull'].includes(bed));
    assert.equal(number, bed === 'miss' ? 0 : 25);
    points = bed === 'inner_bull' ? 50 : number;
  }
  if (dart.position !== null) {
    keys(dart.position, ['x', 'y', 'units', 'origin', 'axis_orientation', 'provenance']);
    assert(Number.isFinite(dart.position.x) && Number.isFinite(dart.position.y));
    for (const field of ['units', 'origin', 'axis_orientation']) if (dart.position[field] !== null) label(dart.position[field]);
    assert(['automatic', 'manual', 'unknown'].includes(dart.position.provenance));
  }
  return points;
}
function detail(value, players) {
  if (value === null) return;
  keys(value, ['coverage', 'legs']);
  assert(['partial', 'complete'].includes(value.coverage));
  assert(Array.isArray(value.legs) && value.legs.length <= 5);
  const ids = players.map(player => player.match_player_id);
  const numbers = new Set();
  let throws = 0;
  for (const leg of value.legs) {
    keys(leg, ['number', 'completed', 'winner_id', 'visits']);
    integer(leg.number, 1, 5);
    assert(!numbers.has(leg.number));
    numbers.add(leg.number);
    assert(typeof leg.completed === 'boolean');
    assert(leg.completed ? ids.includes(leg.winner_id) : leg.winner_id === null);
    assert(Array.isArray(leg.visits) && leg.visits.length <= 200);
    let previous = 0;
    for (const visit of leg.visits) {
      keys(visit, ['number', 'player_id', 'start_remaining', 'end_remaining', 'bust', 'throws']);
      integer(visit.number, previous + 1, 200);
      previous = visit.number;
      assert(ids.includes(visit.player_id));
      integer(visit.start_remaining, 2, 501);
      integer(visit.end_remaining, 0, 501);
      assert(typeof visit.bust === 'boolean');
      assert(Array.isArray(visit.throws) && visit.throws.length >= 1 && visit.throws.length <= 3);
      throws += visit.throws.length;
      let remaining = visit.start_remaining;
      let busted = false;
      visit.throws.forEach((dart, index) => {
        assert(remaining > 1 && !busted, 'throw after checkout/bust');
        remaining -= throwPoints(dart, index);
        const double = ['double', 'inner_bull'].includes(dart.segment.bed);
        busted = remaining < 0 || remaining === 1 || (remaining === 0 && !double);
      });
      assert.equal(visit.bust, busted);
      assert.equal(visit.end_remaining, busted ? visit.start_remaining : remaining);
      if (remaining === 0 && !busted) assert(leg.completed && leg.winner_id === visit.player_id);
    }
  }
  assert(throws <= 3000);
  if (value.coverage === 'complete') {
    assert.equal(value.legs.length, players.reduce((sum, player) => sum + player.legs_won, 0));
    for (const player of players) assert.equal(value.legs.filter(leg => leg.completed && leg.winner_id === player.match_player_id).length, player.legs_won);
  }
}
function validate(payload) {
  assert(Buffer.byteLength(JSON.stringify(payload), 'utf8') <= 128 * 1024);
  if (!Object.hasOwn(payload, 'schema_version')) {
    keys(payload, ['matchId', 'player1', 'player2'], ['playedAt']);
    timestamp(payload.playedAt ?? null);
    id(payload.matchId);
    for (const player of [payload.player1, payload.player2]) {
      keys(player, ['name', 'legsWon'], ['matchAverage']);
      label(player.name);
      if (player.matchAverage !== null && player.matchAverage !== undefined) decimal(player.matchAverage, 0, 180);
    }
    score([payload.player1.legsWon, payload.player2.legsWon]);
    return;
  }
  keys(payload, ['schema_version', 'source', 'external_match_id', 'playedAt', 'settings', 'completed', 'players', 'detail']);
  timestamp(payload.playedAt);
  assert.equal(payload.schema_version, 'autodarts.import.v1');
  assert.equal(payload.source, 'autodarts');
  id(payload.external_match_id);
  assert.deepEqual(payload.settings, { base_score: 501, legs_to_win: 3, out: 'double' });
  assert.equal(payload.completed, true);
  assert(Array.isArray(payload.players) && payload.players.length === 2);
  for (const player of payload.players) {
    keys(player, ['match_player_id', 'account_id', 'display_name', 'legs_won', 'stats']);
    id(player.match_player_id);
    if (player.account_id !== null) id(player.account_id);
    label(player.display_name);
    stats(player.stats);
  }
  assert.notEqual(payload.players[0].match_player_id, payload.players[1].match_player_id);
  score(payload.players.map(player => player.legs_won));
  detail(payload.detail, payload.players);
}

check('versioned detailed, missing-detail and legacy filtered payloads validate', () => {
  for (const payload of [examples.producer, examples.missing_detail, examples.legacy]) validate(payload);
});
check('canonical hash vector, key order independence and legacy digest', () => {
  assert.equal(digest({}), '44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a');
  assert.equal(digest({ b: 2, a: 1 }), digest({ a: 1, b: 2 }));
  assert.equal(digest(examples.legacy), '6b3471c50140c311a64d35f4b4a88cf6751e625513222bfd8aab33f714ef447d');
});
const invalid = [
  ['unknown version', p => { p.schema_version = 'autodarts.import.v2'; }],
  ['profile leakage in stored projection', p => { p.players[0].email = 'fictional@example.invalid'; }],
  ['timestamp without source offset', p => { p.playedAt = '2026-06-15T10:00:00'; }],
  ['bad format', p => { p.settings.legs_to_win = 2; }],
  ['unfinished match', p => { p.completed = false; }],
  ['invalid score', p => { p.players[0].legs_won = 2; }],
  ['fractional score', p => { p.players[1].legs_won = 0.5; }],
  ['duplicate players', p => { p.players[1].match_player_id = 'seat-a'; }],
  ['nonfinite stats', p => { p.players[0].stats.match_average = Infinity; }],
  ['negative stats', p => { p.players[0].stats.darts_thrown = -1; }],
  ['zero darts with average', p => { p.players[0].stats.darts_thrown = 0; }],
  ['nonfinite coordinate', p => { p.detail.legs[0].visits[0].throws[0].position.y = Infinity; }],
  ['synthetic production provenance', p => { p.detail.legs[0].visits[0].throws[0].position.provenance = 'synthetic'; }],
  ['foreign player reference', p => { p.detail.legs[0].visits[0].player_id = 'not-in-match'; }],
  ['score detail disagreement', p => { p.detail.legs[0].visits[0].end_remaining = 10; }],
  ['false complete coverage', p => { p.detail.coverage = 'complete'; }],
  ['too many legs', p => { p.detail.legs = Array(6).fill(p.detail.legs[0]); }],
  ['too many throws', p => { const v = p.detail.legs[0].visits[0]; v.throws = Array(4).fill(v.throws[0]); }],
  ['oversized body', p => { p.players[0].display_name = 'x'.repeat(128 * 1024); }],
];
for (const [name, mutate] of invalid) check(`reject ${name}`, () => {
  const payload = structuredClone(examples.producer);
  mutate(payload);
  assert.throws(() => validate(payload));
});
check('null legacy averages remain unknown and guests have no stable identity', () => {
  assert.equal(examples.legacy.player1.matchAverage ?? null, null);
  assert.equal(examples.legacy.player2.matchAverage ?? null, null);
  assert.equal(examples.producer.players[1].account_id, null);
  assert.equal(examples.missing_detail.detail, null);
  assert.equal(examples.legacy.playedAt ?? null, null);
});
check('optional raw coordinates preserve unknown units and source precision', () => {
  const payload = structuredClone(examples.producer);
  payload.detail.legs[0].visits[0].throws[0].position = examples.unknown_geometry_position;
  validate(payload);
  const stored = JSON.parse(JSON.stringify(payload));
  assert.deepEqual(stored.detail.legs[0].visits[0].throws[0].position, examples.unknown_geometry_position);
});
check('reversed mapping moves every player association', () => {
  const { request, fixture, response } = examples.reversed_confirmation;
  const mapping = new Map(examples.producer.players.map((player, index) => [player.match_player_id, [request.player_one_id, request.player_two_id][index]]));
  assert.equal(new Set(mapping.values()).size, 2);
  const ordered = [fixture.player_one_id, fixture.player_two_id].map(id => examples.producer.players.find(p => mapping.get(p.match_player_id) === id));
  assert.deepEqual(ordered.map(p => p.legs_won), [response.player_one_legs, response.player_two_legs]);
  assert.deepEqual(ordered.map(p => p.stats?.match_average ?? null), [response.player_one_average, response.player_two_average]);
  const normalized = structuredClone(examples.producer.detail);
  for (const leg of normalized.legs) {
    leg.winner_id = mapping.get(leg.winner_id) ?? null;
    for (const visit of leg.visits) visit.player_id = mapping.get(visit.player_id);
  }
  assert.deepEqual(normalized, examples.public_detail.detail);
});
check('changed imports have changed digests; duplicate lifecycle never reopens', () => {
  const changed = structuredClone(examples.producer);
  changed.players[0].stats.match_average = 61;
  validate(changed);
  assert.notEqual(digest(changed), digest(examples.producer));
  const beforeDateChange = digest(changed);
  changed.playedAt = '2025-06-15T10:00:00+01:00';
  assert.notEqual(digest(changed), beforeDateChange);
  for (const row of examples.lifecycle_cases) {
    assert.equal(row.outcome, row.same_digest ? 'duplicate' : 'pending');
  }
});
check('public projection excludes private source; lists exclude nested detail', () => {
  const publicJSON = JSON.stringify([examples.public_detail, examples.public_summary_only]);
  for (const field of ['account_id', 'match_player_id', 'external_match_id', 'source_payload']) assert(!publicJSON.includes(`"${field}"`));
  for (const field of ['detail', 'legs', 'visits', 'throws', 'source_payload']) assert(!JSON.stringify(examples.admin_list).includes(`"${field}"`));
  const coverage = examples.public_detail.coverage;
  assert.equal(coverage.position_fraction, coverage.known_positions / coverage.recorded_throws);
  assert.equal(examples.public_summary_only.coverage.position_fraction, null);
});
check('mean-of-known-match averages differs from later weighted metric', () => {
  const matches = [{ average: 60, points: 600, darts: 30 }, { average: 90, points: 300, darts: 10 }, { average: null }];
  const known = matches.filter(m => m.average !== null);
  assert.equal(known.reduce((sum, m) => sum + m.average, 0) / known.length, 75);
  assert.equal(3 * known.reduce((sum, m) => sum + m.points, 0) / known.reduce((sum, m) => sum + m.darts, 0), 67.5);
});
check('publication policy excludes unrevealed, pending and superseded aggregates', () => {
  const visible = (status, current, reveal, now) => status === 'confirmed' && current && Date.parse(reveal) <= Date.parse(now);
  for (const status of ['pending', 'rejected', 'review_blocked', 'superseded']) assert.equal(visible(status, true, '2026-06-15T08:00:00Z', '2026-06-15T10:00:00Z'), false);
  assert.equal(visible('confirmed', false, '2026-06-15T08:00:00Z', '2026-06-15T10:00:00Z'), false);
  for (const reveal of ['2026-03-30T08:00:00Z', '2026-10-26T09:00:00Z']) {
    const local = new Intl.DateTimeFormat('en-GB', { timeZone: 'Europe/London', weekday: 'long', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).format(new Date(reveal));
    assert(local.includes('Monday') && local.includes('09:00'));
    assert.equal(visible('confirmed', true, reveal, new Date(Date.parse(reveal) - 1).toISOString()), false);
    assert.equal(visible('confirmed', true, reveal, reveal), true);
  }
});
check('replacement snapshot matches the reviewed result and needs explicit intent', () => {
  const { fixture_id, winner_id, ...snapshot } = examples.reversed_confirmation.response;
  assert.equal(winner_id, examples.reversed_confirmation.request.player_one_id);
  assert.equal(examples.replacement_request.fixture_id, fixture_id);
  assert.equal(examples.replacement_request.season_id, examples.reversed_confirmation.fixture.season_id);
  assert.equal(examples.replacement_request.replace_result, true);
  assert(examples.replacement_request.reason.length > 0);
  assert.deepEqual(examples.replacement_request.expected_result, snapshot);
});
check('API errors preserve existing code/message error shape', () => {
  for (const error of examples.errors) {
    keys(error.body, ['error']);
    keys(error.body.error, ['code', 'message']);
    assert(typeof error.body.error.message === 'string');
    assert(typeof error.body.error.code === 'string');
  }
});
console.log(`Passed ${checks} contract-example checks (not production integration tests).`);
