import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

export type SeasonSummary = {
  id: number
  instance_name: string
  name: string
  status: string
  timezone: string
  started_at?: string
  registration_open: boolean
  player_count: number
  week_count: number
  game_variant: string
  legs_to_win: number
  games_per_week: number
  total_fixtures: number
}

export type PublicFixture = {
  id: number
  player_one: string
  player_two: string
  scheduled_at?: string
  game_variant?: string
  legs_to_win?: number
  result?: {
    player_one_legs: number
    player_two_legs: number
    player_one_average?: number
    player_two_average?: number
    winner_id: number
  }
}

export type PublicFixtureWeek = {
  week_number: number
  status: 'locked' | 'unlocked'
  reveal_at: string
  fixtures: PublicFixture[]
}

export type PublicFixturesResponse = {
  current_week: number
  weeks: PublicFixtureWeek[]
}

export type StandingRow = {
  player: string
  display_name: string
  played: number
  won: number
  lost: number
  legs_for: number
  legs_against: number
  leg_difference: number
  points: number
}

export type Player = {
  id: number
  display_name: string
  nickname?: string
  preferred_name: string
  admin_label: string
  registered_at?: string
}

export type AdminFixture = {
  id: number
  player_one: string
  player_two: string
  scheduled_at: string
  game_variant: string
  legs_to_win: number
  status: string
  result?: {
    player_one_legs: number
    player_two_legs: number
    player_one_average?: number
    player_two_average?: number
    winner_id: number
  }
}

export type AdminFixtureWeek = {
  week_number: number
  reveal_at: string
  fixtures: AdminFixture[]
}

export type AuditEntry = {
  id: number
  fixture_id: number
  fixture_label?: string
  action: string
  actor: string
  created_at: string
  old_result?: {
    player_one_legs: number
    player_two_legs: number
    player_one_average?: number
    player_two_average?: number
    winner_id: number
  }
  new_result?: {
    player_one_legs: number
    player_two_legs: number
    player_one_average?: number
    player_two_average?: number
    winner_id: number
  }
}

type LoginRequest = {
  username: string
  password: string
}

type UpdateSeasonRequest = {
  name: string
}

type UpdateSeasonConfigRequest = {
  game_variant: string
  legs_to_win: number
  games_per_week: number
}

export type GamesPerWeekPreset = {
  games_per_week: number
  week_count: number
}

export type SchedulePreview = {
  player_count: number
  game_variant: string
  legs_to_win: number
  games_per_week: number
  week_count: number
  total_fixtures: number
}

type RegisterRequest = {
  display_name: string
  nickname?: string
}

type ResultRequest = {
  fixtureId: number
  playerOneLegs: number
  playerTwoLegs: number
  playerOneAverage?: number
  playerTwoAverage?: number
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  })

  if (response.status === 204) {
    return undefined as T
  }

  const text = await response.text()
  const data = text ? JSON.parse(text) : null

  if (!response.ok) {
    const code = data?.error?.code ?? 'request_failed'
    const message = data?.error?.message ?? 'Request failed.'
    throw new ApiError(response.status, code, message)
  }

  return data as T
}

export function useSeasonSummary() {
  return useQuery({ queryKey: ['season'], queryFn: () => request<SeasonSummary>('/api/season') })
}

export function usePublicFixtures() {
  return useQuery({ queryKey: ['fixtures'], queryFn: () => request<PublicFixturesResponse>('/api/fixtures') })
}

export function useStandings() {
  return useQuery({
    queryKey: ['standings'],
    queryFn: async () => (await request<{ standings: StandingRow[] }>('/api/standings')).standings,
  })
}

export function useAdminPlayers() {
  return useQuery({
    queryKey: ['admin', 'players'],
    queryFn: async () => (await request<{ players: Player[] }>('/api/admin/players')).players,
    retry: false,
  })
}

export function useAdminFixtures(enabled: boolean) {
  return useQuery({
    queryKey: ['admin', 'fixtures'],
    queryFn: async () => (await request<{ weeks: AdminFixtureWeek[] }>('/api/admin/fixtures')).weeks,
    enabled,
  })
}

