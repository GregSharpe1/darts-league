import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { createMockFetch, renderApp, response } from '../../test/helpers'

afterEach(() => { cleanup(); vi.unstubAllGlobals() })

async function setup(status = 201, preferredName = 'Player') {
  const register = vi.fn(async () => status === 201
    ? response({ id: 1, preferred_name: preferredName }, status)
    : response({ error: { code: 'duplicate_display_name', message: 'Already registered.' } }, status))
  const fallback = createMockFetch({ authenticated: false, seasonStarted: false, seasonName: 'Test' })
  vi.stubGlobal('fetch', (input: RequestInfo | URL, init?: RequestInit) => input === '/api/players/register' ? register() : fallback(input, init))
  renderApp('/register')
  await waitFor(() => expect(screen.getByLabelText('Display name')).toBeEnabled())
  return register
}

it.each([
  ['Display name', 'a'.repeat(61)],
  ['Display name', '   '],
  ['Display name', 'A\tB'],
  ['Nickname', '\u{1f3af}'.repeat(31)],
  ['Nickname', 'B\u0085'],
  ['Display name', '\ufeff' + 'a'.repeat(60)],
])('rejects invalid %s without submitting', async (field, value) => {
  const register = await setup()
  fireEvent.change(screen.getByLabelText('Display name'), { target: { value: 'A' } })
  fireEvent.change(screen.getByLabelText(field), { target: { value } })
  fireEvent.click(screen.getByRole('button', { name: 'Register for the league' }))
  expect(screen.getByLabelText(field)).toHaveAttribute('aria-invalid', 'true')
  expect(screen.getByLabelText(field)).toHaveAccessibleDescription(/required|at most|control characters/i)
  expect(screen.getByLabelText(field)).toHaveValue(value)
  expect(register).not.toHaveBeenCalled()
})

it.each([
  ['\u{1f3af}'.repeat(60), '\u{1f3af}'.repeat(30)],
  ['  ' + 'a'.repeat(58) + '   B  ', '  '],
  ["\u674e O'Brien-\u00e9", ''],
])('accepts valid Unicode input and resets after success', async (displayName, nickname) => {
  const register = await setup()
  fireEvent.change(screen.getByLabelText('Display name'), { target: { value: displayName } })
  fireEvent.change(screen.getByLabelText('Nickname'), { target: { value: nickname } })
  fireEvent.click(screen.getByRole('button', { name: 'Register for the league' }))
  await screen.findByText(/Player is registered/)
  expect(register).toHaveBeenCalledOnce()
  expect(screen.getByLabelText('Display name')).toHaveValue('')
  expect(screen.getByLabelText('Nickname')).toHaveValue('')
})

it('keeps input when the server rejects registration', async () => {
  await setup(409)
  fireEvent.change(screen.getByLabelText('Display name'), { target: { value: 'A' } })
  fireEvent.click(screen.getByRole('button', { name: 'Register for the league' }))
  expect(await screen.findByText('Already registered.')).toBeVisible()
  expect(screen.getByLabelText('Display name')).toHaveValue('A')
})

it('renders returned names as text rather than HTML', async () => {
  await setup(201, '<img src=x onerror=alert(1)>')
  expect(screen.getByLabelText('Display name')).toBeRequired()
  fireEvent.change(screen.getByLabelText('Display name'), { target: { value: 'A' } })
  fireEvent.click(screen.getByRole('button', { name: 'Register for the league' }))
  expect(await screen.findByText(/<img src=x onerror=alert\(1\)> is registered/)).toBeVisible()
  expect(document.querySelector('img[src="x"]')).toBeNull()
})
