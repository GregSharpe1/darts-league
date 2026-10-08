import { expect, it } from 'vitest'
import { formatAverage } from './api'

it.each([
  [undefined, ''],
  [null, ''],
  [0, '0.0'],
  [61.27, '61.3'],
] as const)('formats average %s without treating missing data as zero', (value, expected) => {
  expect(formatAverage(value)).toBe(expected)
})
