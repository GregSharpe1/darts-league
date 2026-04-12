import { cleanup, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMockFetch, renderApp } from '../../test/helpers'

describe('Standings page', () => {
  beforeEach(() => {
    const state = { authenticated: false, seasonStarted: false, seasonName: 'MVP Season' }
    vi.stubGlobal('fetch', createMockFetch(state))
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('renders standings with LW and LL column labels', async () => {
    renderApp('/standings')

    expect(await screen.findByRole('heading', { name: /^standings$/i })).toBeInTheDocument()
    expect(await screen.findByRole('columnheader', { name: 'LW' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'LL' })).toBeInTheDocument()
    expect(screen.queryByRole('columnheader', { name: 'LF' })).not.toBeInTheDocument()
    expect(screen.queryByRole('columnheader', { name: 'LA' })).not.toBeInTheDocument()
    expect(screen.getByText('The Freeze')).toBeInTheDocument()
    expect(screen.getByText('Luke Humphries')).toBeInTheDocument()
  })
})