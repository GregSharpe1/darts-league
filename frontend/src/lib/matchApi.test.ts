import { afterEach, describe, expect, it, vi } from 'vitest'
import { fetchMatch, parseMatch } from './matchApi'

const stats = { match_average: 60.12, points_scored: null, darts_thrown: null, checkout_hits: 3, checkout_attempts: null, first_nine_average: 72, total_180: 0, highest_finish: 120 }
const response = {
  schema_version: 'autodarts.api.v1', fixture_id: 7, season_id: 2, playedAt: null,
  players: [
    { league_player_id: 11, preferred_name: 'Arrow', legs_won: 3, stats },
    { league_player_id: 22, preferred_name: 'Bob', legs_won: 1, stats: null },
  ],
  detail: { coverage: 'partial', legs: [{ number: 1, completed: true, winner_id: '11', visits: [{ number: 1, player_id: '11', start_remaining: 40, end_remaining: 0, bust: false, throws: [{ number: 1, segment: { bed: 'double', number: 20 }, entry_type: 'manual', position: { x: 0, y: 0, units: null, origin: null, axis_orientation: null, provenance: 'unknown' } }] }] }] },
  coverage: { recorded_throws: 1, known_positions: 1, position_fraction: 1 },
}

afterEach(() => vi.unstubAllGlobals())

describe('match API boundary', () => {
  it('maps league IDs to strings and preserves optional stats, zero and raw coordinates', () => {
    const got = parseMatch(response)
    expect(got.match.players[0]).toEqual({ id: '11', label: 'Arrow', legs_won: 3, stats: { ...stats, average_until_170: null, less_60: null, plus_60: null, plus_100: null, plus_140: null, plus_170: null } })
    expect(got.match.detail?.legs[0].visits[0].throws[0].position).toEqual(response.detail.legs[0].visits[0].throws[0].position)
    expect(got.match.synthetic).toBe(false)
  })

  it('preserves score-only unknown evidence without inventing time or positions', () => {
    const got = parseMatch({ ...response, detail: null, coverage: { recorded_throws: 0, known_positions: 0, position_fraction: null } })
    expect(got.match.detail).toBeNull()
    expect(got.match.playedAt).toBeNull()
    expect(got.match.players[1].stats).toBeNull()
  })

  it.each([
    { ...response, schema_version: 'unknown' },
    { ...response, players: [] },
    { ...response, fixture_id: -1 },
    { ...response, playedAt: 'not-a-date' },
    { ...response, players: [response.players[0], response.players[0]] },
    { ...response, coverage: { recorded_throws: 0, known_positions: 1, position_fraction: 1 } },
    { ...response, detail: { coverage: 'complete', legs: [{ ...response.detail.legs[0], winner_id: 'source-id' }] } },
    { ...response, players: [{ ...response.players[0], stats: { ...stats, match_average: Infinity } }, response.players[1]] },
  ])('rejects malformed evidence', value => {
    expect(() => parseMatch(value)).toThrow()
  })

  it('allowlists fields instead of returning internal or source attributes', () => {
    const got = parseMatch({ ...response, digest: 'secret', players: response.players.map(p => ({ ...p, account_id: 'secret' })) })
    expect(JSON.stringify(got)).not.toContain('secret')
  })

  it('uses the admin endpoint with credentials and bypasses browser cache', async () => {
    const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify(response)))
    vi.stubGlobal('fetch', fetch)
    await fetchMatch('7', true)
    expect(fetch).toHaveBeenCalledWith('/api/admin/fixtures/7/autodarts', expect.objectContaining({ credentials: 'include', cache: 'no-store' }))
  })

  it('does not echo server source errors', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('private-source-id', { status: 404 })))
    await expect(fetchMatch('7', false)).rejects.toMatchObject({ status: 404, message: 'Match details could not be loaded.' })
  })
})
