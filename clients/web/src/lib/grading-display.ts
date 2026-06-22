/** Map a course percentage to a letter label using the course grading scheme (plan 3.16). */

export type GradingSchemeLike = {
  type: string
  scaleJson: unknown
} | null | undefined

type LetterTier = { label: string; min_pct: number; gpa?: number }

function parseLetterTiers(scheme: GradingSchemeLike): LetterTier[] {
  const raw = scheme?.scaleJson
  if (!Array.isArray(raw)) return []
  const tiers: LetterTier[] = []
  for (const x of raw) {
    if (!x || typeof x !== 'object') continue
    const o = x as Record<string, unknown>
    const label = typeof o.label === 'string' ? o.label.trim() : ''
    const minPct =
      typeof o.min_pct === 'number'
        ? o.min_pct
        : typeof o.minPct === 'number'
          ? o.minPct
          : NaN
    if (!label || !Number.isFinite(minPct)) continue
    tiers.push({ label, min_pct: minPct })
  }
  tiers.sort((a, b) => b.min_pct - a.min_pct)
  return tiers
}

const DEFAULT_LETTER_TIERS: LetterTier[] = [
  { label: 'A', min_pct: 90 },
  { label: 'B', min_pct: 80 },
  { label: 'C', min_pct: 70 },
  { label: 'D', min_pct: 60 },
  { label: 'F', min_pct: 0 },
]

/** Returns the letter (or pass/fail) label for a percentage, or null when not applicable. */
export function percentToDisplayGrade(
  percent: number | null,
  scheme: GradingSchemeLike,
): string | null {
  if (percent == null || !Number.isFinite(percent)) return null
  const kind = scheme?.type?.trim() ?? 'points'
  if (kind === 'pass_fail') return percent >= 60 ? 'Pass' : 'Fail'
  if (kind === 'complete_incomplete') return percent >= 60 ? 'Complete' : 'Incomplete'
  if (kind !== 'letter' && kind !== 'gpa') return null
  const tiers = parseLetterTiers(scheme)
  const use = tiers.length > 0 ? tiers : DEFAULT_LETTER_TIERS
  for (const t of use) {
    if (percent + 1e-9 >= t.min_pct) return t.label
  }
  return use.at(-1)?.label ?? null
}

/** Letter tier labels for a target-grade selector (highest first). */
export function letterTierOptions(scheme: GradingSchemeLike): { label: string; minPct: number }[] {
  const tiers = parseLetterTiers(scheme)
  const use = tiers.length > 0 ? tiers : DEFAULT_LETTER_TIERS
  return use.map((t) => ({ label: t.label, minPct: t.min_pct }))
}
