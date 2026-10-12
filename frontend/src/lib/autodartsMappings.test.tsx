import { act, cleanup, renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { afterEach, expect, it, vi } from 'vitest'
import { ApiError, useSaveAutodartsMapping } from './api'

afterEach(() => { cleanup(); vi.unstubAllGlobals() })

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  const keys = [['admin', 'autodarts-mappings', 4], ['admin', 'pending-results', 7]]
  keys.forEach(key => client.setQueryData(key, {}))
  const wrapper = ({ children }: { readonly children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>
  return { client, keys, ...renderHook(useSaveAutodartsMapping, { wrapper }) }
}

it.each(['gsharpe', null])('sends season-scoped mapping mutation %s and invalidates suggestions', async username => {
  const fetch = vi.fn().mockResolvedValue(username === null ? new Response(null, { status: 204 }) : new Response(JSON.stringify({ player_id: 11, username })))
  vi.stubGlobal('fetch', fetch)
  const { result, client, keys } = setup()
  await act(() => result.current.mutateAsync({ seasonId: 4, playerId: 11, username }))
  expect(fetch).toHaveBeenCalledExactlyOnceWith('/api/admin/autodarts-mappings/11', expect.objectContaining({ credentials: 'include', method: username === null ? 'DELETE' : 'PUT', body: JSON.stringify({ season_id: 4, ...(username === null ? {} : { username }) }) }))
  for (const key of keys) expect(client.getQueryState(key)?.isInvalidated).toBe(true)
})

it.each([401, 409])('retains mapping API error %s without retrying or invalidating review', async status => {
  const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: { code: 'mapping_conflict', message: 'Mapping was not saved.' } }), { status }))
  vi.stubGlobal('fetch', fetch)
  const { result, client, keys } = setup()
  await act(async () => { await expect(result.current.mutateAsync({ seasonId: 4, playerId: 11, username: 'gsharpe' })).rejects.toEqual(new ApiError(status, 'mapping_conflict', 'Mapping was not saved.')) })
  expect(fetch).toHaveBeenCalledOnce()
  for (const key of keys) expect(client.getQueryState(key)?.isInvalidated).toBe(false)
})
