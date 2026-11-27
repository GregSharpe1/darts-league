import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
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

  it('gives the home page separate league and tournament destinations', () => {
    renderApp('/')

    const toolbar = within(screen.getByRole('navigation', { name: 'Primary' }))
    expect(toolbar.getAllByRole('link').map(link => link.textContent)).toEqual(['Records', 'Player stats', 'Admin'])
    expect(toolbar.getByRole('link', { name: 'Admin' })).toHaveAttribute('href', '/admin')
    expect(toolbar.getByRole('link', { name: 'Records' })).toHaveAttribute('href', '/players/records')
    expect(toolbar.getByRole('link', { name: 'Player stats' })).toHaveAttribute('href', '/players')
    const league = screen.getByRole('region', { name: 'League matches' })
    const tournaments = screen.getByRole('region', { name: 'Tournaments' })
    expect(within(league).getByRole('link', { name: 'Browse divisions' })).toHaveAttribute('href', '/divisions')
    expect(within(tournaments).getByRole('link', { name: 'Browse tournaments' })).toHaveAttribute('href', '/tournaments')
  })

  it('shows the current league, open tournament sign-ups, and a published bracket', async () => {
    vi.stubGlobal('fetch', createMockFetch({
      authenticated: false,
      seasonStarted: false,
      seasonName: 'Cardiff Spring League',
      tournaments: [
        { slug: 'night-cup', name: 'Night Cup', starts_on: '2027-02-06', registration_open: true, started: false },
        { slug: 'city-cup', name: 'City Cup', starts_on: '2027-02-01', registration_open: false, started: true },
      ],
    }))
    renderApp('/')

    const league = within(screen.getByRole('region', { name: 'League matches' }))
    const tournaments = within(screen.getByRole('region', { name: 'Tournaments' }))
    expect(await league.findByRole('heading', { name: 'Cardiff Spring League' })).toBeInTheDocument()
    expect(league.getByRole('link', { name: 'Register for the league' })).toHaveAttribute('href', '/register')
    expect(await tournaments.findByRole('link', { name: 'Sign up for Night Cup' })).toHaveAttribute('href', '/tournaments/night-cup')
    expect(tournaments.getByRole('link', { name: 'City Cup' })).toHaveAttribute('href', '/tournaments/city-cup')
    expect(tournaments.getByText(/tournament started/i)).toBeInTheDocument()
  })

  it('renders the public divisions page', async () => {
    renderApp('/divisions')

    const navigation = within(screen.getByRole('navigation', { name: 'Primary' }))
    expect(navigation.queryByRole('link', { name: /^Divisions$/ })).not.toBeInTheDocument()
    expect(navigation.getByRole('link', { name: /^Tournaments$/ })).toHaveAttribute('href', '/tournaments')
    expect(screen.getByRole('heading', { name: 'League divisions', level: 1 })).toBeInTheDocument()
    expect(screen.getByText(/admins assign players to divisions after they register for the league/i)).toBeInTheDocument()
    expect(await screen.findByText(/premier division/i)).toBeInTheDocument()
    expect(screen.getByText(/challenger division/i)).toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: /view fixtures/i }).length).toBe(2)
    expect(screen.getByText('Registered players')).toBeInTheDocument()
    expect(await screen.findByText('Dates will appear once the league schedule is set.')).toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: 'Register' })).toHaveLength(2)
    expect(screen.queryByRole('link', { name: 'Join the waitlist' })).not.toBeInTheDocument()
    expect(await screen.findByText(/backend v0.0.6/i)).toBeInTheDocument()
    expect(screen.getByText(/frontend dev/i)).toBeInTheDocument()
    await waitFor(() => {
      expect(document.title).toBe('Cardiff Office - Darts League')
    })
  })

  it('collapses only recorded weeks by default and lets the user toggle them', async () => {
    renderApp('/divisions/premier')

    const results = within(await screen.findByRole('region', { name: 'Recorded results' }))
    const weekOneButton = results.getByRole('button', { name: 'Week 1' })
    const weekTwoButton = results.getByRole('button', { name: 'Week 2' })
    const navigation = within(screen.getByRole('navigation', { name: 'Primary' }))
    expect(navigation.getByRole('link', { name: /^All Divisions$/ })).toHaveAttribute('href', '/divisions')
    expect(await navigation.findByRole('link', { name: /^Register$/ })).toHaveAttribute('href', '/register')
    expect(navigation.getByRole('link', { name: /^Tournaments$/ })).toHaveAttribute('href', '/tournaments')

    expect(weekOneButton).toHaveAttribute('aria-expanded', 'false')
    expect(weekTwoButton).toHaveAttribute('aria-expanded', 'false')
    expect(results.queryByRole('link')).not.toBeInTheDocument()
    expect(screen.getByText(/the asp.*the ferret/i)).toBeVisible()
    expect(screen.getByText(/voltage.*snakebite/i)).toBeVisible()
    expect(screen.getByText(/nothing to see here/i)).toBeInTheDocument()

    fireEvent.click(weekOneButton)
    expect(weekOneButton).toHaveAttribute('aria-expanded', 'true')
    expect(results.getByText(/the freeze.*bully boy/i)).toBeVisible()
    fireEvent.click(weekOneButton)
    expect(weekOneButton).toHaveAttribute('aria-expanded', 'false')
    expect(results.queryByRole('link')).not.toBeInTheDocument()
  })

  it.each([false, true])('shows previous-league boards after every match is played (closed: %s)', async (seasonCompleted) => {
    vi.stubGlobal('fetch', createMockFetch({ authenticated: false, seasonStarted: true, seasonCompleted, remainingFixtures: 0, seasonName: 'Finished League' }))
    renderApp('/divisions')

    expect(await screen.findByRole('heading', { name: "View the previous league's scores here" })).toBeInTheDocument()
    expect(screen.queryByText('Monday 09:00 unlocks')).not.toBeInTheDocument()
    expect(screen.queryByText('Final standings')).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Public division boards' })).not.toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: 'View standings' })).toHaveLength(2)
  })

  it('shows the final league week, catch-up deadline and simulation rules while fixtures remain', async () => {
    vi.stubGlobal('fetch', createMockFetch({ authenticated: false, seasonStarted: true, remainingFixtures: 1, seasonName: 'Live League' }))
    renderApp('/divisions')

    expect(await screen.findByText('season live across 2 divisions.')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Public division boards' })).toBeInTheDocument()
    expect(screen.getByText('Monday 09:00 unlocks')).toBeInTheDocument()

    const deadline = within(screen.getByRole('region', { name: 'Final week and catch-up deadline' }))
    expect(deadline.getByText('Week 3 starts')).toBeInTheDocument()
    expect(deadline.getByText(/6 apr.*09:00 uk time/i)).toHaveAttribute('datetime', '2026-04-06T08:00:00Z')
    expect(deadline.getByText(/20 apr.*09:00 uk time/i)).toHaveAttribute('datetime', '2026-04-20T08:00:00Z')
    expect(deadline.getByText(/one extra week/i)).toBeInTheDocument()
    expect(deadline.getByText(/represent both players.*drops one level per missed fixture/i)).toBeInTheDocument()
  })

  it('filters the fixtures and opens every matching recorded week while searching', async () => {
    renderApp('/divisions/premier')

    const search = await screen.findByRole('combobox', { name: /find a player's games/i })
    await screen.findByRole('region', { name: 'Recorded results' })
    fireEvent.focus(search)
    fireEvent.change(search, { target: { value: 'luke' } })
    fireEvent.click(await screen.findByRole('option', { name: 'The Freeze (Luke Humphries)' }))

    expect(search).toHaveValue('The Freeze (Luke Humphries)')
    const results = within(screen.getByRole('region', { name: 'Recorded results' }))
    expect(results.getByRole('button', { name: 'Week 1' })).toHaveAttribute('aria-expanded', 'true')
    expect(results.getByRole('button', { name: 'Week 2' })).toHaveAttribute('aria-expanded', 'true')
    expect(results.getAllByRole('link')).toHaveLength(2)
    expect(screen.queryByText(/the asp|voltage|nothing to see here/i)).not.toBeInTheDocument()

    fireEvent.change(search, { target: { value: 'snakebite' } })
    expect(screen.getByText(/voltage.*snakebite/i)).toBeVisible()
    expect(screen.queryByRole('region', { name: 'Recorded results' })).not.toBeInTheDocument()
    expect(screen.queryByText(/the asp/i)).not.toBeInTheDocument()

    fireEvent.change(search, { target: { value: '' } })
    const resetResults = within(screen.getByRole('region', { name: 'Recorded results' }))
    expect(resetResults.getByRole('button', { name: 'Week 1' })).toHaveAttribute('aria-expanded', 'false')
    expect(resetResults.getByRole('button', { name: 'Week 2' })).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByText(/the asp.*the ferret/i)).toBeVisible()
  })
})
