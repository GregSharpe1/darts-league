import { createRoot } from 'react-dom/client'
import { MatchAnalysis } from '../../src/components/match'
import type { MatchData } from '../../src/components/match'
import { syntheticMatch, withoutGeometry } from './fixture'
import '../../src/index.css'
import './harness.css'

const mode = new URLSearchParams(location.search).get('state')
let match: MatchData = syntheticMatch
if (mode === 'unknown') match = withoutGeometry(match)
if (mode === 'summary') match = { ...match, detail: null, players: [
  { ...match.players[0], stats: null }, { ...match.players[1], stats: null },
] }
if (mode === 'long') match = { ...match, players: [
  { ...match.players[0], label: 'A'.repeat(80) }, { ...match.players[1], label: 'Casey Very Long Display Name '.repeat(2) },
] }
if (mode === 'partial') match = { ...match,
  players: [match.players[0], { ...match.players[1], legs_won: 2, stats: null }],
  detail: { coverage: 'partial', legs: [1, 2, 3, 4, 5].map(number => ({ number, completed: true,
    winner_id: number % 2 === 1 ? match.players[0].id : match.players[1].id,
    visits: [{ number: 12, player_id: number % 2 === 1 ? match.players[0].id : match.players[1].id,
      start_remaining: 16, end_remaining: 0, bust: false,
      throws: [{ number: 1, segment: { bed: 'double', number: 8 }, entry_type: 'unknown', position: null }],
    }],
  })) },
}

const root = document.getElementById('root')
if (!root) throw new TypeError('Missing harness root')
createRoot(root).render(<main className="match-harness">
  <p className="harness-label">Component harness / Synthetic test data</p>
  <h1>Match <span>analysis.</span></h1>
  <p className="harness-note">Fictitious identities. Recorded test positions only. No requests, mutations or hardware-accuracy claims.</p>
  {mode === 'loading' ? <MatchAnalysis status="loading" /> : mode === 'error' ?
    <MatchAnalysis status="error" onRetry={() => location.assign(location.pathname)} /> : <MatchAnalysis status="ready" match={match} />}
</main>)
