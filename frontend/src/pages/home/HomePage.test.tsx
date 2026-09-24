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

  it('renders the public division landing page', async () => {
    renderApp('/')

    expect(screen.getByRole('heading', { name: /pick a division and follow the board/i })).toBeInTheDocument()
    expect(await screen.findByText(/premier division/i)).toBeInTheDocument()
    expect(screen.getByText(/challenger division/i)).toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: /view fixtures/i }).length).toBe(2)
    expect(screen.getByText(/registered players waiting for assignment/i)).toBeInTheDocument()
    expect(await screen.findByText(/backend v0.0.6/i)).toBeInTheDocument()
    expect(screen.getByText(/frontend dev/i)).toBeInTheDocument()
    await waitFor(() => {
      expect(document.title).toBe('Cardiff Office - Darts League')
    })
  })

  it('shows a division fixture board and lets the user switch weeks', async () => {
    renderApp('/divisions/premier')

    const weekOneButton = await screen.findByRole('button', { name: /week 1/i })
    const weekTwoButton = await screen.findByRole('button', { name: /week 2/i })

    expect(weekOneButton).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText(/the asp/i)).toBeInTheDocument()
    expect(screen.queryByText(/voltage/i)).not.toBeInTheDocument()

    fireEvent.click(weekTwoButton)

    expect(weekOneButton).toHaveAttribute('aria-expanded', 'false')
    expect(weekTwoButton).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText(/voltage/i)).toBeInTheDocument()
  })

  it.each([false, true])('shows previous-league boards after every match is played (closed: %s)', async (seasonCompleted) => {
    vi.stubGlobal('fetch', createMockFetch({ authenticated: false, seasonStarted: true, seasonCompleted, remainingFixtures: 0, seasonName: 'Finished League' }))
    renderApp('/')

    expect(await screen.findByRole('heading', { name: "View the previous league's scores here" })).toBeInTheDocument()
    expect(screen.queryByText('Monday 09:00 unlocks')).not.toBeInTheDocument()
    expect(screen.queryByText('Final standings')).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Public division boards' })).not.toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: 'View standings' })).toHaveLength(2)
  })

  it('keeps the weekly unlock notice while matches remain', async () => {
    vi.stubGlobal('fetch', createMockFetch({ authenticated: false, seasonStarted: true, remainingFixtures: 1, seasonName: 'Live League' }))
    renderApp('/')

    expect(await screen.findByText('season live across 2 divisions.')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Public division boards' })).toBeInTheDocument()
    expect(screen.getByText('Monday 09:00 unlocks')).toBeInTheDocument()
  })

  it('lets a player search their remaining unplayed fixtures inside a division', async () => {
    renderApp('/divisions/premier')

    const search = await screen.findByLabelText(/find remaining games in this division/i)
    fireEvent.focus(search)
    await waitFor(() => {
      expect(screen.getByRole('option', { name: /snakebite/i })).toBeInTheDocument()
    })

    fireEvent.change(search, { target: { value: 'sna' } })
    fireEvent.click(screen.getByRole('option', { name: /snakebite/i }))

    expect(search).toHaveValue('Snakebite')
    expect(screen.getByRole('heading', { name: /1 remaining match/i })).toBeInTheDocument()
    expect(screen.getByText(/voltage/i)).toBeInTheDocument()
  })
})
