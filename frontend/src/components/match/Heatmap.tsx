import { useId } from 'react'
import { boardPoint, segmentLabel } from './geometry'
import type { MatchThrow } from './types'
import './match.css'

export interface HeatmapProps {
  readonly throws: readonly MatchThrow[]
  readonly label: string
  readonly synthetic: boolean
}

const order = [20, 1, 18, 4, 13, 6, 10, 15, 2, 17, 3, 19, 7, 16, 8, 11, 14, 9, 12, 5] as const
const rings = [[19, 111], [111, 121], [121, 181], [181, 191]] as const

function point(radius: number, angle: number) {
  return [250 + radius * Math.sin(angle), 250 - radius * Math.cos(angle)]
}

function wedge(inner: number, outer: number, angle: number) {
  const start = angle - Math.PI / 20
  const end = angle + Math.PI / 20
  return `M${point(inner, start)} L${point(outer, start)} A${outer},${outer} 0 0 1 ${point(outer, end)} L${point(inner, end)} A${inner},${inner} 0 0 0 ${point(inner, start)}Z`
}

export function Heatmap({ throws, label, synthetic }: HeatmapProps) {
  const id = useId()
  const recorded = throws.filter(dart => dart.position !== null).length
  const points = throws.flatMap((dart, index) => {
    const position = boardPoint(dart.position)
    return position ? [{ ...position, dart, index }] : []
  })
  const minX = Math.min(0, ...points.map(p => p.x - 20))
  const minY = Math.min(0, ...points.map(p => p.y - 20))
  const width = Math.max(500, ...points.map(p => p.x + 20)) - minX
  const height = Math.max(500, ...points.map(p => p.y + 20)) - minY

  return <figure className="ma ma-heatmap">
    {points.length ? <>
      <svg className="ma-board" viewBox={`${minX} ${minY} ${width} ${height}`} role="img" aria-labelledby={`${id}-title ${id}-desc`}>
        <title id={`${id}-title`}>{label}: recorded dart positions</title>
        <desc id={`${id}-desc`}>{points.length} plottable of {throws.length} recorded throws. Exact recorded positions, no jitter. Textual segments and visit routes follow.</desc>
        <defs><radialGradient id={`${id}-heat`}><stop stopColor="#fff0b0" stopOpacity=".96" />
          <stop offset=".25" stopColor="#ffc66d" stopOpacity=".9" /><stop offset=".6" stopColor="#ff5b5b" stopOpacity=".65" />
          <stop offset="1" stopColor="#ff5b5b" stopOpacity="0" /></radialGradient></defs>
        <circle cx="250" cy="250" r="244" fill="#090909" stroke="#343436" strokeWidth="2" />
        <circle cx="250" cy="250" r="228" fill="#121314" />
        {order.map((number, index) => {
          const angle = index * Math.PI / 10
          const [x, y] = point(212, angle)
          return <g key={number}>{rings.map(([inner, outer], ring) =>
            <path key={inner} d={wedge(inner, outer, angle)} stroke="#78736d" strokeWidth=".65"
              fill={ring % 2 ? (index % 2 ? '#355e51' : '#9b4347') : (index % 2 ? '#c8c1b6' : '#1a1b1e')} />)}
            <text x={x} y={y} dy=".35em" textAnchor="middle" fill="#c8c1b6" fontSize="16">{number}</text>
          </g>
        })}
        <circle cx="250" cy="250" r="19" fill="#355e51" stroke="#78736d" />
        <circle cx="250" cy="250" r="8" fill="#9b4347" stroke="#78736d" />
        {points.map(p => <circle key={p.index} data-heat="" cx={p.x} cy={p.y} r="19" fill={`url(#${id}-heat)`} opacity=".3" />)}
        {points.map(p => <circle key={p.index} data-dart="" cx={p.x} cy={p.y} r="2.2" fill="#fff0b0" stroke="#090909" strokeWidth=".5">
          <title>{segmentLabel(p.dart.segment)}; position provenance: {p.dart.position?.provenance}</title>
        </circle>)}
      </svg>
      <div className="ma-legend"><span className="ma-legend-ramp" />Overlapping recorded positions</div>
    </> : <div className="ma-unavailable"><h3>Coordinates unavailable</h3><p>Missing positions or unsupported geometry. No locations inferred from segments.</p></div>}
    <figcaption className="ma-note">{recorded} / {throws.length} recorded positions; {points.length} board-plottable.
      {recorded > points.length && <> {recorded - points.length} positions have unsupported geometry.</>}
      {throws.length > recorded && <> {throws.length - recorded} throws have no position.</>}
      <br />{synthetic ? 'Synthetic test positions, not measured physical accuracy.' : 'Declared geometry only; this view does not verify hardware accuracy.'}
    </figcaption>
  </figure>
}
