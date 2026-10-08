import { describe, expect, it } from 'vitest'
import { boardPoint, segmentLabel } from './geometry'
import type { Position } from './types'

const position: Position = { x: -0.0013284654446509365, y: 0.6086262465766682,
  units: 'board-radius', origin: 'bull', axis_orientation: 'x-right-y-up', provenance: 'manual' }

describe('declared geometry', () => {
  it('projects exact recorded coordinates when geometry is supported', () => {
    expect(boardPoint(position)).toEqual({ x: 249.74626310007167, y: 133.75238690385638 })
  })
  it('keeps genuine zero coordinates when recorded', () => {
    expect(boardPoint({ ...position, x: 0, y: 0 })).toEqual({ x: 250, y: 250 })
  })
  it('returns unavailable when finite raw coordinates overflow during projection', () => {
    expect(boardPoint({ ...position, x: Number.MAX_VALUE })).toBeNull()
  })
  it.each([null, { ...position, units: null }, { ...position, origin: null },
    { ...position, axis_orientation: 'unknown' }, { ...position, x: NaN }])(
    'refuses to guess geometry for %j', (value) => { expect(boardPoint(value)).toBeNull() })
  it('labels segments independently of coordinates', () => {
    expect(segmentLabel({ bed: 'miss', number: 0 })).toBe('Miss')
    expect(segmentLabel({ bed: 'triple', number: 20 })).toBe('T20')
    expect(segmentLabel({ bed: 'inner_bull', number: 25 })).toBe('Bull')
  })
})
