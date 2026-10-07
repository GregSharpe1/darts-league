import { ApiError } from './api'
import type { MatchThrow, Position, Provenance, Segment } from '../components/match'

function invalid(): never { throw new ApiError(502, 'invalid_statistics', 'Player statistics could not be loaded.') }
function isObject(value: unknown): value is Record<string, unknown> { return typeof value === 'object' && value !== null && !Array.isArray(value) }
function object(value: unknown): Record<string, unknown> { return isObject(value) ? value : invalid() }
function text(value: unknown): string { return typeof value === 'string' && value.length > 0 && value.length <= 128 ? value : invalid() }
function number(value: unknown, max = Number.MAX_SAFE_INTEGER): number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0 && value <= max ? value : invalid()
}
function integer(value: unknown, max = Number.MAX_SAFE_INTEGER): number { const n = number(value, max); return Number.isSafeInteger(n) ? n : invalid() }
function id(value: unknown): number { const n = integer(value); return n > 0 ? n : invalid() }
function nullable(value: unknown, max = Number.MAX_SAFE_INTEGER): number | null { return value === null ? null : number(value, max) }
function count(value: unknown, max = Number.MAX_SAFE_INTEGER): number | null { return value === null ? null : integer(value, max) }
function boolean(value: unknown): boolean { return typeof value === 'boolean' ? value : invalid() }
function array(value: unknown): readonly unknown[] { return Array.isArray(value) ? value : invalid() }
function provenance(value: unknown): Provenance { return value === 'manual' || value === 'automatic' || value === 'unknown' ? value : invalid() }
function segment(value: unknown): Segment {
  const s = object(value)
  switch (s.bed) {
    case 'miss': return s.number === 0 ? { bed: s.bed, number: 0 } : invalid()
    case 'outer_bull': case 'inner_bull': return s.number === 25 ? { bed: s.bed, number: 25 } : invalid()
    case 'single': case 'double': case 'triple': { const n = id(s.number); return n <= 20 ? { bed: s.bed, number: n } : invalid() }
    default: return invalid()
  }
}
function position(value: unknown): Position | null {
  if (value === null) return null
  const p = object(value)
  if (typeof p.x !== 'number' || !Number.isFinite(p.x) || typeof p.y !== 'number' || !Number.isFinite(p.y)) return invalid()
  return { x: p.x, y: p.y, units: p.units === null ? null : text(p.units), origin: p.origin === null ? null : text(p.origin),
    axis_orientation: p.axis_orientation === null ? null : text(p.axis_orientation), provenance: provenance(p.provenance) }
}
function historyMatch(value: unknown) {
  const h = object(value)
  const scheduled_at = text(h.scheduled_at)
  if (!/^\d{4}-\d{2}-\d{2}T.*(?:Z|[+-]\d{2}:\d{2})$/.test(scheduled_at) || !Number.isFinite(Date.parse(scheduled_at))) return invalid()
  if (h.detail_coverage !== 'none' && h.detail_coverage !== 'partial' && h.detail_coverage !== 'complete') return invalid()
  return { fixture_id: id(h.fixture_id), week_number: id(h.week_number), scheduled_at, opponent_id: id(h.opponent_id),
    opponent_name: text(h.opponent_name), won: boolean(h.won), legs_for: integer(h.legs_for, 3), legs_against: integer(h.legs_against, 3),
    match_average: nullable(h.match_average, 180), detail_coverage: h.detail_coverage } as const
}
function recordedThrow(value: unknown) {
  const d = object(value)
  const throw_number = id(d.throw_number)
  if (throw_number > 3) return invalid()
  const point = position(d.position)
  const plottable = boolean(d.plottable)
  if (plottable && point === null) return invalid()
  return { fixture_id: id(d.fixture_id), leg_number: id(d.leg_number), visit_number: id(d.visit_number), throw_number,
    segment: segment(d.segment), entry_type: provenance(d.entry_type), position: point, plottable } as const
}

