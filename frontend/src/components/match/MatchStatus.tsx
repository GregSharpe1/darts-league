import './match.css'

export type MatchStatusProps =
  | { readonly status: 'loading' }
  | { readonly status: 'error'; readonly onRetry?: () => void }

export function MatchStatus(props: MatchStatusProps) {
  switch (props.status) {
    case 'loading': return <div className="ma ma-panel" role="status">Loading match analysis...</div>
    case 'error': return <div className="ma ma-panel" role="alert"><p>Match analysis could not be loaded.</p>
      {props.onRetry && <button type="button" onClick={props.onRetry}>Try again</button>}</div>
    default: return assertNever(props)
  }
}

function assertNever(value: never): never { throw new TypeError(`Unexpected status: ${String(value)}`) }
