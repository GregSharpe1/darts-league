export function StateNotice({ message, tone = 'neutral', compact = false }: {
  message: string
  tone?: 'neutral' | 'error'
  compact?: boolean
}) {
  return <p className={`state-notice ${tone}${compact ? ' compact' : ''}`}>{message}</p>
}
