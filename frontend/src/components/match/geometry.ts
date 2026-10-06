import type { Position, Segment } from './types'

// This declaration describes the reference geometry, NOT an Autodarts adapter.
export function boardPoint(position: Position | null) {
  if (position?.units !== 'board-radius' || position.origin !== 'bull' ||
    position.axis_orientation !== 'x-right-y-up' ||
    !Number.isFinite(position.x) || !Number.isFinite(position.y)) return null
  return { x: 250 + position.x * 191, y: 250 - position.y * 191 }
}

export function segmentLabel(segment: Segment): string {
  switch (segment.bed) {
    case 'miss': return 'Miss'
    case 'single': return `S${segment.number}`
    case 'double': return `D${segment.number}`
    case 'triple': return `T${segment.number}`
    case 'outer_bull': return 'Outer bull'
    case 'inner_bull': return 'Bull'
    default: return assertNever(segment)
  }
}

function assertNever(value: never): never {
  throw new TypeError(`Unexpected segment: ${String(value)}`)
}
