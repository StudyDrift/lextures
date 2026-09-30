import { getAccountType } from './auth'

/** College quality-rubric codes that are opaque to Homeschool / K–12 parents. */
export function isCampusRubricSource(src: string): boolean {
  const s = src.trim()
  if (!s) return false
  const upper = s.toUpperCase()
  return (
    upper === 'QM' ||
    upper.startsWith('QM ') ||
    upper.startsWith('QM.') ||
    upper === 'OSCQR' ||
    upper.startsWith('OSCQR ') ||
    upper.startsWith('OSCQR.')
  )
}

/**
 * Hide QM/OSCQR tags for Homeschool / K–12 audiences.
 * Server already filters for HomeschoolMode / OrgIsK12; this covers parent
 * accounts (self.lextures.com) where the course may still look like higher-ed.
 */
export function filterChecklistSourcesForAudience(
  sources: string[] | null | undefined,
  opts?: { forceFamilyAudience?: boolean },
): string[] {
  const list = Array.isArray(sources) ? sources : []
  const family =
    opts?.forceFamilyAudience === true || getAccountType() === 'parent'
  if (!family) return list
  return list.filter((src) => !isCampusRubricSource(src))
}
