type WindPath = {
  d: string
  stroke: string
  opacity: number
  dash: string
  anim: 'lx-drift' | 'lx-drift-slow'
  dur: number
}

type Variant = 'hero' | 'deep' | 'sand' | 'teal'

const VARIANTS: Record<
  Variant,
  { viewBox: string; preserve: string; opacity: number; paths: WindPath[] }
> = {
  hero: {
    viewBox: '0 0 1440 620',
    preserve: 'xMidYMid slice',
    opacity: 0.4,
    paths: [
      { d: 'M-200,150 C120,110 320,190 560,150 S1040,110 1360,150 1720,190 1960,150', stroke: '#6ac5b0', opacity: 0.5, dash: '34 40', anim: 'lx-drift', dur: 26 },
      { d: 'M-200,235 C160,200 360,270 620,235 S1080,200 1420,235 1780,270 2020,235', stroke: '#f2684e', opacity: 0.28, dash: '20 46', anim: 'lx-drift', dur: 34 },
      { d: 'M-200,330 C120,296 300,360 560,330 S1060,296 1380,330 1760,360 2000,330', stroke: '#6ac5b0', opacity: 0.32, dash: '52 60', anim: 'lx-drift-slow', dur: 40 },
      { d: 'M-200,430 C150,398 340,456 600,430 S1080,398 1400,430 1780,456 2020,430', stroke: '#f49b44', opacity: 0.3, dash: '16 52', anim: 'lx-drift', dur: 30 },
    ],
  },
  deep: {
    viewBox: '0 0 1440 400',
    preserve: 'xMidYMid slice',
    opacity: 0.3,
    paths: [
      { d: 'M-200,120 C160,86 360,150 620,120 S1080,86 1420,120 1800,150 2040,120', stroke: '#6ac5b0', opacity: 0.4, dash: '40 54', anim: 'lx-drift', dur: 32 },
      { d: 'M-200,250 C150,216 340,280 600,250 S1080,216 1400,250 1780,280 2020,250', stroke: '#f2684e', opacity: 0.34, dash: '18 50', anim: 'lx-drift-slow', dur: 42 },
    ],
  },
  sand: {
    viewBox: '0 0 1440 360',
    preserve: 'none',
    opacity: 0.7,
    paths: [
      { d: 'M-200,90 C160,60 360,120 620,90 S1080,60 1420,90 1800,120 2040,90', stroke: '#c9a86a', opacity: 0.5, dash: '44 56', anim: 'lx-drift', dur: 38 },
      { d: 'M-200,270 C150,240 340,300 600,270 S1080,240 1400,270 1780,300 2020,270', stroke: '#6ac5b0', opacity: 0.4, dash: '22 48', anim: 'lx-drift-slow', dur: 46 },
    ],
  },
  teal: {
    viewBox: '0 0 1440 420',
    preserve: 'xMidYMid slice',
    opacity: 0.4,
    paths: [
      { d: 'M-200,130 C160,96 360,160 620,130 S1080,96 1420,130 1800,160 2040,130', stroke: '#ffffff', opacity: 0.4, dash: '38 52', anim: 'lx-drift', dur: 28 },
      { d: 'M-200,270 C150,236 340,300 600,270 S1080,236 1400,270 1780,300 2020,270', stroke: '#f1e5c6', opacity: 0.5, dash: '18 46', anim: 'lx-drift-slow', dur: 36 },
    ],
  },
}

/** Drifting current-of-wind lines used as an ambient background motif across sections. */
export function WindLines({ variant }: { variant: Variant }) {
  const { viewBox, preserve, opacity, paths } = VARIANTS[variant]
  return (
    <svg
      viewBox={viewBox}
      preserveAspectRatio={preserve}
      aria-hidden
      className="pointer-events-none absolute inset-0 h-full w-full"
      style={{ opacity }}
    >
      <g fill="none" strokeLinecap="round" strokeWidth={variant === 'hero' ? 2.4 : 2}>
        {paths.map((p, i) => (
          <path
            key={i}
            d={p.d}
            stroke={p.stroke}
            strokeOpacity={p.opacity}
            strokeDasharray={p.dash}
            style={{ animation: `${p.anim} ${p.dur}s linear infinite` }}
          />
        ))}
      </g>
    </svg>
  )
}
