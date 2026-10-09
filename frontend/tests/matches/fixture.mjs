import { readFile } from 'node:fs/promises'

const source = JSON.parse(await readFile(new URL('../../../docs/autodarts-fixtures/randomized.json', import.meta.url), 'utf8'))
const ids = new Map(source.players.map((p, i) => [p.id, String(101 + i)]))
const legs = [1, 2, 3].map(number => {
  const darts = source.darts.filter(d => d.leg === number)
  const remaining = new Map(source.players.map(p => [p.id, 501]))
  return { number, completed: true, winner_id: ids.get(source.winnerId), visits: [...new Set(darts.map(d => d.visit))].map(visit => {
    const recorded = darts.filter(d => d.visit === visit)
    const player = recorded[0].playerId
    const start_remaining = remaining.get(player)
    const score = recorded.reduce((sum, d) => sum + (d.segment.startsWith('M') ? 0 : Number(d.segment.slice(1)) * (d.segment.startsWith('T') ? 3 : 2)), 0)
    remaining.set(player, start_remaining - score)
    return { number: visit + 1, player_id: ids.get(player), start_remaining, end_remaining: start_remaining - score, bust: false,
      throws: recorded.map(d => ({ number: d.dart + 1, segment: d.segment.startsWith('M') ? { bed: 'miss', number: 0 } : { bed: d.segment.startsWith('T') ? 'triple' : 'double', number: Number(d.segment.slice(1)) },
        entry_type: 'manual', position: { x: d.x, y: d.y, units: 'board-radius', origin: 'bull', axis_orientation: 'x-right-y-up', provenance: 'manual' } })) }
  }) }
})

// Offline synthetic geometry declaration only; never a physical adapter.
export const fixtureMatch = {
  schema_version: 'autodarts.api.v1', fixture_id: 7, season_id: 2, playedAt: null,
  players: source.players.map(p => {
    const stats = source.matchStats.find(s => s.playerId === p.id)
    return { league_player_id: Number(ids.get(p.id)), preferred_name: p.name, legs_won: p.legsWon,
      stats: { match_average: stats.average, points_scored: null, darts_thrown: stats.dartsThrown, checkout_hits: stats.checkoutsHit, checkout_attempts: stats.checkouts,
        first_nine_average: null, total_180: null, highest_finish: null } }
  }),
  detail: { coverage: 'complete', legs },
  coverage: { recorded_throws: 48, known_positions: 48, position_fraction: 1 },
}

export function matchState(state) {
  const data = structuredClone(fixtureMatch)
  if (state === 'summary') {
    data.detail = null
    data.coverage = { recorded_throws: 0, known_positions: 0, position_fraction: null }
    data.players.forEach(p => { p.stats = null })
  }
  if (state === 'unknown') data.detail.legs.forEach(l => l.visits.forEach(v => v.throws.forEach(d => { d.position.units = null })))
  return data
}
