import { describe, expect, it } from 'vitest'
import { approvalProblem, toMatchData, type PendingDetail } from './pendingReview'
import { parsePendingDetail } from './pendingDetail'

export const detail: PendingDetail = {
  pending_result: { id: 1, external_match_id: 'demo', player_one_name: 'A', player_two_name: 'B', player_one_legs: 3, player_two_legs: 1, status: 'pending', received_at: '2026-06-15T10:00:00Z' },
  source: 'autodarts', digest: 'demo-digest', changed_import: false, settings_evidence: 'source_reported', review_reason: '', played_at: '2026-06-15T09:00:00Z',
  players: [{ match_player_id: 'source-a', account_id: null, display_name: 'A', legs_won: 3, stats: { match_average: 60, points_scored: null, darts_thrown: null, checkout_hits: null, checkout_attempts: null, first_nine_average: 72, total_180: 2 } }, { match_player_id: 'source-b', account_id: null, display_name: 'B', legs_won: 1, stats: null }],
  detail: null, original_payload: {}, season_id: null, fixture_id: null, result_id: null, source_active: false, mapping: null, approval: null,
}

describe('source match adapter', () => {
  it('parses exact source player DTO and keeps optional metrics', () => {
    expect(parsePendingDetail(detail)).toEqual(detail)
  })
  it('rejects display-component player names at the API boundary', () => {
    expect(() => parsePendingDetail({ ...detail, players: [{ id: 'wrong', label: 'wrong' }] })).toThrow(TypeError)
  })
  it('preserves all extended source metrics without recomputing them', () => {
    const stats = { ...detail.players[0].stats, first_nine_average: 0, average_until_170: 67.123, highest_finish: 141, total_180: 2, less_60: 1, plus_60: 2, plus_100: 3, plus_140: 4, plus_170: 5 }
    expect(parsePendingDetail({ ...detail, players: [{ ...detail.players[0], stats }, detail.players[1]] }).players[0].stats).toEqual(stats)
  })
  it('rejects non-finite stats at the boundary', () => {
    expect(() => parsePendingDetail({ ...detail, players: [{ ...detail.players[0], stats: { match_average: Infinity } }] })).toThrow(TypeError)
  })
  it('preserves extended stats and source match identity when adapting', () => {
    const match = toMatchData(detail)
    expect(match?.players[0]).toEqual({ id: 'source-a', label: 'A', legs_won: 3, stats: detail.players[0].stats })
    expect(match?.players[1].stats).toBeNull()
    expect(match?.synthetic).toBe(false)
  })
  it('does not present blocked data as validated performance', () => {
    expect(toMatchData({ ...detail, review_reason: 'Totals conflict' })).toBeNull()
  })
  it('does not claim legacy format is source validated', () => {
    expect(toMatchData({ ...detail, settings_evidence: 'legacy_stored_summary' })).toBeNull()
  })
})

describe('approval guard', () => {
  const fixture = { id: 10, player_one_id: 11, player_two_id: 22, player_one: 'A', player_two: 'B', scheduled_at: '', game_variant: '501', legs_to_win: 3, status: 'unplayed', expected_result: null }
  const selection = { seasonId: 1, playerIds: [11, 22], fixture, replace: false, reason: '', attest: false, missingDateReason: '' }
  it('allows an explicit pair with an unscored snapshot', () => {
    expect(approvalProblem(detail, selection)).toBe('')
  })
  it.each([[11, 11], [11, 33], [0, 22]])('rejects invalid pair %j', (...playerIds) => {
    expect(approvalProblem(detail, { ...selection, playerIds })).not.toBe('')
  })
  it('rejects missing expected snapshot even when fixture looks unscored', () => {
    expect(approvalProblem(detail, { ...selection, fixture: { ...fixture, expected_result: undefined } })).not.toBe('')
  })
  it('requires replacement consent and reason for changed imports', () => {
    expect(approvalProblem({ ...detail, changed_import: true }, selection)).not.toBe('')
    expect(approvalProblem({ ...detail, changed_import: true }, { ...selection, replace: true, reason: 'Corrected capture' })).toBe('')
  })
  it('blocks an existing result without its concurrency snapshot', () => {
    expect(approvalProblem(detail, { ...selection, fixture: { ...fixture, result: { player_one_legs: 3, player_two_legs: 0, winner_id: 11 } } })).not.toBe('')
  })
  it('requires a selected season and fixed-format fixture', () => {
    expect(approvalProblem(detail, { ...selection, seasonId: 0 })).not.toBe('')
    expect(approvalProblem(detail, { ...selection, fixture: { ...fixture, legs_to_win: 5 } })).not.toBe('')
  })
  it('blocks changed source score and review-blocked status', () => {
    expect(approvalProblem({ ...detail, players: detail.players.map(player => ({ ...player, legs_won: 3 })) }, selection)).not.toBe('')
    expect(approvalProblem({ ...detail, pending_result: { ...detail.pending_result, status: 'review_blocked' } }, selection)).not.toBe('')
  })
  it('requires both format attestation and missing-date explanation for legacy', () => {
    const legacy = { ...detail, settings_evidence: 'legacy_stored_summary', played_at: null }
    expect(approvalProblem(legacy, selection)).not.toBe('')
    expect(approvalProblem(legacy, { ...selection, attest: true })).not.toBe('')
    expect(approvalProblem(legacy, { ...selection, attest: true, missingDateReason: 'Confirmed with players' })).toBe('')
  })
})
