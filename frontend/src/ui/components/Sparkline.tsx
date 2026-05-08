import type { SparkPoint } from '../types'

export function Sparkline({ points, width = 120, height = 28 }: { points?: SparkPoint[]; width?: number; height?: number }) {
  if (!points || points.length < 2) {
    return <div style={{ width, height, opacity: 0.5, fontSize: 12 }}>-</div>
  }

  const ps = points.slice(-60) // last ~60s
  let min = Number.POSITIVE_INFINITY
  let max = Number.NEGATIVE_INFINITY
  for (const p of ps) {
    min = Math.min(min, p.p)
    max = Math.max(max, p.p)
  }
  if (min === max) {
    min -= 1
    max += 1
  }

  const pad = 2
  const innerW = width - pad * 2
  const innerH = height - pad * 2

  const coords = ps.map((p, i) => {
    const x = pad + (innerW * i) / (ps.length - 1)
    const y = pad + innerH - ((p.p - min) / (max - min)) * innerH
    return `${x.toFixed(1)},${y.toFixed(1)}`
  })

  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`}>
      <polyline fill="none" stroke="currentColor" strokeWidth="1.5" points={coords.join(' ')} opacity={0.9} />
    </svg>
  )
}

