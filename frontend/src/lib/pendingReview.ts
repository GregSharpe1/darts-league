import type { AdminFixture, PendingResult } from './api'
import type { MatchData, MatchPlayer, MatchStats } from '../components/match/types'

export type ExpectedResult = {
  readonly id: number
  readonly updated_at: string
  readonly player_one_legs: number
  readonly player_two_legs: number
  readonly player_one_average?: number | null
  readonly player_two_average?: number | null
  readonly winner_id: number
}

export type SourcePlayer = {
  readonly match_player_id: string
  readonly account_id: string | null
  readonly display_name: string
  readonly legs_won: number
  readonly stats: MatchStats | null
}

export type PendingDetail = {
  readonly pending_result: PendingResult
  readonly source: string
  readonly digest: string
  readonly changed_import: boolean
  readonly settings_evidence: string
  readonly review_reason: string
  readonly played_at: string | null
  readonly players: readonly SourcePlayer[]
  readonly detail: MatchData['detail']
  readonly original_payload: Readonly<Record<string, unknown>>
  readonly season_id: number | null
  readonly fixture_id: number | null
  readonly result_id: number | null
  readonly source_active: boolean
  readonly mapping: Readonly<Record<string, number>> | null
  readonly approval: Readonly<Record<string, unknown>> | null
}

export type ConfirmPendingRequest = {
  readonly season_id: number
  readonly fixture_id: number
  readonly mapping: Readonly<Record<string, number>>
  readonly expected_result: ExpectedResult | null
  readonly replace: boolean
  readonly reason: string
  readonly attest_format: boolean
  readonly missing_date_reason: string
}

export function reviewBlocked(detail: PendingDetail) {
  return detail.pending_result.status === 'review_blocked' || Boolean(detail.review_reason)
}

export function toMatchData(detail: PendingDetail): MatchData | null {
  if (reviewBlocked(detail) || detail.settings_evidence !== 'source_reported' || detail.players.length !== 2) return null
  const [one, two] = detail.players
  if (!one || !two || one.match_player_id === two.match_player_id) return null
  const player = (source: SourcePlayer): MatchPlayer => ({ id: source.match_player_id, label: source.display_name, legs_won: source.legs_won, stats: source.stats })
  return { players: [player(one), player(two)], playedAt: detail.played_at, detail: detail.detail, synthetic: false }
}

export type ReviewSelection = {
  readonly seasonId: number
  readonly playerIds: readonly number[]
  readonly fixture: AdminFixture | undefined
  readonly replace: boolean
  readonly reason: string
  readonly attest: boolean
  readonly missingDateReason: string
}

export function approvalProblem(detail: PendingDetail, selection: ReviewSelection): string {
  const { fixture, playerIds } = selection
  if (reviewBlocked(detail)) return 'Source conflict: approval is blocked. Reject this import or resolve it at the source.'
  if (detail.pending_result.status !== 'pending') return 'This import is no longer pending. Reload the inbox.'
  if (detail.players.length !== 2 || !detail.players[0].match_player_id || !detail.players[1].match_player_id || detail.players[0].match_player_id === detail.players[1].match_player_id) return 'Two distinct source players are required.'
  const scores = detail.players.map(player => player.legs_won)
  if (!scores.every(Number.isInteger) || !((scores[0] === 3 && scores[1] >= 0 && scores[1] < 3) || (scores[1] === 3 && scores[0] >= 0 && scores[0] < 3))) return 'Source score must be 3-0, 3-1 or 3-2.'
  if (!selection.seasonId) return 'Choose the current season explicitly.'
  if (playerIds.length !== 2 || !playerIds.every(id => id > 0) || playerIds[0] === playerIds[1]) return 'Map both source players to distinct league players.'
  if (!fixture || ![fixture.player_one_id, fixture.player_two_id].every(id => id !== undefined && playerIds.includes(id))) return 'Choose a fixture for this exact player pair.'
  if (fixture.game_variant !== '501' || fixture.legs_to_win !== 3) return 'The selected fixture must be 501, first to 3.'
  if (fixture.expected_result === undefined || (fixture.result && !fixture.expected_result)) return 'Expected result snapshot is unavailable. Reload before approving.'
  if (fixture.expected_result && (!fixture.expected_result.id || !fixture.expected_result.updated_at)) return 'Expected result snapshot is incomplete. Reload before approving.'
  if ((fixture.expected_result || detail.changed_import) && (!selection.replace || !selection.reason.trim())) return 'Replacement requires explicit consent and a reason.'
  if (detail.settings_evidence !== 'source_reported' && !selection.attest) return 'Attest the legacy match format before approving.'
  if (!detail.played_at && !selection.missingDateReason.trim()) return 'Explain how the fixture was verified without a source date.'
  return ''
}
