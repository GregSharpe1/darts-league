export type Provenance = 'automatic' | 'manual' | 'unknown'
export type Segment =
  | { readonly bed: 'miss'; readonly number: 0 }
  | { readonly bed: 'single' | 'double' | 'triple'; readonly number: number }
  | { readonly bed: 'outer_bull' | 'inner_bull'; readonly number: 25 }

export interface Position {
  readonly x: number
  readonly y: number
  readonly units: string | null
  readonly origin: string | null
  readonly axis_orientation: string | null
  readonly provenance: Provenance
}

export interface MatchThrow {
  readonly number: number
  readonly segment: Segment
  readonly entry_type: Provenance
  readonly position: Position | null
}

export interface MatchVisit {
  readonly number: number
  readonly player_id: string
  readonly start_remaining: number
  readonly end_remaining: number
  readonly bust: boolean
  readonly throws: readonly MatchThrow[]
}

export interface MatchLeg {
  readonly number: number
  readonly completed: boolean
  readonly winner_id: string | null
  readonly visits: readonly MatchVisit[]
}

export interface MatchStats {
  readonly first_nine_average?: number | null
  readonly average_until_170?: number | null
  readonly highest_finish?: number | null
  readonly total_180?: number | null
  readonly less_60?: number | null
  readonly plus_60?: number | null
  readonly plus_100?: number | null
  readonly plus_140?: number | null
  readonly plus_170?: number | null
  readonly match_average: number | null
  readonly points_scored: number | null
  readonly darts_thrown: number | null
  readonly checkout_hits: number | null
  readonly checkout_attempts: number | null
}

export interface MatchPlayer {
  readonly id: string
  readonly label: string
  readonly legs_won: number
  readonly stats: MatchStats | null
}

export interface MatchData {
  readonly players: readonly [MatchPlayer, MatchPlayer]
  readonly playedAt: string | null
  readonly detail: {
    readonly coverage: 'partial' | 'complete'
    readonly legs: readonly MatchLeg[]
  } | null
  readonly synthetic: boolean
}

export type MatchAnalysisProps =
  | { readonly status: 'loading' }
  | { readonly status: 'error'; readonly onRetry?: () => void }
  | { readonly status: 'ready'; readonly match: MatchData }
