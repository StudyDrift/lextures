/**
 * University / district hosted pricing: per-student rate with bulk discounts.
 *
 * Tiers (users = enrolled students):
 *   < 15,000              → $6.00 / student
 *   15,000 – 24,999       → $5.50 / student
 *   25,000 – 50,000       → $4.50 / student
 *   > 50,000              → $3.00 / student
 */

export type PricingTier = {
  /** Inclusive lower bound of the tier (null = no lower bound). */
  minUsers: number | null
  /** Inclusive upper bound of the tier (null = no upper bound). */
  maxUsers: number | null
  /** USD per student for this tier. */
  pricePerStudent: number
  label: string
}

export const PRICING_TIERS: readonly PricingTier[] = [
  { minUsers: null, maxUsers: 14_999, pricePerStudent: 6, label: 'Under 15,000 students' },
  { minUsers: 15_000, maxUsers: 24_999, pricePerStudent: 5.5, label: '15,000 – 24,999 students' },
  { minUsers: 25_000, maxUsers: 50_000, pricePerStudent: 4.5, label: '25,000 – 50,000 students' },
  { minUsers: 50_001, maxUsers: null, pricePerStudent: 3, label: 'More than 50,000 students' },
] as const

/** Slider bounds for the public pricing calculator. */
export const CALCULATOR_MIN_USERS = 500
export const CALCULATOR_MAX_USERS = 100_000
export const CALCULATOR_STEP = 500
export const CALCULATOR_DEFAULT_USERS = 5_000

/**
 * Returns the per-student USD rate for a given enrolled-user count.
 * Counts at or below zero fall back to the base ($6) rate.
 */
export function pricePerStudent(users: number): number {
  const n = Math.max(0, Math.floor(users))
  if (n > 50_000) return 3
  if (n >= 25_000) return 4.5
  if (n >= 15_000) return 5.5
  return 6
}

/** Estimated total cost for the given number of students. */
export function estimatedTotal(users: number): number {
  const n = Math.max(0, Math.floor(users))
  return n * pricePerStudent(n)
}

/** Human-readable tier description for the current user count. */
export function tierLabelForUsers(users: number): string {
  const rate = pricePerStudent(users)
  const tier = PRICING_TIERS.find(t => t.pricePerStudent === rate)
  return tier?.label ?? 'Custom'
}

/** Format a USD amount (e.g. 5.5 → "$5.50", 7 → "$7"). */
export function formatUsd(amount: number, opts?: { forceCents?: boolean }): string {
  const forceCents = opts?.forceCents ?? false
  if (!forceCents && Number.isInteger(amount)) {
    return `$${amount.toLocaleString('en-US')}`
  }
  return `$${amount.toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}`
}

/** Format a student count with thousands separators. */
export function formatUsers(users: number): string {
  return Math.floor(users).toLocaleString('en-US')
}
