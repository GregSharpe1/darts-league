import { ApiError } from './api'
import type { MatchData, MatchPlayer, MatchStats, MatchLeg, MatchVisit, MatchThrow, Position, Provenance, Segment } from '../components/match'

export interface FixtureMatch {
  readonly fixtureId: number
  readonly seasonId: number
  readonly match: MatchData
  readonly coverage: { readonly recorded_throws: number; readonly known_positions: number; readonly position_fraction: number | null }
}

function invalid(): never { throw new ApiError(502, 'invalid_match', 'Match details could not be loaded.') }
function isObject(value: unknown): value is Record<string, unknown> { return typeof value === 'object' && value !== null && !Array.isArray(value) }
function object(value: unknown): Record<string, unknown> { return isObject(value) ? value : invalid() }
function text(value: unknown): string { return typeof value === 'string' && value.length > 0 && value.length <= 128 ? value : invalid() }
function nullableText(value: unknown): string | null { return value === null ? null : text(value) }
function number(value: unknown, max = Number.MAX_SAFE_INTEGER): number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0 && value <= max ? value : invalid()
}
function integer(value: unknown, max = Number.MAX_SAFE_INTEGER): number { const n = number(value, max); return Number.isSafeInteger(n) ? n : invalid() }
function id(value: unknown): number { const n = integer(value); return n > 0 ? n : invalid() }
function nullableNumber(value: unknown, max: number): number | null { return value === null || value === undefined ? null : number(value, max) }
function nullableCount(value: unknown, max: number): number | null { return value === null || value === undefined ? null : integer(value, max) }
function boolean(value: unknown): boolean { return typeof value === 'boolean' ? value : invalid() }
function array(value: unknown, max: number): readonly unknown[] { return Array.isArray(value) && value.length <= max ? value : invalid() }
function provenance(value: unknown): Provenance {
  return value === 'automatic' || value === 'manual' || value === 'unknown' ? value : invalid()
}

function stats(value: unknown): MatchStats | null {
  if (value === null) return null
  const s = object(value)
  return {
    match_average: nullableNumber(s.match_average, 180), points_scored: nullableCount(s.points_scored, 5010),
    darts_thrown: nullableCount(s.darts_thrown, 3000), checkout_hits: nullableCount(s.checkout_hits, 3), checkout_attempts: nullableCount(s.checkout_attempts, 3000),
    first_nine_average: nullableNumber(s.first_nine_average, 180), average_until_170: nullableNumber(s.average_until_170, 180),
    highest_finish: nullableCount(s.highest_finish, 170), total_180: nullableCount(s.total_180, 1000),
    less_60: nullableCount(s.less_60, 1000), plus_60: nullableCount(s.plus_60, 1000), plus_100: nullableCount(s.plus_100, 1000),
    plus_140: nullableCount(s.plus_140, 1000), plus_170: nullableCount(s.plus_170, 1000),
  }
}

function player(value: unknown): MatchPlayer {
  const p = object(value)
  return { id: String(id(p.league_player_id)), label: text(p.preferred_name), legs_won: integer(p.legs_won), stats: stats(p.stats) }
}

function segment(value: unknown): Segment {
  const s = object(value)
  switch (s.bed) {
    case 'miss': return s.number === 0 ? { bed: s.bed, number: 0 } : invalid()
    case 'outer_bull': case 'inner_bull': return s.number === 25 ? { bed: s.bed, number: 25 } : invalid()
    case 'single': case 'double': case 'triple': {
      const n = integer(s.number, 20)
      return n > 0 ? { bed: s.bed, number: n } : invalid()
    }
    default: return invalid()
  }
}

function position(value: unknown): Position | null {
  if (value === null) return null
  const p = object(value)
  if (typeof p.x !== 'number' || !Number.isFinite(p.x) || typeof p.y !== 'number' || !Number.isFinite(p.y)) return invalid()
  return { x: p.x, y: p.y, units: nullableText(p.units), origin: nullableText(p.origin), axis_orientation: nullableText(p.axis_orientation), provenance: provenance(p.provenance) }
}

