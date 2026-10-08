import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
const read = async name => JSON.parse(await readFile(new URL(`${name}.json`, import.meta.url), 'utf8'));
const hashes = await read('dart-hashes');
const fixtures = {};
const points = { T20: 60, T19: 57, D12: 24, M20: 0 };
const sorted = darts => [...darts].sort((a, b) => a.id.localeCompare(b.id));
for (const name of ['manual', 'randomized', 'missing-coordinate', 'score-only', 'reversed-order']) {
  const f = fixtures[name] = await read(name);
  assert.equal(f.matchId, 'demo-match-001');
  assert.equal(f.winnerId, 'demo-player-1');
  assert.deepEqual([...f.players].sort((a, b) => a.id.localeCompare(b.id)), [
    { id: 'demo-player-1', name: 'Morgan Ember', legsWon: 3 },
    { id: 'demo-player-2', name: 'Casey Vale', legsWon: 0 },
  ]);
  assert.doesNotMatch(JSON.stringify(f), /https?:|hostId|profile|token|password|createdAt|[0-9a-f]{8}-[0-9a-f]{4}-/i);
  if (name === 'score-only') { assert.deepEqual(Object.keys(f).sort(), ['matchId', 'players', 'provenance', 'winnerId']); continue; }
  assert.equal(f.darts.length, 48);
  assert.deepEqual(f.legStats.map(leg => leg.leg).sort(), [1,2,3]);
  for (const rows of [f.matchStats, ...f.legStats.map(leg => leg.stats)]) {
    assert.deepEqual(rows.map(row => row.playerId).sort(), ['demo-player-1', 'demo-player-2']);
  }
  assert.equal(new Set(f.darts.map(d => d.id)).size, 48);
  assert.equal(createHash('sha256').update(JSON.stringify(f.darts)).digest('hex'), hashes[name], 'Coordinate/relationship integrity');
  for (const d of f.darts) {
    assert([1, 2, 3].includes(d.leg));
    assert(Number.isInteger(d.visit) && d.visit >= 0);
    assert([0, 1, 2].includes(d.dart));
    assert.equal(d.gameId, `demo-leg-${d.leg}`);
    assert.equal(d.visitId, `${d.gameId}-visit-${d.visit}`);
    assert.equal(d.id, `${d.visitId}-dart-${d.dart}`);
    assert(f.players.some(p => p.id === d.playerId));
    if (name === 'missing-coordinate') { assert.equal(d.x, null); assert.equal(d.y, null); }
    else { assert(Number.isFinite(d.x)); assert(Number.isFinite(d.y)); }
  }
  for (const player of f.players) {
    const winner = player.id === f.winnerId;
    const all = f.darts.filter(d => d.playerId === player.id);
    const stats = f.matchStats.find(s => s.playerId === player.id);
    assert.equal(all.length, winner ? 27 : 21);
    assert.equal(stats.dartsThrown, all.length);
    assert.equal(stats.average, winner ? 167 : 0);
    assert.equal(stats.total180, winner ? 6 : 0);
    assert.equal(stats.checkoutsHit, winner ? 3 : 0);
    for (const leg of f.legStats) {
      const darts = all.filter(d => d.leg === leg.leg);
      const s = leg.stats.find(row => row.playerId === player.id);
      assert.equal(darts.length, winner ? 9 : leg.leg === 2 ? 9 : 6);
      assert.equal(s.dartsThrown, darts.length);
      assert.equal(darts.reduce((sum, d) => sum + points[d.segment], 0), winner ? 501 : 0);
      assert.equal(s.average, winner ? 167 : 0);
      assert.equal(s.total180, winner ? 2 : 0);
      assert.equal(s.checkoutsHit, winner ? 1 : 0);
      const visits = [...new Set(darts.map(d => d.visit))].sort((a,b) => a-b);
      assert.deepEqual(visits.map(visit => darts.filter(d => d.visit === visit).reduce((sum,d) => sum + points[d.segment],0)), winner ? [180,180,141] : visits.map(() => 0));
      for (const visit of visits) assert.deepEqual(darts.filter(d => d.visit === visit).map(d => d.dart).sort(), [0,1,2]);
    }
    const segments = Object.fromEntries(Object.keys(points).map(segment => [segment, all.filter(d => d.segment === segment).length]));
    assert.deepEqual(segments, winner ? { T20:21,T19:3,D12:3,M20:0 } : { T20:0,T19:0,D12:0,M20:21 });
  }
  console.log(`PASS ${name}: totals, IDs, visits, positions, sanitized identities`);
}
assert.deepEqual(sorted(fixtures.randomized.darts), sorted(fixtures['reversed-order'].darts));
const relationships = darts => darts.map(({x,y,...dart}) => dart);
assert.deepEqual(relationships(fixtures.manual.darts), relationships(fixtures.randomized.darts));
assert.deepEqual(relationships(fixtures.randomized.darts), relationships(fixtures['missing-coordinate'].darts));
console.log('PASS score-only and cross-fixture relationships; no private inputs or network used');