export function useAuditLog(enabled: boolean) {
  return useQuery({
    queryKey: ['admin', 'audit'],
    queryFn: async () => (await request<{ entries: AuditEntry[] }>('/api/admin/audit')).entries,
    enabled,
  })
}

export function useRegisterPlayer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: RegisterRequest) => request<Player>('/api/players/register', { method: 'POST', body: JSON.stringify(payload) }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['season'] })
      await queryClient.invalidateQueries({ queryKey: ['admin', 'players'] })
    },
  })
}

export function useAdminLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: LoginRequest) => request<{ authenticated: boolean; actor: string }>('/api/admin/login', { method: 'POST', body: JSON.stringify(payload) }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['admin'] })
    },
  })
}

export function useAdminLogout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => request<{ authenticated: boolean }>('/api/admin/logout', { method: 'POST' }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['admin'] })
    },
  })
}

export function useSeasonStart() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => request<SeasonSummary>('/api/admin/season/start', { method: 'POST' }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['season'] })
      await queryClient.invalidateQueries({ queryKey: ['fixtures'] })
      await queryClient.invalidateQueries({ queryKey: ['admin', 'fixtures'] })
      await queryClient.invalidateQueries({ queryKey: ['admin', 'players'] })
    },
  })
}

export function useUpdateSeason() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: UpdateSeasonRequest) => request<SeasonSummary>('/api/admin/season', { method: 'PUT', body: JSON.stringify(payload) }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['season'] })
    },
  })
}

export function useUpdateSeasonConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: UpdateSeasonConfigRequest) => request<SeasonSummary>('/api/admin/season/config', { method: 'PUT', body: JSON.stringify(payload) }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['season'] })
      await queryClient.invalidateQueries({ queryKey: ['admin', 'presets'] })
      await queryClient.invalidateQueries({ queryKey: ['admin', 'preview'] })
    },
  })
}

export function useGamesPerWeekPresets(enabled: boolean) {
  return useQuery({
    queryKey: ['admin', 'presets'],
    queryFn: async () => (await request<{ presets: GamesPerWeekPreset[] }>('/api/admin/season/presets')).presets,
    enabled,
  })
}

export function useSchedulePreview(enabled: boolean) {
  return useQuery({
    queryKey: ['admin', 'preview'],
    queryFn: () => request<SchedulePreview>('/api/admin/season/preview'),
    enabled,
  })
}

export function useDeletePlayer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (playerId: number) => request<void>(`/api/admin/players/${playerId}`, { method: 'DELETE' }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['admin', 'players'] })
      await queryClient.invalidateQueries({ queryKey: ['season'] })
    },
  })
}

export function useSaveResult() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (payload: ResultRequest) => {
      const body = JSON.stringify({
        player_one_legs: payload.playerOneLegs,
        player_two_legs: payload.playerTwoLegs,
        player_one_average: payload.playerOneAverage,
        player_two_average: payload.playerTwoAverage,
      })
      try {
        return await request(`/api/admin/fixtures/${payload.fixtureId}/result`, { method: 'POST', body })
      } catch (error) {
        if (error instanceof ApiError && error.code === 'result_exists') {
          return request(`/api/admin/fixtures/${payload.fixtureId}/result`, { method: 'PUT', body })
        }
        throw error
      }
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['standings'] })
      await queryClient.invalidateQueries({ queryKey: ['admin', 'fixtures'] })
      await queryClient.invalidateQueries({ queryKey: ['fixtures'] })
      await queryClient.invalidateQueries({ queryKey: ['admin', 'audit'] })
    },
  })
}

export function useUndoResult() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (fixtureId: number) => request<void>(`/api/admin/fixtures/${fixtureId}/result`, { method: 'DELETE' }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['standings'] })
      await queryClient.invalidateQueries({ queryKey: ['fixtures'] })
      await queryClient.invalidateQueries({ queryKey: ['admin', 'fixtures'] })
      await queryClient.invalidateQueries({ queryKey: ['admin', 'audit'] })
    },
  })
}

export function formatAverage(value?: number) {
	if (value === undefined) {
		return ''
	}
	return value.toFixed(1)
}

export function formatWhen(value?: string) {
  if (!value) {
    return ''
  }
  return new Intl.DateTimeFormat('en-GB', {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
    timeZone: 'Europe/London',
  }).format(new Date(value))
}
