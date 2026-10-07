import reference from '../../docs/autodarts-fixtures/randomized.json' with { type: 'json' }

const sourcePlayers = reference.players.map(player => {
  const stats = reference.matchStats.find(stats => stats.playerId === player.id)
  return { match_player_id: player.id, account_id: null, display_name: player.name, legs_won: player.legsWon,
    stats: { match_average: stats.average, points_scored: player.legsWon * 501, darts_thrown: stats.dartsThrown, checkout_hits: stats.checkoutsHit, checkout_attempts: stats.checkouts, first_nine_average: player.legsWon ? 167 : null, total_180: player.legsWon ? 6 : 0, highest_finish: player.legsWon ? 141 : null } }
})
const detail = { coverage: 'complete', legs: [1, 2, 3].map(number => {
  const darts = reference.darts.filter(dart => dart.leg === number)
  const remaining = new Map(reference.players.map(player => [player.id, 501]))
  return { number, completed: true, winner_id: reference.winnerId, visits: [...new Set(darts.map(dart => dart.visit))].map(visit => {
    const source = darts.filter(dart => dart.visit === visit)
    const player_id = source[0].playerId
    const start_remaining = remaining.get(player_id)
    const points = source.reduce((sum, dart) => sum + (dart.segment.startsWith('M') ? 0 : Number(dart.segment.slice(1)) * (dart.segment.startsWith('T') ? 3 : 2)), 0)
    remaining.set(player_id, start_remaining - points)
    return { number: visit + 1, player_id, start_remaining, end_remaining: start_remaining - points, bust: false, throws: source.map(dart => ({ number: dart.dart + 1,
      segment: { bed: dart.segment.startsWith('M') ? 'miss' : dart.segment.startsWith('T') ? 'triple' : 'double', number: dart.segment.startsWith('M') ? 0 : Number(dart.segment.slice(1)) }, entry_type: 'manual',
      position: { x: dart.x, y: dart.y, units: 'board-radius', origin: 'bull', axis_orientation: 'x-right-y-up', provenance: 'manual' } })) }
  }) }
}) }

export const players = ['Morgan Ember', 'Casey Vale', 'Rowan Ash'].map((name, index) => ({ id: index + 11, display_name: name, preferred_name: name, admin_label: name, status: 'assigned', division_id: 1 }))
export const divisions = [{ id: 1, name: 'Premier Division', slug: 'premier', position: 1 }]
export const pending = { id: 7, external_match_id: 'demo-match', player_one_name: 'Morgan Ember', player_two_name: 'Casey Vale', player_one_legs: 3, player_two_legs: 0, received_at: '2026-06-15T10:00:00Z', status: 'pending' }
export const fixture = { id: 90, player_one_id: 12, player_two_id: 11, player_one: 'Casey Vale', player_two: 'Morgan Ember', scheduled_at: '2026-06-15T09:00:00Z', game_variant: '501', legs_to_win: 3, status: 'unplayed', expected_result: null }
export const expectedResult = { id: 900, updated_at: '2026-06-15T11:00:00.123456Z', player_one_legs: 3, player_two_legs: 2, player_one_average: 54.125, player_two_average: null, winner_id: 12 }
export const source = {
  pending_result: pending, source: 'autodarts / synthetic QA fixture', digest: 'demo-digest', changed_import: false, settings_evidence: 'source_reported', review_reason: '', played_at: '2026-06-15T09:30:00Z',
  players: sourcePlayers,
  detail, original_payload: {}, season_id: null, fixture_id: null, result_id: null, source_active: false, mapping: null, approval: null,
}
export const season = { id: 4, instance_name: 'Cardiff Darts League', name: 'Summer League', status: 'started', timezone: 'Europe/London', registration_open: false, season_started: true, admin_locked: true, can_start_season: false, can_close_season: false, can_create_next_season: false, remaining_fixtures: 1, can_edit_settings: false, can_edit_division_channel: false, can_edit_divisions: false, can_assign_players: false, player_count: 3, week_count: 1, game_variant: '501', legs_to_win: 3, games_per_week: 1, total_fixtures: 1, division_count: 1, assigned_count: 3, waitlist_count: 0 }

export async function mockReview(page, overrides = {}) {
  const state = { source: structuredClone(source), fixture: structuredClone(fixture), pending: [pending, { ...pending, id: 8, player_one_name: 'Other source' }], detailStatus: 200, conflict: false, authenticated: true, ...overrides }
  const requests = []
  await page.route('**/api/**', async route => {
    const request = route.request()
    const url = new URL(request.url()).pathname
    requests.push({ url, method: request.method(), body: request.postDataJSON() })
    if (url.startsWith('/api/admin/') && !state.authenticated) return route.fulfill({ status: 401, json: { error: { code: 'unauthorized', message: 'Log in to continue.' } } })
    if (url === '/api/season') return route.fulfill({ json: season })
    if (url === '/api/version') return route.fulfill({ json: { version: 'pending-review-test' } })
    if (url.endsWith('/divisions')) return route.fulfill({ json: { divisions } })
    if (url.endsWith('/players')) return route.fulfill({ json: { players } })
    if (url.endsWith('/fixtures')) return route.fulfill({ json: { weeks: [{ week_number: 1, reveal_at: '2026-06-15T09:00:00Z', fixtures: [state.fixture] }] } })
    if (url === '/api/admin/pending-results') return route.fulfill({ json: { pending_results: state.pending } })
    if (/\/pending-results\/\d+$/.test(url)) return route.fulfill({ status: state.detailStatus, json: state.detailStatus === 200 ? { ...state.source, pending_result: { ...state.source.pending_result, id: Number(url.split('/').at(-1)) } } : { error: { code: 'unavailable', message: 'Source temporarily unavailable.' } } })
    if (url.endsWith('/confirm')) {
      if (state.conflict) return route.fulfill({ status: 409, json: { error: { code: 'approval_conflict', message: 'The result changed while you reviewed it.' } } })
      state.pending = []
      return route.fulfill({ json: { id: 901, fixture_id: state.fixture.id, player_one_legs: 0, player_two_legs: 3, winner_id: 11 } })
    }
    if (url.endsWith('/reject')) { state.pending = []; return route.fulfill({ status: 204 }) }
    if (url.endsWith('/poll')) return route.fulfill({ status: 204 })
    return route.fulfill({ status: 404, json: { error: { code: 'unexpected', message: 'Unexpected test endpoint.' } } })
  })
  return { state, requests }
}

export async function mapReview(page) {
  await page.getByLabel('Season', { exact: true }).selectOption('4')
  await page.getByLabel('League', { exact: true }).selectOption('1')
  await page.getByLabel('Player 1:', { exact: false }).selectOption('11')
  await page.getByLabel('Player 2:', { exact: false }).selectOption('12')
  await page.getByLabel('Fixture', { exact: true }).selectOption('90')
}
