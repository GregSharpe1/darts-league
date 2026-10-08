import reference from '../../../docs/autodarts-fixtures/randomized.json'
import type { MatchData, MatchLeg, MatchPlayer, MatchThrow, Segment } from '../../src/components/match/types'

function segment(value: string): Segment {
  const number = Number(value.slice(1))
  if (value.startsWith('M')) return { bed: 'miss', number: 0 }
  if (value.startsWith('T')) return { bed: 'triple', number }
  if (value.startsWith('D')) return { bed: 'double', number }
  throw new TypeError('Unexpected synthetic fixture segment')
}

function player(index: number): MatchPlayer {
  const source = reference.players[index]
  const stats = reference.matchStats.find(item => item.playerId === source.id)
  if (!stats) throw new TypeError('Missing synthetic fixture stats')
  return { id: source.id, label: source.name, legs_won: source.legsWon,
    stats: { match_average: stats.average, darts_thrown: stats.dartsThrown,
      points_scored: source.legsWon * 501, checkout_hits: stats.checkoutsHit, checkout_attempts: stats.checkouts } }
}

const legs: MatchLeg[] = [1, 2, 3].map(number => {
  const darts = reference.darts.filter(dart => dart.leg === number)
  const remaining = new Map(reference.players.map(item => [item.id, 501]))
  return { number, completed: true, winner_id: reference.winnerId,
    visits: [...new Set(darts.map(dart => dart.visit))].map(visit => {
      const visitDarts = darts.filter(dart => dart.visit === visit)
      const player_id = visitDarts[0].playerId
      const start_remaining = remaining.get(player_id) ?? 501
      const score = visitDarts.reduce((sum, dart) => sum + (dart.segment.startsWith('M') ? 0 :
        Number(dart.segment.slice(1)) * (dart.segment.startsWith('T') ? 3 : 2)), 0)
      const end_remaining = start_remaining - score
      remaining.set(player_id, end_remaining)
      const throws: MatchThrow[] = visitDarts.map(dart => ({ number: dart.dart + 1,
        segment: segment(dart.segment), entry_type: 'manual',
        position: { x: dart.x, y: dart.y, units: 'board-radius', origin: 'bull',
          axis_orientation: 'x-right-y-up', provenance: 'manual' } }))
      return { number: visit + 1, player_id, start_remaining, end_remaining, bust: false, throws }
    }) }
})

// Test-only declaration of the mockup's geometry; never apply to live raw imports.
export const syntheticMatch: MatchData = {
  players: [player(0), player(1)], playedAt: null, synthetic: true,
  detail: { coverage: 'complete', legs },
}

export function withoutGeometry(match: MatchData): MatchData {
  return { ...match, detail: match.detail && { ...match.detail,
    legs: match.detail.legs.map(leg => ({ ...leg, visits: leg.visits.map(visit => ({ ...visit,
      throws: visit.throws.map(dart => ({ ...dart, position: dart.position && { ...dart.position, units: null } })),
    })) })),
  } }
}
