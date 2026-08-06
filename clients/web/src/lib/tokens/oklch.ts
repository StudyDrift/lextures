/**
 * UX.1 — OKLCH colour helpers for org accent ramps and contrast checks.
 * Pure computation; no DOM. Safe to use on server (Go port) and client.
 */

export type Oklch = { l: number; c: number; h: number; a?: number }

const OKLCH_RE =
  /^oklch\(\s*([0-9.]+%?)\s+([0-9.]+)\s+([0-9.]+)\s*(?:\/\s*([0-9.]+%?))?\s*\)$/i

/** Parse `oklch(L C H)` or `oklch(L% C H)`; reject anything else (security: no CSS injection). */
export function parseOklch(input: string): Oklch | null {
  const s = input.trim()
  const m = OKLCH_RE.exec(s)
  if (!m) return null
  let l = Number(m[1].replace('%', ''))
  if (m[1].includes('%')) l = l / 100
  if (!Number.isFinite(l) || l < 0 || l > 1.001) return null
  const c = Number(m[2])
  const h = Number(m[3])
  if (!Number.isFinite(c) || c < 0 || c > 0.5) return null
  if (!Number.isFinite(h) || h < 0 || h >= 360) return null
  let a: number | undefined
  if (m[4] != null) {
    a = Number(m[4].replace('%', ''))
    if (m[4].includes('%')) a = a / 100
    if (!Number.isFinite(a) || a < 0 || a > 1) return null
  }
  return { l: Math.min(1, Math.max(0, l)), c, h, a }
}

export function formatOklch({ l, c, h, a }: Oklch): string {
  const base = `oklch(${round4(l)} ${round4(c)} ${round4(h)})`
  if (a == null || a >= 1) return base
  return `oklch(${round4(l)} ${round4(c)} ${round4(h)} / ${round4(a)})`
}

function round4(n: number): number {
  return Math.round(n * 10000) / 10000
}

/** Hex #RRGGBB → approximate OKLCH (for accent picker from brand hex). */
export function hexToOklch(hex: string): Oklch | null {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim())
  if (!m) return null
  const r = parseInt(m[1].slice(0, 2), 16) / 255
  const g = parseInt(m[1].slice(2, 4), 16) / 255
  const b = parseInt(m[1].slice(4, 6), 16) / 255
  const lin = (c: number) => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4)
  const lr = lin(r)
  const lg = lin(g)
  const lb = lin(b)
  // sRGB → linear LMS → OKLab (Björn Ottosson)
  const l_ = 0.4122214708 * lr + 0.5363325363 * lg + 0.0514459929 * lb
  const m_ = 0.2119034982 * lr + 0.6806995451 * lg + 0.1073969566 * lb
  const s_ = 0.0883024619 * lr + 0.2817188376 * lg + 0.6299787005 * lb
  const l_c = Math.cbrt(l_)
  const m_c = Math.cbrt(m_)
  const s_c = Math.cbrt(s_)
  const L = 0.2104542553 * l_c + 0.793617785 * m_c - 0.0040720468 * s_c
  const a = 1.9779984951 * l_c - 2.428592205 * m_c + 0.4505937099 * s_c
  const b2 = 0.0259040371 * l_c + 0.7827717662 * m_c - 0.808675766 * s_c
  const C = Math.sqrt(a * a + b2 * b2)
  let H = (Math.atan2(b2, a) * 180) / Math.PI
  if (H < 0) H += 360
  return { l: L, c: C, h: H }
}

/** OKLCH → sRGB hex (for contrast ratio). Clamps out-of-gamut. */
export function oklchToHex({ l, c, h }: Oklch): string {
  const hr = (h * Math.PI) / 180
  const a = c * Math.cos(hr)
  const b = c * Math.sin(hr)
  const l_ = l + 0.3963377774 * a + 0.2158037573 * b
  const m_ = l - 0.1055613458 * a - 0.0638541728 * b
  const s_ = l - 0.0894841775 * a - 1.291485548 * b
  const l3 = l_ * l_ * l_
  const m3 = m_ * m_ * m_
  const s3 = s_ * s_ * s_
  const lr = +4.0767416621 * l3 - 3.3077115913 * m3 + 0.2309699292 * s3
  const lg = -1.2684380046 * l3 + 2.6097574011 * m3 - 0.3413193965 * s3
  const lb = -0.0041960863 * l3 - 0.7034186147 * m3 + 1.707614701 * s3
  const gam = (v: number) => {
    const x = Math.min(1, Math.max(0, v))
    return x <= 0.0031308 ? 12.92 * x : 1.055 * x ** (1 / 2.4) - 0.055
  }
  const toByte = (v: number) =>
    Math.round(gam(v) * 255)
      .toString(16)
      .padStart(2, '0')
  return `#${toByte(lr)}${toByte(lg)}${toByte(lb)}`
}

export type AccentStep = '50' | '100' | '200' | '300' | '400' | '500' | '600' | '700' | '800' | '900' | '950'

/** Relative lightness offsets for a full accent ramp around a seed. */
const RAMP_L: Record<AccentStep, number> = {
  '50': 0.96,
  '100': 0.93,
  '200': 0.87,
  '300': 0.79,
  '400': 0.67,
  '500': 0.59,
  '600': 0.51,
  '700': 0.46,
  '800': 0.4,
  '900': 0.36,
  '950': 0.26,
}

/**
 * Derive a full accent ramp from a seed OKLCH (brand hue).
 * Chroma is scaled with lightness so light steps stay pastel.
 */
export function deriveAccentRamp(seed: Oklch): Record<AccentStep, string> {
  const hue = seed.h
  const chroma = Math.min(0.22, Math.max(0.08, seed.c))
  const out = {} as Record<AccentStep, string>
  for (const step of Object.keys(RAMP_L) as AccentStep[]) {
    const l = RAMP_L[step]
    // Lower chroma at extremes
    const cScale = l > 0.9 ? 0.15 : l < 0.3 ? 0.45 : 0.85 + (0.5 - Math.abs(l - 0.55)) * 0.4
    const c = chroma * cScale
    out[step] = formatOklch({ l, c, h: hue })
  }
  return out
}

/** Apply derived ramp as CSS custom properties on an element (usually documentElement). */
export function applyAccentRampToElement(
  el: HTMLElement,
  ramp: Record<AccentStep, string> | null,
): void {
  const steps: AccentStep[] = [
    '50',
    '100',
    '200',
    '300',
    '400',
    '500',
    '600',
    '700',
    '800',
    '900',
    '950',
  ]
  if (!ramp) {
    for (const s of steps) el.style.removeProperty(`--lx-accent-${s}`)
    el.removeAttribute('data-brand-accent')
    return
  }
  for (const s of steps) {
    el.style.setProperty(`--lx-accent-${s}`, ramp[s])
  }
  el.setAttribute('data-brand-accent', '1')
}
