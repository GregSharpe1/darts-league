// Opt-in projection only. Routine checks use the committed JSON, never raw captures.
import { readFile, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import assert from 'node:assert/strict';

const read = async path => JSON.parse(await readFile(path, 'utf8'));
const save = (name, value) => writeFile(new URL(name, import.meta.url), JSON.stringify(value, null, 2) + '\n');
const fields = ['average', 'first9Average', 'checkouts', 'checkoutsHit', 'total180', 'dartsThrown'];
function project(source, provenance) {
  const playerId = id => {
    const index = source.players.findIndex(player => player.id === id);
    assert(index >= 0, 'Unknown player reference');
    return `demo-player-${index + 1}`;
  };
  const stats = row => ({ playerId: playerId(row.playerId), ...Object.fromEntries(fields.map(key => [key, row[key]])) });
  return {
    provenance, matchId: 'demo-match-001', winnerId: playerId(source.players[source.winner].id),
    players: source.players.map((player, index) => ({ id: playerId(player.id), name: ['Morgan Ember', 'Casey Vale'][index], legsWon: source.scores[index].legs })),
    matchStats: source.matchStats.map(stats),
    legStats: source.legStats.map(leg => ({ leg: leg.leg, stats: leg.stats.map(stats) })),
    darts: source.games.flatMap(game => game.turns.flatMap(turn => turn.throws.map(dart => ({
      id: `demo-leg-${game.leg + 1}-visit-${turn.turn}-dart-${dart.throw}`,
      gameId: `demo-leg-${game.leg + 1}`, visitId: `demo-leg-${game.leg + 1}-visit-${turn.turn}`,
      playerId: playerId(turn.playerId), leg: game.leg + 1, visit: turn.turn, dart: dart.throw,
      segment: dart.segment.name, x: dart.coords.x, y: dart.coords.y,
    })))),
  };
}
assert.equal(process.argv.length, 4, 'Usage: node prepare.mjs MANUAL_CAPTURE RANDOMIZED_CAPTURE (private local inputs)');
const manual = project((await read(process.argv[2])).response.body, 'Observed manual browser-entry test; coordinates are not physical board measurements. Fictitious identities.');
const randomized = project((await read(process.argv[3])).response.body, 'User-supplied synthetic randomized coordinates; exact positions preserved, not physical accuracy. Fictitious identities.');
assert.deepEqual(manual.darts.map(({ x, y, ...dart }) => dart), randomized.darts.map(({ x, y, ...dart }) => dart));
const missing = structuredClone(randomized);
missing.provenance = 'Synthetic missing-coordinate variant; segments and statistics retained, every position removed.';
missing.darts.forEach(dart => { dart.x = null; dart.y = null; });
const scoreOnly = { provenance: 'Synthetic score-only variant; no statistics or dart evidence available.', matchId: manual.matchId, winnerId: manual.winnerId, players: manual.players };
const reversed = structuredClone(randomized);
reversed.provenance = 'Synthetic ordering variant of randomized fixture; all IDs and positions unchanged.';
reversed.players.reverse(); reversed.matchStats.reverse(); reversed.legStats.reverse(); reversed.darts.reverse();
reversed.legStats.forEach(leg => leg.stats.reverse());
const hashes = {};
for (const [name, fixture] of Object.entries({ manual, randomized, 'missing-coordinate': missing, 'score-only': scoreOnly, 'reversed-order': reversed })) {
  await save(`${name}.json`, fixture);
  if (fixture.darts) hashes[name] = createHash('sha256').update(JSON.stringify(fixture.darts)).digest('hex');
}
await save('dart-hashes.json', hashes);
console.log('Prepared allowlisted projections; raw profiles, host data, timestamps and IDs excluded.');
