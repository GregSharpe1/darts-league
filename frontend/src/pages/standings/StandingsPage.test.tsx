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

  it('renders division standings with the expected columns', async () => {
    renderApp('/divisions/premier/standings')

    expect(await screen.findByRole('heading', { name: /premier division/i })).toBeInTheDocument()
    expect(await screen.findByRole('columnheader', { name: 'LW' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'LL' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Avg' })).toBeInTheDocument()
    expect(screen.getByText('The Freeze')).toBeInTheDocument()
    expect(screen.getByLabelText('Position 1')).toBeInTheDocument()
    expect(screen.getByText('Luke Humphries')).toBeInTheDocument()
    expect(screen.getByText('95.40')).toBeInTheDocument()
  })
})
