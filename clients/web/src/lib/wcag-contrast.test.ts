/**
 * Unit tests for WCAG 2.1 SC 1.4.3 contrast-ratio calculation.
 *
 * The same formula is used by scripts/check-contrast.mjs; these tests verify
 * correctness against known reference values so that the CI gate is trustworthy.
 */
import { describe, it, expect } from 'vitest'

// ── Inline WCAG formula (mirrors check-contrast.mjs, kept in-sync manually) ─

function hexToRgb(hex: string): [number, number, number] {
  const h = hex.replace('#', '')
  return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)]
}

function linearize(c: number): number {
  return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
}

function luminance(hex: string): number {
  const [r, g, b] = hexToRgb(hex).map((v) => linearize(v / 255))
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

function contrastRatio(fg: string, bg: string): number {
  const l1 = luminance(fg)
  const l2 = luminance(bg)
  const lighter = Math.max(l1, l2)
  const darker = Math.min(l1, l2)
  return (lighter + 0.05) / (darker + 0.05)
}

// ── Tests ────────────────────────────────────────────────────────────────────

describe('contrastRatio', () => {
  it('black on white is 21:1', () => {
    expect(contrastRatio('#000000', '#ffffff')).toBeCloseTo(21, 0)
  })

  it('white on white is 1:1', () => {
    expect(contrastRatio('#ffffff', '#ffffff')).toBeCloseTo(1, 1)
  })

  it('is symmetric — foreground/background order does not matter', () => {
    const a = contrastRatio('#64748b', '#ffffff')
    const b = contrastRatio('#ffffff', '#64748b')
    expect(a).toBeCloseTo(b, 5)
  })

  describe('passing pairs (WCAG AA normal text ≥ 4.5:1)', () => {
    const passingPairs: Array<[string, string, string]> = [
      // [fg, bg, label]
      ['#0f172a', '#ffffff', 'slate-900 on white'],
      ['#475569', '#ffffff', 'slate-600 on white'],
      ['#64748b', '#ffffff', 'slate-500 on white (placeholder)'],
      ['#0f172a', '#f8fafc', 'slate-900 on slate-50'],
      ['#475569', '#f8fafc', 'slate-600 on slate-50'],
      ['#64748b', '#f8fafc', 'slate-500 on slate-50 (placeholder)'],
      ['#ffffff', '#4f46e5', 'white on indigo-600 (primary button)'],
      ['#ffffff', '#4338ca', 'white on indigo-700 (button hover)'],
      ['#dc2626', '#ffffff', 'red-600 on white (error text)'],
      ['#f5f5f5', '#0a0a0a', 'neutral-100 on neutral-950 (dark body)'],
      ['#e5e5e5', '#171717', 'neutral-200 on neutral-900 (dark card heading)'],
      ['#a3a3a3', '#171717', 'neutral-400 on neutral-900 (dark muted text)'],
      ['#f5f5f5', '#262626', 'neutral-100 on neutral-800 (dark elevated surface)'],
      ['#a3a3a3', '#262626', 'neutral-400 on neutral-800 (dark muted on elevated)'],
      ['#a3a3a3', '#0a0a0a', 'neutral-400 on neutral-950 (dark placeholder)'],
      ['#818cf8', '#0a0a0a', 'indigo-400 on neutral-950 (dark accent)'],
      ['#818cf8', '#171717', 'indigo-400 on neutral-900 (dark card accent)'],
      ['#15803d', '#ffffff', 'green-700 on white (success text)'],
    ]

    it.each(passingPairs)('%s on %s (%s) meets 4.5:1', (fg, bg) => {
      expect(contrastRatio(fg, bg)).toBeGreaterThanOrEqual(4.5)
    })
  })

  describe('failing pairs (below WCAG AA)', () => {
    it('slate-400 on white fails 4.5:1 (old placeholder color, fixed in index.css)', () => {
      // #94a3b8 on white was the old TipTap placeholder — ~2.57:1, fails AA
      expect(contrastRatio('#94a3b8', '#ffffff')).toBeLessThan(4.5)
    })

    it('neutral-500 on neutral-950 fails 4.5:1 (old dark placeholder, fixed in index.css)', () => {
      // rgb(115 115 115) = #737373 on neutral-950 — ~4.17:1, fails AA
      expect(contrastRatio('#737373', '#0a0a0a')).toBeLessThan(4.5)
    })

    it('green-600 on white fails 4.5:1 — use green-700 or darker for success text', () => {
      // green-600 (#16a34a) on white is only 3.30:1 — use green-700 (#15803d) instead
      expect(contrastRatio('#16a34a', '#ffffff')).toBeLessThan(4.5)
    })
  })
})

describe('WCAG AA compliance for approved token pairs from contrast-config.json', () => {
  it('every light-mode pair in the config meets ≥ 4.5:1', async () => {
    const { default: configRaw } = await import('../../contrast-config.json', {
      with: { type: 'json' },
    })
    const config = configRaw as {
      tokens: Record<string, string>
      pairs: { light?: Array<{ foreground: string; background: string; minRatio?: number }> }
    }

    for (const pair of config.pairs.light ?? []) {
      const fg = config.tokens[pair.foreground]
      const bg = config.tokens[pair.background]
      const ratio = contrastRatio(fg, bg)
      const threshold = pair.minRatio ?? 4.5
      expect(
        ratio,
        `${pair.foreground} on ${pair.background} should meet ${threshold}:1`,
      ).toBeGreaterThanOrEqual(threshold)
    }
  })

  it('every dark-mode pair in the config meets ≥ 4.5:1', async () => {
    const { default: configRaw } = await import('../../contrast-config.json', {
      with: { type: 'json' },
    })
    const config = configRaw as {
      tokens: Record<string, string>
      pairs: { dark?: Array<{ foreground: string; background: string; minRatio?: number }> }
    }

    for (const pair of config.pairs.dark ?? []) {
      const fg = config.tokens[pair.foreground]
      const bg = config.tokens[pair.background]
      const ratio = contrastRatio(fg, bg)
      const threshold = pair.minRatio ?? 4.5
      expect(
        ratio,
        `${pair.foreground} on ${pair.background} should meet ${threshold}:1`,
      ).toBeGreaterThanOrEqual(threshold)
    }
  })
})
