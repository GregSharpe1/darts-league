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

  it('gates admin tools behind login and reveals division setup controls after authentication', async () => {
    renderApp('/admin')

    expect(await screen.findByRole('heading', { name: /^login$/i })).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText(/username/i), { target: { value: 'admin' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /unlock admin tools/i }))

    expect(await screen.findByRole('heading', { name: /registered players/i })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /divisions/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /create divisions/i })).toBeInTheDocument()
    expect(screen.getByLabelText(/the freeze \(luke humphries\) division/i)).toBeInTheDocument()
  })

  it('lets the admin rename the league before the season starts', async () => {
    renderApp('/admin')

    fireEvent.change(screen.getByLabelText(/username/i), { target: { value: 'admin' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /unlock admin tools/i }))

    const leagueNameInput = await screen.findByDisplayValue(/mvp season/i)
    fireEvent.change(leagueNameInput, { target: { value: 'Cardiff Premier League' } })
    fireEvent.click(screen.getByRole('button', { name: /save config/i }))

    await waitFor(() => {
      expect(screen.getAllByText(/cardiff premier league/i).length).toBeGreaterThan(0)
    })
  })

  it('locks central admin editing after the season starts', async () => {
    renderApp('/admin')

    fireEvent.change(screen.getByLabelText(/username/i), { target: { value: 'admin' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /unlock admin tools/i }))

    await screen.findByRole('heading', { name: /registered players/i })
    fireEvent.click(screen.getByRole('button', { name: /start season/i }))

    await waitFor(() => {
      expect(screen.getByText(/registration is locked, division names are frozen/i)).toBeInTheDocument()
    })

    expect(screen.getByRole('button', { name: /start season/i })).toBeDisabled()
  })
})
