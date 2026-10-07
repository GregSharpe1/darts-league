import type { PendingResult } from './api'
import type { MatchLeg, MatchStats, MatchThrow, Position, Provenance, Segment } from '../components/match/types'
import type { PendingDetail } from './pendingReview'

function invalid(): never { throw new TypeError('The selected import has an invalid response. Reload or contact an administrator.') }
function object(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return invalid()
  return Object.fromEntries(Object.entries(value))
}
function text(value: unknown): string { return typeof value === 'string' ? value : invalid() }
function number(value: unknown): number { return typeof value === 'number' && Number.isFinite(value) ? value : invalid() }
function bool(value: unknown): boolean { return typeof value === 'boolean' ? value : invalid() }
function array(value: unknown): unknown[] { return Array.isArray(value) ? value : invalid() }
function nullable<T>(value: unknown, parse: (value: unknown) => T): T | null { return value == null ? null : parse(value) }
function provenance(value: unknown): Provenance { return value === 'automatic' || value === 'manual' || value === 'unknown' ? value : invalid() }

function stats(value: unknown): MatchStats {
  const data = object(value)
  const optional = Object.fromEntries(['first_nine_average', 'average_until_170', 'highest_finish', 'total_180', 'less_60', 'plus_60', 'plus_100', 'plus_140', 'plus_170']
    .filter(key => key in data).map(key => [key, nullable(data[key], number)]))
  return { ...optional, match_average: nullable(data.match_average, number), points_scored: nullable(data.points_scored, number), darts_thrown: nullable(data.darts_thrown, number), checkout_hits: nullable(data.checkout_hits, number), checkout_attempts: nullable(data.checkout_attempts, number) }
}
function segment(value: unknown): Segment {
  const data = object(value)
  const n = number(data.number)
  switch (data.bed) {
    case 'miss': return n === 0 ? { bed: data.bed, number: 0 } : invalid()
    case 'outer_bull': case 'inner_bull': return n === 25 ? { bed: data.bed, number: 25 } : invalid()
    case 'single': case 'double': case 'triple': return Number.isInteger(n) && n >= 1 && n <= 20 ? { bed: data.bed, number: n } : invalid()
    default: return invalid()
  }
}
function position(value: unknown): Position {
  const data = object(value)
  return { x: number(data.x), y: number(data.y), units: nullable(data.units, text), origin: nullable(data.origin, text), axis_orientation: nullable(data.axis_orientation, text), provenance: provenance(data.provenance) }
}
function dart(value: unknown): MatchThrow {
  const data = object(value)
  return { number: number(data.number), segment: segment(data.segment), entry_type: provenance(data.entry_type), position: nullable(data.position, position) }
}
function leg(value: unknown): MatchLeg {
  const data = object(value)
  return { number: number(data.number), completed: bool(data.completed), winner_id: nullable(data.winner_id, text), visits: array(data.visits).map(value => {
    const visit = object(value)
    return { number: number(visit.number), player_id: text(visit.player_id), start_remaining: number(visit.start_remaining), end_remaining: number(visit.end_remaining), bust: bool(visit.bust), throws: array(visit.throws).map(dart) }
  }) }
}
function summary(value: unknown): PendingResult {
  const data = object(value)
  const status = data.status
  if (status !== 'pending' && status !== 'review_blocked' && status !== 'confirmed' && status !== 'rejected') return invalid()
  return { id: number(data.id), external_match_id: text(data.external_match_id), player_one_name: text(data.player_one_name), player_two_name: text(data.player_two_name), player_one_legs: number(data.player_one_legs), player_two_legs: number(data.player_two_legs),
    ...(data.player_one_average == null ? {} : { player_one_average: number(data.player_one_average) }), ...(data.player_two_average == null ? {} : { player_two_average: number(data.player_two_average) }), status, received_at: text(data.received_at) }
}

export function parsePendingDetail(value: unknown): PendingDetail {
  const data = object(value)
  return {
    pending_result: summary(data.pending_result), source: text(data.source), digest: text(data.digest), changed_import: bool(data.changed_import), settings_evidence: text(data.settings_evidence), review_reason: text(data.review_reason), played_at: nullable(data.played_at, text),
    players: array(data.players).map(value => {
      const player = object(value)
      return { match_player_id: text(player.match_player_id), account_id: nullable(player.account_id, text), display_name: text(player.display_name), legs_won: number(player.legs_won), stats: nullable(player.stats, stats) }
    }),
    detail: nullable(data.detail, value => {
      const detail = object(value)
      if (detail.coverage !== 'partial' && detail.coverage !== 'complete') return invalid()
      return { coverage: detail.coverage, legs: array(detail.legs).map(leg) }
    }),
    original_payload: object(data.original_payload), season_id: nullable(data.season_id, number), fixture_id: nullable(data.fixture_id, number), result_id: nullable(data.result_id, number), source_active: bool(data.source_active),
    mapping: nullable(data.mapping, value => Object.fromEntries(Object.entries(object(value)).map(([key, value]) => [key, number(value)]))), approval: nullable(data.approval, object),
  }
}
