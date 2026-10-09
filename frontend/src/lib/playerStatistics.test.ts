import { afterEach, expect, it, vi } from 'vitest'
import fixture from './playerStatistics.fixture.json'
import { fetchPlayerStatistics, parsePlayerStatistics, playerPlotThrows } from './playerStatistics'

afterEach(() => vi.unstubAllGlobals())

it('preserves known zero, missing metrics, and per-metric coverage', () => {
  const got = parsePlayerStatistics(fixture)
  expect(got.total_180).toBe(0)
  expect(got.dart_weighted_average).toBeNull()
  expect(got.checkout_percentage).toBe(0)
  expect(got.coverage.first_nine_match_average_mean).toBe(1)
})

it('honors plottable even when automatic geometry would otherwise render', () => {
  const got = parsePlayerStatistics(fixture)
  expect(playerPlotThrows(got.throws).map(d => d.position?.x ?? null)).toEqual([0, null, null])
  expect(got.coverage.known_positions).toBe(2)
})

it.each([null, {}, { ...fixture, player_id: -1 }, { ...fixture, total_180: undefined },
  { ...fixture, coverage: { ...fixture.coverage, total_180: 3 } },
  { ...fixture, coverage: { ...fixture.coverage, known_positions: 3 } },
  { ...fixture, throws: [{ ...fixture.throws[0], plottable: 'yes' }] },
  { ...fixture, history: [{ ...fixture.history[0], detail_coverage: 'locked' }] },
])('rejects malformed or inconsistent statistics', value => {
  expect(() => parsePlayerStatistics(value)).toThrow()
})

it('retains zero attempts without fabricating a percentage', () => {
  expect(parsePlayerStatistics({ ...fixture, checkout_attempts: 0, checkout_percentage: null }).checkout_percentage).toBeNull()
})

it('fetches explicit season and player with no-store and cancellation', async () => {
  const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify(fixture)))
  vi.stubGlobal('fetch', fetcher)
  await fetchPlayerStatistics({ seasonId: '2', playerId: '7', admin: true }, new AbortController().signal)
  expect(fetcher).toHaveBeenCalledWith('/api/admin/seasons/2/players/7/statistics', expect.objectContaining({ cache: 'no-store', credentials: 'include', signal: expect.any(AbortSignal) }))
})

it('rejects mismatched scope and invalid IDs', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(fixture))))
  await expect(fetchPlayerStatistics({ seasonId: '3', playerId: '7', admin: false })).rejects.toThrow()
  await expect(fetchPlayerStatistics({ seasonId: '2', playerId: 'Arrow', admin: false })).rejects.toThrow()
})
