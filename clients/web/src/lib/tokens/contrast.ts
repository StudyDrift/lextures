/**
 * UX.1 — WCAG contrast helpers for semantic pair validation.
 */

function hexToRgb(hex: string): [number, number, number] {
  const h = hex.replace('#', '')
  return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)]
}

function linearize(c: number): number {
  return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
}

export function relativeLuminance(hex: string): number {
  const [r, g, b] = hexToRgb(hex).map((v) => linearize(v / 255))
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

export function contrastRatio(hex1: string, hex2: string): number {
  const l1 = relativeLuminance(hex1)
  const l2 = relativeLuminance(hex2)
  const lighter = Math.max(l1, l2)
  const darker = Math.min(l1, l2)
  return (lighter + 0.05) / (darker + 0.05)
}

export const AA_NORMAL = 4.5
export const AA_LARGE = 3.0
export const AA_UI = 3.0

export type ContrastPair = {
  fg: string
  bg: string
  minRatio: number
  usage: string
}

/** Semantic (fg, bg) pairings that must pass in every theme. */
export const SEMANTIC_PAIRS: ContrastPair[] = [
  { fg: 'fg-default', bg: 'surface-base', minRatio: AA_NORMAL, usage: 'body on page' },
  { fg: 'fg-default', bg: 'surface-raised', minRatio: AA_NORMAL, usage: 'body on cards' },
  { fg: 'fg-muted', bg: 'surface-base', minRatio: AA_NORMAL, usage: 'muted on page' },
  { fg: 'fg-muted', bg: 'surface-raised', minRatio: AA_NORMAL, usage: 'muted on cards' },
  { fg: 'fg-subtle', bg: 'surface-raised', minRatio: AA_NORMAL, usage: 'placeholder / tertiary' },
  { fg: 'fg-on-accent', bg: 'accent-solid', minRatio: AA_NORMAL, usage: 'primary button label' },
  { fg: 'fg-default', bg: 'surface-sunken', minRatio: AA_NORMAL, usage: 'text on sunken' },
  { fg: 'info-fg', bg: 'info-surface', minRatio: AA_NORMAL, usage: 'info status' },
  { fg: 'success-fg', bg: 'success-surface', minRatio: AA_NORMAL, usage: 'success status' },
  { fg: 'warning-fg', bg: 'warning-surface', minRatio: AA_NORMAL, usage: 'warning status' },
  { fg: 'danger-fg', bg: 'danger-surface', minRatio: AA_NORMAL, usage: 'danger status' },
  { fg: 'accent-fg', bg: 'accent-surface', minRatio: AA_NORMAL, usage: 'accent status' },
  { fg: 'border-strong', bg: 'surface-raised', minRatio: AA_UI, usage: 'strong UI border' },
  { fg: 'focus-ring', bg: 'surface-raised', minRatio: AA_UI, usage: 'focus ring vs surface' },
]
