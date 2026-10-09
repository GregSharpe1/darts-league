import { readFile } from 'node:fs/promises';

const fixture = JSON.parse(await readFile(new URL('../docs/autodarts-fixtures/manual.json', import.meta.url)));

export function source() {
  const state = {
    id: 'offline-match', createdAt: '2026-06-15T10:00:00+01:00', finishedAt: '2026-06-15T11:00:00+01:00',
    variant: 'X01', targetLegs: 3, targetSets: null, winner: 0,
    settings: { baseScore: 501, outMode: 'Double' },
    players: fixture.players.map(p => ({ id: p.id, name: p.name, userId: null, user: { email: 'private@example.invalid' } })),
    scores: fixture.players.map(p => ({ legs: p.legsWon })), matchStats: structuredClone(fixture.matchStats),
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
}
