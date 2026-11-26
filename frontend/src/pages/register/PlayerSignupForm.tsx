import { useState } from 'react'
import type { FormEvent } from 'react'
import { StateNotice } from '../../components/StateNotice'
import { readError } from '../../lib/utils'

function nameError(value: string, label: string, limit: number) {
  if (/\p{Cc}/u.test(value)) return `${label} must not contain control characters.`
  // Unicode White_Space matches Go strings.Fields; JS trim also removes U+FEFF.
  const normalized = value.split(/\p{White_Space}+/u).filter(Boolean).join(' ')
  if (label === 'Display name' && !normalized) return 'Display name is required.'
  if (Array.from(normalized).length > limit) return `${label} must be at most ${limit} characters.`
  return ''
}

type Props = {
  action: string
  busy: boolean
  disabled?: boolean
  error: unknown
  onRegister: (entry: { display_name: string; nickname: string }) => Promise<string>
}

export function PlayerSignupForm({ action, busy, disabled, error, onRegister }: Props) {
  const [displayName, setDisplayName] = useState('')
  const [nickname, setNickname] = useState('')
  const [successMessage, setSuccessMessage] = useState('')
  const [submitted, setSubmitted] = useState(false)
  const displayNameError = submitted ? nameError(displayName, 'Display name', 60) : ''
  const nicknameError = submitted ? nameError(nickname, 'Nickname', 30) : ''

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSuccessMessage('')
    setSubmitted(true)
    if (nameError(displayName, 'Display name', 60) || nameError(nickname, 'Nickname', 30)) return
    try {
      const message = await onRegister({ display_name: displayName, nickname })
      setSuccessMessage(message)
      setDisplayName('')
      setNickname('')
      setSubmitted(false)
    } catch {
      // handled by mutation error state
    }
  }

  return (
    <>
      <form className="form-preview" onSubmit={handleSubmit}>
        <div className="field">
          <label htmlFor="display-name">Display name</label>
          <input id="display-name" name="display-name" required aria-invalid={Boolean(displayNameError)} aria-describedby="display-name-help display-name-error" value={displayName} onChange={(event) => setDisplayName(event.target.value)} disabled={disabled || busy} />
          <span id="display-name-help" className="fixture-meta">Required. Up to 60 characters.</span>
          <span id="display-name-error" role="alert">{displayNameError}</span>
        </div>
        <div className="field">
          <label htmlFor="nickname">Nickname</label>
          <input id="nickname" name="nickname" aria-invalid={Boolean(nicknameError)} aria-describedby="nickname-help nickname-error" value={nickname} onChange={(event) => setNickname(event.target.value)} disabled={disabled || busy} />
          <span id="nickname-help" className="fixture-meta">Optional. Up to 30 characters.</span>
          <span id="nickname-error" role="alert">{nicknameError}</span>
        </div>
        <button type="submit" disabled={disabled || busy}>{busy ? 'Registering...' : action}</button>
      </form>
      {error ? <StateNotice tone="error" message={readError(error)} compact /> : null}
      {successMessage ? <StateNotice message={successMessage} compact /> : null}
    </>
  )
}
