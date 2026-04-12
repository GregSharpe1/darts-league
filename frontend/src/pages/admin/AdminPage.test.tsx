import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMockFetch, renderApp } from '../../test/helpers'

describe('Admin page', () => {
  beforeEach(() => {
    const state = { authenticated: false, seasonStarted: false, seasonName: 'MVP Season' }
    vi.stubGlobal('fetch', createMockFetch(state))
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('gates admin tools behind login and reveals live admin data after authentication', async () => {
    renderApp('/admin')

    expect(await screen.findByRole('heading', { name: /^login$/i })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /^admin$/i })).not.toBeInTheDocument()

    fireEvent.change(screen.getByLabelText(/username/i), { target: { value: 'admin' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /unlock admin tools/i }))

    expect(await screen.findByRole('heading', { name: /registered players/i })).toBeInTheDocument()
    expect(await screen.findByText(/the freeze \(luke humphries\)/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/the freeze \(luke humphries\) legs/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/the freeze \(luke humphries\) average/i)).toHaveValue('96.4')
    expect(screen.getByRole('button', { name: /undo result/i })).toBeInTheDocument()

    fireEvent.click(await screen.findByRole('button', { name: /audit trail/i }))
    expect(screen.getByText(/result edited/i)).toBeInTheDocument()
  })

  it('prevents saving an invalid scoreline for the fixture format', async () => {
    renderApp('/admin')

    fireEvent.change(screen.getByLabelText(/username/i), { target: { value: 'admin' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /unlock admin tools/i }))

    await screen.findByRole('heading', { name: /registered players/i })

    fireEvent.change(await screen.findByLabelText(/The Freeze \(Luke Humphries\) legs/i), { target: { value: '1' } })
    fireEvent.change(screen.getByLabelText(/bully boy \(michael smith\) legs/i), { target: { value: '1' } })

    expect(screen.getByText(/valid scores: 3-0 to 3-2/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /save score/i })).toBeDisabled()
  })

  it('locks admin roster controls after the season starts', async () => {
    renderApp('/admin')

    fireEvent.change(screen.getByLabelText(/username/i), { target: { value: 'admin' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /unlock admin tools/i }))

    await screen.findByRole('heading', { name: /registered players/i })
    expect(screen.getAllByRole('button', { name: /^delete$/i }).length).toBeGreaterThan(0)

    // Click "Start season" to open the confirmation dialog.
    fireEvent.click(screen.getAllByRole('button', { name: /start season/i })[0])

    // Wait for the confirmation dialog to appear with the preview data.
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /start season\?/i })).toBeInTheDocument()
    })

    // Click "Start Season" in the confirmation dialog to actually start.
    const confirmButtons = screen.getAllByRole('button', { name: /^start season$/i })
    fireEvent.click(confirmButtons[confirmButtons.length - 1])

    await waitFor(() => {
      expect(screen.getByText(/registration is locked and player deletion is now disabled/i)).toBeInTheDocument()
    })
    await waitFor(() => {
      expect(screen.queryAllByRole('link', { name: /register/i })).toHaveLength(0)
    })

    expect(screen.getAllByText(/roster locked/i).length).toBeGreaterThan(0)
    expect(screen.getAllByRole('button', { name: /start season/i })[0]).toBeDisabled()
  })

  it('lets the admin rename the league before the season starts', async () => {
    renderApp('/admin')

    fireEvent.change(screen.getByLabelText(/username/i), { target: { value: 'admin' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /unlock admin tools/i }))

    expect(await screen.findByRole('heading', { name: /league settings/i })).toBeInTheDocument()

    const leagueNameInput = await screen.findByDisplayValue(/mvp season/i)
    fireEvent.change(leagueNameInput, { target: { value: 'Cardiff Premier League' } })
    fireEvent.click(screen.getByRole('button', { name: /save config/i }))

    await waitFor(() => {
      expect(screen.getAllByText(/cardiff premier league/i).length).toBeGreaterThan(0)
    })
  })
})
