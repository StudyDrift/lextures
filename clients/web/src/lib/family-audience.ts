export type OrgAudienceType = 'higher-ed' | 'k-12'

/**
 * Homeschool and K–12 audiences get plain-language quiz labels.
 * Matches checklist HomeschoolMode (no organization) and OrgIsK12 (grade levels
 * or org type), and also parent accounts on the hosted homeschool app.
 */
export function isHomeschoolOrK12Audience(input: {
  accountType?: string | null
  /** True once the course record is loaded. Unknown courses stay on campus copy. */
  orgKnown?: boolean
  /** null or empty when the course has no organization. */
  orgId?: string | null
  gradeLevels?: readonly string[] | null
  orgType?: OrgAudienceType | null
}): boolean {
  if (input.accountType === 'parent') return true
  if ((input.gradeLevels ?? []).some((level) => level.trim() !== '')) return true
  if (input.orgType === 'k-12') return true
  if (!input.orgKnown) return false
  return (input.orgId?.trim() ?? '') === ''
}
