import { useEffect, useState } from 'react'
import { usePlatformFeatures } from '../context/platform-features-context'
import { getAccountType } from '../lib/auth'
import { isHomeschoolOrK12Audience } from '../lib/family-audience'
import { fetchOrgType, type CoursePublic, type OrgType } from '../lib/courses-api'
import { listDependents } from '../lib/managed-learners-api'

export function useFamilyAudience(course: CoursePublic | null | undefined): boolean {
  const [orgType, setOrgType] = useState<OrgType | null>(null)
  const orgId = course?.orgId?.trim() ?? ''
  const gradeLevels = course?.gradeLevels

  useEffect(() => {
    const hasGradeLevels = (gradeLevels ?? []).some((level) => level.trim() !== '')
    if (!orgId || hasGradeLevels) {
      setOrgType(null)
      return
    }
    let cancelled = false
    void fetchOrgType(orgId)
      .then((next) => {
        if (!cancelled) setOrgType(next)
      })
      .catch(() => {
        if (!cancelled) setOrgType('higher-ed')
      })
    return () => {
      cancelled = true
    }
  }, [orgId, gradeLevels])

  return isHomeschoolOrK12Audience({
    accountType: getAccountType(),
    orgKnown: course != null,
    orgId: course?.orgId,
    gradeLevels,
    orgType,
  })
}

/**
 * Family progress UI for homeschool / K–12 courses, and for any course that
 * currently enrolls one of the viewer's managed learners.
 */
export function useFamilyProgressAudience(
  course: CoursePublic | null | undefined,
  rosterUserIds: readonly string[],
): boolean {
  const courseFamily = useFamilyAudience(course)
  const { ffHomeschoolManagedLearners, loading } = usePlatformFeatures()
  const [managedOnRoster, setManagedOnRoster] = useState(false)
  const rosterKey = rosterUserIds.join('\n')

  useEffect(() => {
    if (loading || !ffHomeschoolManagedLearners || rosterKey === '') {
      setManagedOnRoster(false)
      return
    }
    let cancelled = false
    void listDependents()
      .then((deps) => {
        if (cancelled) return
        const ids = new Set(deps.map((d) => d.id))
        setManagedOnRoster(rosterKey.split('\n').some((id) => ids.has(id)))
      })
      .catch(() => {
        if (!cancelled) setManagedOnRoster(false)
      })
    return () => {
      cancelled = true
    }
  }, [ffHomeschoolManagedLearners, loading, rosterKey])

  return courseFamily || managedOnRoster
}
