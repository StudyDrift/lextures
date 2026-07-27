export type SortItem = { id: string; text: string; imageUrl?: string; imageAlt?: string }
export type SortBucket = { id: string; label: string; description?: string }

export type CheckResult = {
  perItem?: Record<string, { correct?: boolean; feedback?: string }>
  scorePct?: number
  attemptsRemaining?: number
  showPerItem?: boolean
  error?: string
  message?: string
}

export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) return false
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}
