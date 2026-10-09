import { act, cleanup, renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, useConfirmPendingResult, useRejectPendingResult } from './api'
import type { ConfirmPendingRequest } from './pendingReview'

afterEach(() => { cleanup(); vi.unstubAllGlobals() })
const keys = [['admin', 'pending-results'], ['admin', 'pending-results', 7], ['admin', 'division', 'premier', 'fixtures'], ['division', 'premier', 'fixtures'], ['division', 'premier', 'standings'], ['season']]
const payload: ConfirmPendingRequest = { season_id: 4, fixture_id: 90, mapping: { 'source-a': 11, 'source-b': 12 }, expected_result: null, replace: false, reason: '', attest_format: false, missing_date_reason: '' }
function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  keys.forEach(key => client.setQueryData(key, {}))
  return { client, wrapper: ({ children }: { readonly children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider> }
}

describe('pending approval HTTP adapter', () => {
  it('sends the exact confirm DTO with explicit null snapshot and invalidates all affected caches', async () => {
    const { client, wrapper } = setup()
    const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: 901, fixture_id: 90 })))
    vi.stubGlobal('fetch', fetch)
    const { result } = renderHook(useConfirmPendingResult, { wrapper })
    await act(() => result.current.mutateAsync({ pendingId: 7, payload }))
    expect(fetch).toHaveBeenCalledExactlyOnceWith('/api/admin/pending-results/7/confirm', expect.objectContaining({ credentials: 'include', method: 'POST', body: JSON.stringify(payload) }))
    for (const key of keys) expect(client.getQueryState(key)?.isInvalidated).toBe(true)
  })
  it('preserves the entire server-issued replacement snapshot including precision and null averages', async () => {
    const { wrapper } = setup()
    const snapshot = { id: 900, updated_at: '2026-06-15T11:00:00.123456Z', player_one_legs: 3, player_two_legs: 2, player_one_average: 54.125, player_two_average: null, winner_id: 12 }
    const replacement = { ...payload, expected_result: snapshot, replace: true, reason: 'Correction' }
    const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: 900, fixture_id: 90 })))
    vi.stubGlobal('fetch', fetch)
    const { result } = renderHook(useConfirmPendingResult, { wrapper })
    await act(() => result.current.mutateAsync({ pendingId: 7, payload: replacement }))
    expect(fetch).toHaveBeenCalledWith('/api/admin/pending-results/7/confirm', expect.objectContaining({ body: JSON.stringify(replacement) }))
  })
  it('retains conflict code without retrying or invalidating the reviewed snapshot', async () => {
    const { client, wrapper } = setup()
    const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: { code: 'approval_conflict', message: 'Changed' } }), { status: 409 }))
    vi.stubGlobal('fetch', fetch)
    const { result } = renderHook(useConfirmPendingResult, { wrapper })
    await act(async () => { await expect(result.current.mutateAsync({ pendingId: 7, payload })).rejects.toEqual(new ApiError(409, 'approval_conflict', 'Changed')) })
    expect(fetch).toHaveBeenCalledOnce()
    for (const key of keys) expect(client.getQueryState(key)?.isInvalidated).toBe(false)
  })
  it('sends rejection reason, accepts 204 and invalidates affected caches', async () => {
    const { client, wrapper } = setup()
    const fetch = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetch)
    const { result } = renderHook(useRejectPendingResult, { wrapper })
    await act(() => result.current.mutateAsync({ pendingId: 7, reason: 'Conflicting capture' }))
    expect(fetch).toHaveBeenCalledExactlyOnceWith('/api/admin/pending-results/7/reject', expect.objectContaining({ credentials: 'include', method: 'POST', body: '{"reason":"Conflicting capture"}' }))
    for (const key of keys) expect(client.getQueryState(key)?.isInvalidated).toBe(true)
  })
})
