import { GraduationCap } from 'lucide-react'
import { matchPath, useLocation } from 'react-router-dom'
import { useEffect, useMemo, useRef } from 'react'
import {
  notifyCourseViewerEnrollmentChanged,
  setCourseViewAs,
  useCourseViewAs,
} from '../lib/course-view-as'
import {
  ensureCourseTestStudentEnrollment,
  isStudentEquivalentEnrollmentRole,
  viewerIsCourseStaffEnrollment,
} from '../lib/courses-api'
import { useViewerEnrollmentRoles } from '../lib/use-viewer-enrollment-roles'
import { Button } from './ui/button'

/**
 * Sticky course chrome banner while staff are previewing as Test Student.
 * Exit returns to staff view without removing the Test Student enrollment.
 */
export function StudentPreviewBanner() {
  const location = useLocation()
  const courseCode = useMemo(() => {
    const m = matchPath({ path: '/courses/:courseCode', end: false }, location.pathname)
    const code = m?.params.courseCode
    return code && code !== 'create' ? code : null
  }, [location.pathname])

  const viewAs = useCourseViewAs(courseCode ?? undefined)
  const viewerRoles = useViewerEnrollmentRoles(courseCode)
  const isStaff = viewerIsCourseStaffEnrollment(viewerRoles)
  const hasLearnerSeat = (viewerRoles ?? []).some((r) => isStudentEquivalentEnrollmentRole(r))
  const ensureAttempted = useRef<string | null>(null)

  // If staff reloads while already in student preview, ensure the Test Student seat exists.
  useEffect(() => {
    if (!courseCode || viewAs !== 'student' || !isStaff || hasLearnerSeat) return
    if (ensureAttempted.current === courseCode) return
    ensureAttempted.current = courseCode
    void ensureCourseTestStudentEnrollment(courseCode)
      .then(() => notifyCourseViewerEnrollmentChanged(courseCode))
      .catch(() => {
        ensureAttempted.current = null
      })
  }, [courseCode, viewAs, isStaff, hasLearnerSeat])

  if (!courseCode || viewAs !== 'student' || !isStaff) {
    return null
  }

  return (
    <div
      role="status"
      aria-live="polite"
      className="flex shrink-0 items-center justify-center gap-3 border-b border-warning-border bg-warning-surface px-4 py-2 text-sm font-medium text-warning-fg print:hidden"
    >
      <GraduationCap className="h-4 w-4 shrink-0" aria-hidden />
      <span className="text-center">
        You are previewing this course as <strong className="font-semibold">Test Student</strong>.
      </span>
      <Button
        type="button"
        size="sm"
        variant="secondary"
        className="shrink-0 border border-warning-border bg-surface-raised text-warning-fg hover:bg-surface-sunken"
        onClick={() => setCourseViewAs(courseCode, 'teacher')}
      >
        Exit preview
      </Button>
    </div>
  )
}
