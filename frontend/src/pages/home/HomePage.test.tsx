import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMockFetch, renderApp } from '../../test/helpers'

describe('Home page', () => {
  beforeEach(() => {
    const state = { authenticated: false, seasonStarted: false, seasonName: 'MVP Season' }
    vi.stubGlobal('fetch', createMockFetch(state))
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('renders live fixtures from the API', async () => {
    renderApp('/')

    expect(screen.getByRole('heading', { name: /fixtures with a little theatre/i })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /^admin$/i })).not.toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /week 1/i })).toBeInTheDocument()
    })

    expect(screen.getByRole('button', { name: /week 1/i })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText(/the asp vs the ferret/i)).toBeInTheDocument()
    expect(screen.getByText(/week 1 - 501 - first to 3 legs/i)).toBeInTheDocument()
    expect(screen.getByText(/arrange within the week/i)).toBeInTheDocument()
    expect(screen.getByText(/i knew you'd look vs nothing to see here/i)).toBeInTheDocument()
    expect(screen.getByText(/players registered before the season start action/i)).toBeInTheDocument()
    expect(screen.getByText(/cardiff office - darts league/i)).toBeInTheDocument()
    expect(await screen.findByText(/backend v0.0.6/i)).toBeInTheDocument()
    expect(screen.getByText(/frontend dev/i)).toBeInTheDocument()
    await waitFor(() => {
      expect(document.title).toBe('Cardiff Office - Darts League')
    })
  })

  it('groups unfinished unlocked fixtures by week and lets the user switch between weeks', async () => {
    renderApp('/')

    const weekOneButton = await screen.findByRole('button', { name: /week 1/i })
    const weekTwoButton = await screen.findByRole('button', { name: /week 2/i })

    expect(weekOneButton).toHaveAttribute('aria-expanded', 'true')
    expect(weekTwoButton).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByText(/the asp vs the ferret/i)).toBeInTheDocument()
    expect(screen.queryByText(/voltage vs snakebite/i)).not.toBeInTheDocument()

    fireEvent.click(weekTwoButton)

    expect(weekOneButton).toHaveAttribute('aria-expanded', 'false')
    expect(weekTwoButton).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText(/voltage vs snakebite/i)).toBeInTheDocument()
    expect(screen.queryByText(/the asp vs the ferret/i)).not.toBeInTheDocument()
  })
})