export function parsePlayerStatistics(value: unknown) {
  const s = object(value)
  if (s.schema_version !== 'autodarts.api.v1') return invalid()
  const c = object(s.coverage)
  const eligible = integer(c.eligible_matches)
  const coverage = {
    eligible_matches: eligible, matches_with_detail: integer(c.matches_with_detail, eligible),
    match_average_mean: integer(c.match_average_mean, eligible), dart_weighted_average: integer(c.dart_weighted_average, eligible),
    first_nine_match_average_mean: integer(c.first_nine_match_average_mean, eligible), checkout: integer(c.checkout, eligible),
    total_180: integer(c.total_180, eligible), best_leg_darts: integer(c.best_leg_darts, eligible), highest_finish: integer(c.highest_finish, eligible),
    matches_with_recorded_throws: integer(c.matches_with_recorded_throws, eligible), matches_with_positions: integer(c.matches_with_positions, eligible),
    recorded_throws: integer(c.recorded_throws), known_positions: integer(c.known_positions), plottable_positions: integer(c.plottable_positions),
    position_fraction: nullable(c.position_fraction, 1),
  } as const
  const history = array(s.history).map(historyMatch)
  const throws = array(s.throws).map(recordedThrow)
  const fixtureIDs = new Set(history.map(h => h.fixture_id))
  if (history.length !== eligible || fixtureIDs.size !== eligible || throws.some(d => !fixtureIDs.has(d.fixture_id)) ||
    coverage.recorded_throws !== throws.length || coverage.known_positions !== throws.filter(d => d.position !== null).length ||
    coverage.plottable_positions !== throws.filter(d => d.plottable).length ||
    coverage.position_fraction !== (throws.length === 0 ? null : coverage.known_positions / throws.length)) return invalid()
  if (typeof s.leg_difference !== 'number' || !Number.isSafeInteger(s.leg_difference)) return invalid()
  const result = {
    season_id: id(s.season_id), player_id: id(s.player_id), preferred_name: text(s.preferred_name),
    played: integer(s.played), won: integer(s.won), lost: integer(s.lost), legs_for: integer(s.legs_for), legs_against: integer(s.legs_against),
    leg_difference: s.leg_difference, points: integer(s.points), match_average_mean: nullable(s.match_average_mean, 180),
    dart_weighted_average: nullable(s.dart_weighted_average, 180), first_nine_match_average_mean: nullable(s.first_nine_match_average_mean, 180),
    checkout_hits: count(s.checkout_hits), checkout_attempts: count(s.checkout_attempts), checkout_percentage: nullable(s.checkout_percentage, 100),
    total_180: count(s.total_180), best_leg_darts: count(s.best_leg_darts), highest_finish: count(s.highest_finish, 170), coverage, history, throws,
  } as const
  for (const key of ['match_average_mean', 'dart_weighted_average', 'first_nine_match_average_mean', 'total_180', 'best_leg_darts', 'highest_finish'] as const) {
    if ((result[key] === null) !== (coverage[key] === 0)) return invalid()
  }
  if (result.played !== eligible || result.won + result.lost !== eligible || result.leg_difference !== result.legs_for - result.legs_against ||
    (result.checkout_hits === null) !== (coverage.checkout === 0) || (result.checkout_attempts === null) !== (coverage.checkout === 0)) return invalid()
  const { checkout_hits: hits, checkout_attempts: attempts, checkout_percentage: percentage } = result
  if (hits !== null && attempts !== null && (hits > attempts || (attempts === 0 ? percentage !== null : percentage === null || Math.abs(percentage - 100 * hits / attempts) > 1e-8))) return invalid()
  if (hits === null && percentage !== null) return invalid()
  return result
}

export type PlayerStatistics = ReturnType<typeof parsePlayerStatistics>
export function playerPlotThrows(throws: PlayerStatistics['throws']): readonly MatchThrow[] {
  return throws.map(d => ({ number: d.throw_number, segment: d.segment, entry_type: d.entry_type, position: d.plottable ? d.position : null }))
}

export async function fetchPlayerStatistics(scope: { readonly seasonId: string; readonly playerId: string; readonly admin: boolean }, signal?: AbortSignal): Promise<PlayerStatistics> {
  const { seasonId, playerId, admin } = scope
  if (![seasonId, playerId].every(id => /^[1-9]\d*$/.test(id) && Number.isSafeInteger(Number(id)))) throw new ApiError(404, 'not_found', 'Player not found in this season.')
  const timeout = AbortSignal.timeout(15000)
  const response = await fetch(`/api/${admin ? 'admin/' : ''}seasons/${seasonId}/players/${playerId}/statistics`, {
    credentials: 'include', cache: 'no-store', signal: signal ? AbortSignal.any([signal, timeout]) : timeout,
  })
  if (!response.ok) throw new ApiError(response.status, 'statistics_unavailable', 'Player statistics could not be loaded.')
  const value: unknown = await response.json()
  const parsed = parsePlayerStatistics(value)
  if (parsed.season_id !== Number(seasonId) || parsed.player_id !== Number(playerId)) return invalid()
  return parsed
}
