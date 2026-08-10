import { useMemo, type ReactNode } from 'react'
import { ChecklistBadge } from '../checklist/checklist-badge'
import { useCourseChecklistSummary } from '../../context/course-checklist-summary-context'
import { useCourseNavFeatures } from '../../context/course-nav-features-context'
import {
  courseEnrollmentsReadPermission,
  viewerIsCourseStaffEnrollment,
  viewerShouldHideCourseEnrollmentsNav,
  viewerShouldShowMyGradesNav,
} from '../../lib/courses-api'
import { useCourseViewAs } from '../../lib/course-view-as'
import { useViewerEnrollmentRoles } from '../../lib/use-viewer-enrollment-roles'
import { usePermissions } from '../../context/use-permissions'
import { courseChecklistI18n } from '../../lib/course-checklist-i18n'
import type { NavAudience } from '../../lib/nav'
import { RegistryNavLinks } from './nav/registry-nav-links'
import { NavCustomiseSheetTrigger } from './nav/customise-nav-sheet'

type SideNavCourseLinksProps = {
  courseCode: string
}

/**
 * In-course destinations — UX.7 registry with audience from "View as" + enrollment.
 */
export function SideNavCourseLinks({ courseCode }: SideNavCourseLinksProps) {
  const courseFeatures = useCourseNavFeatures()
  const { allows, loading: permLoading } = usePermissions()
  const courseViewPreview = useCourseViewAs(courseCode)
  const viewerEnrollmentRoles = useViewerEnrollmentRoles(courseCode)
  const { summary: checklistSummary } = useCourseChecklistSummary()

  const audience: NavAudience = useMemo(() => {
    if (courseViewPreview === 'student') return 'student'
    if (viewerEnrollmentRoles !== null && viewerIsCourseStaffEnrollment(viewerEnrollmentRoles)) {
      return 'instructor'
    }
    if (viewerShouldShowMyGradesNav(viewerEnrollmentRoles, courseViewPreview)) {
      return 'student'
    }
    // Staff with manage rights without enrollment roles loaded yet
    if (!permLoading && allows(`course:${courseCode}:item:create`.replace('item:create', 'item:create'))) {
      // fall through — permission gates still apply per destination
    }
    if (courseViewPreview === 'teacher') return 'instructor'
    return 'any'
  }, [courseViewPreview, viewerEnrollmentRoles, permLoading, allows, courseCode])

  // Refine instructor: can manage course or view gradebook
  const effectiveAudience: NavAudience = useMemo(() => {
    if (courseViewPreview === 'student') return 'student'
    if (audience === 'instructor') return 'instructor'
    if (audience === 'student') return 'student'
    // Default staff check via enrollment roles
    if (viewerEnrollmentRoles !== null && viewerIsCourseStaffEnrollment(viewerEnrollmentRoles)) {
      return 'instructor'
    }
    return audience
  }, [courseViewPreview, audience, viewerEnrollmentRoles])

  const canViewEnrollments =
    viewerEnrollmentRoles !== null &&
    viewerIsCourseStaffEnrollment(viewerEnrollmentRoles) &&
    !viewerShouldHideCourseEnrollmentsNav(viewerEnrollmentRoles, courseViewPreview) &&
    !permLoading &&
    allows(courseEnrollmentsReadPermission(courseCode))

  const canViewMyGrades = viewerShouldShowMyGradesNav(viewerEnrollmentRoles, courseViewPreview)

  const checklistOutstanding = checklistSummary?.outstandingEssential ?? 0
  const showChecklist = effectiveAudience === 'instructor' && courseViewPreview !== 'student'

  const itemExtras = useMemo(() => {
    if (!showChecklist) {
      return {} as Record<string, { badge?: ReactNode; tooltip?: string }>
    }
    const checklistBadge =
      checklistOutstanding > 0 ? (
        <ChecklistBadge outstandingEssential={checklistOutstanding} />
      ) : undefined
    const checklistTooltip =
      checklistOutstanding > 0
        ? `${courseChecklistI18n.navLabel} (${checklistOutstanding > 99 ? '99+' : checklistOutstanding})`
        : courseChecklistI18n.navLabel
    return {
      'course.checklist': {
        badge: checklistBadge,
        tooltip: checklistTooltip,
      },
    }
  }, [showChecklist, checklistOutstanding])

  const platformExtras = useMemo(
    () => ({
      _canViewEnrollments: canViewEnrollments,
      _canViewMyGrades: canViewMyGrades,
    }),
    [canViewEnrollments, canViewMyGrades],
  )

  const courseFeatureRecord = useMemo(
    () => courseFeatures as unknown as Record<string, unknown>,
    [courseFeatures],
  )

  return (
    <RegistryNavLinks
      scope="course"
      courseCode={courseCode}
      audience={effectiveAudience}
      platformExtras={platformExtras}
      courseFeatures={courseFeatureRecord}
      itemExtras={itemExtras}
      footer={<NavCustomiseSheetTrigger scope="course" courseCode={courseCode} />}
    />
  )
}