function dart(value: unknown): MatchThrow {
  const d = object(value)
  const n = integer(d.number, 3)
  if (n === 0) return invalid()
  return { number: n, segment: segment(d.segment), entry_type: provenance(d.entry_type), position: position(d.position) }
}

function reference(value: unknown, players: readonly MatchPlayer[]): string {
  const ref = text(value)
  return players.some(p => p.id === ref) ? ref : invalid()
}

function visit(value: unknown, players: readonly MatchPlayer[]): MatchVisit {
  const v = object(value)
  const n = integer(v.number, 200)
  const throws = array(v.throws, 3).map(dart)
  if (n === 0 || throws.length === 0 || throws.some((d, i) => d.number !== i + 1)) return invalid()
  return { number: n, player_id: reference(v.player_id, players), start_remaining: integer(v.start_remaining, 501), end_remaining: integer(v.end_remaining, 501), bust: boolean(v.bust), throws }
}

function detail(value: unknown, players: readonly MatchPlayer[]): MatchData['detail'] {
  if (value === null) return null
  const d = object(value)
  if (d.coverage !== 'partial' && d.coverage !== 'complete') return invalid()
  const legs = array(d.legs, 5).map((value): MatchLeg => {
    const l = object(value)
    const n = integer(l.number, 5)
    if (n === 0) return invalid()
    const completed = boolean(l.completed)
    const winner = l.winner_id === null ? null : reference(l.winner_id, players)
    if (completed !== (winner !== null)) return invalid()
    const visits = array(l.visits, 200).map(v => visit(v, players))
    if (visits.some((v, i) => i > 0 && visits[i - 1].number >= v.number)) return invalid()
    return { number: n, completed, winner_id: winner, visits }
  })
  if (new Set(legs.map(l => l.number)).size !== legs.length) return invalid()
  return { coverage: d.coverage, legs }
}

export function parseMatch(value: unknown): FixtureMatch {
  const data = object(value)
  if (data.schema_version !== 'autodarts.api.v1') return invalid()
  const players = array(data.players, 2).map(player)
  const [one, two] = players
  if (!one || !two || one.id === two.id) return invalid()
  const playedAt = nullableText(data.playedAt)
  if (playedAt !== null && (!/^\d{4}-\d{2}-\d{2}T.*(?:Z|[+-]\d{2}:\d{2})$/.test(playedAt) || !Number.isFinite(Date.parse(playedAt)))) return invalid()
  const match: MatchData = { players: [one, two], playedAt, detail: detail(data.detail, players), synthetic: false }
  const c = object(data.coverage)
  const recorded_throws = integer(c.recorded_throws, 3000)
  const known_positions = integer(c.known_positions, recorded_throws)
  const position_fraction = c.position_fraction === null ? null : number(c.position_fraction, 1)
  const darts = match.detail?.legs.flatMap(l => l.visits.flatMap(v => v.throws)) ?? []
  if (recorded_throws !== darts.length || known_positions !== darts.filter(d => d.position !== null).length ||
      position_fraction !== (recorded_throws === 0 ? null : known_positions / recorded_throws)) return invalid()
  return { fixtureId: id(data.fixture_id), seasonId: id(data.season_id), match, coverage: { recorded_throws, known_positions, position_fraction } }
}

export async function fetchMatch(fixtureId: string, admin: boolean, signal?: AbortSignal): Promise<FixtureMatch> {
  if (!/^[1-9]\d*$/.test(fixtureId) || !Number.isSafeInteger(Number(fixtureId))) throw new ApiError(404, 'not_found', 'Match not found.')
  const timeout = AbortSignal.timeout(15000)
  const response = await fetch(`/api/${admin ? 'admin/' : ''}fixtures/${fixtureId}/autodarts`, {
    credentials: 'include', cache: 'no-store', signal: signal ? AbortSignal.any([signal, timeout]) : timeout,
  })
  if (!response.ok) throw new ApiError(response.status, 'match_unavailable', 'Match details could not be loaded.')
  const value: unknown = await response.json()
  const parsed = parseMatch(value)
  if (parsed.fixtureId !== Number(fixtureId)) return invalid()
  return parsed
}
