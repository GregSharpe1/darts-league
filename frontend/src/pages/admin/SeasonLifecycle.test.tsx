import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { createMockFetch, renderApp } from '../../test/helpers'

afterEach(() => { cleanup(); vi.unstubAllGlobals() })

it('requires every result before offering closure', async () => {
  vi.stubGlobal('fetch', createMockFetch({ authenticated: true, seasonStarted: true, seasonName: 'Finals', remainingFixtures: 1 }))
  renderApp('/admin')
  expect(await screen.findByRole('button', { name: 'Close league' })).toBeDisabled()
  expect(screen.getByText(/1 match.*remaining/i)).toBeInTheDocument()
})

it('confirms closure then opens a fresh registration season with refreshed boards', async () => {
  const state = { authenticated: true, seasonStarted: true, seasonName: 'Finals', remainingFixtures: 0 }
  const fetch = createMockFetch(state)
  const confirm = vi.fn().mockReturnValue(false)
  vi.stubGlobal('fetch', fetch)
  vi.stubGlobal('confirm', confirm)
  renderApp('/admin')
  fireEvent.click(await screen.findByRole('button', { name: 'Close league' }))
  expect(fetch.mock.calls.some(([path]) => path === '/api/admin/season/close')).toBe(false)
  confirm.mockReturnValue(true)
  fireEvent.click(screen.getByRole('button', { name: 'Close league' }))
  expect(await screen.findByText('League completed')).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Save config' })).not.toBeInTheDocument()
  fireEvent.change(screen.getByLabelText('Next league name'), { target: { value: 'New league' } })
  confirm.mockReturnValue(false)
  fireEvent.click(screen.getByRole('button', { name: 'Open next season registration' }))
  expect(fetch.mock.calls.some(([path]) => path === '/api/admin/season/next')).toBe(false)
  confirm.mockReturnValue(true)
  fireEvent.click(screen.getByRole('button', { name: 'Open next season registration' }))
  await waitFor(() => expect(screen.queryByText('League completed')).not.toBeInTheDocument())
  expect(await screen.findByRole('button', { name: 'Start season' })).toBeInTheDocument()
  await waitFor(() => expect(screen.queryByRole('link', { name: 'Open scoring page' })).not.toBeInTheDocument())
  expect(fetch).toHaveBeenCalledWith('/api/admin/season/next', expect.objectContaining({ body: JSON.stringify({ season_id: 1, name: 'New league' }) }))
})

it('shows server errors without claiming completion', async () => {
  vi.stubGlobal('fetch', createMockFetch({ authenticated: true, seasonStarted: true, seasonName: 'Finals', remainingFixtures: 0, lifecycleError: true }))
  vi.stubGlobal('confirm', vi.fn().mockReturnValue(true))
  renderApp('/admin')
  fireEvent.click(await screen.findByRole('button', { name: 'Close league' }))
  expect(await screen.findByText('Results are still missing.')).toBeInTheDocument()
  expect(screen.queryByText('League completed')).not.toBeInTheDocument()
})

it('keeps completed scores and audits readable without editing controls', async () => {
  vi.stubGlobal('fetch', createMockFetch({ authenticated: true, seasonStarted: true, seasonCompleted: true, seasonName: 'Finals', remainingFixtures: 0 }))
  renderApp('/admin/divisions/premier')
  expect(await screen.findByText(/league completed/i)).toBeInTheDocument()
  expect(await screen.findByLabelText('The Freeze (Luke Humphries) legs')).toHaveValue(3)
  expect(screen.getByLabelText('The Freeze (Luke Humphries) legs')).toHaveAttribute('readonly')
  expect(screen.queryByRole('button', { name: 'Save score' })).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Undo result' })).not.toBeInTheDocument()
  expect(screen.getByRole('heading', { name: 'Audit log' })).toBeInTheDocument()
})

it('keeps public completed results visible without exposing locked scores', async () => {
  vi.stubGlobal('fetch', createMockFetch({ authenticated: false, seasonStarted: true, seasonCompleted: true, seasonName: 'Finals' }))
  renderApp('/divisions/premier')
  expect(await screen.findByRole('heading', { name: 'Final results' })).toBeInTheDocument()
  expect(await screen.findByText('The Freeze (Luke Humphries) 3-1 Bully Boy (Michael Smith)')).toBeInTheDocument()
  expect(screen.getByText('Averages: 96.4 / 89.3')).toBeInTheDocument()
  expect(screen.queryByRole('heading', { name: 'Games left to play' })).not.toBeInTheDocument()
})
