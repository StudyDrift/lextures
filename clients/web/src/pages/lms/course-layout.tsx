import { useEffect, useMemo, useState } from 'react'
import { Link, Outlet, useParams } from 'react-router-dom'
import { TutorPanel } from '../../components/tutor-panel'
import { CourseLiveContext } from '../../context/course-live-context'
import { useCourseNavFeatures } from '../../context/course-nav-features-context'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { useCourseStructureRevision } from '../../hooks/use-course-structure-ws'
import { fetchEvaluationStatus } from '../../lib/course-evaluations-api'
import { CourseSyllabusAcceptanceOverlay } from './course-syllabus-acceptance-overlay'

function EvaluationReminderBanner({ courseCode }: { courseCode: string }) {
  const [show, setShow] = useState(false)

  useEffect(() => {
    let cancelled = false
    fetchEvaluationStatus(courseCode)
      .then((s) => {
        if (!cancelled && s.windowOpen && !s.hasSubmitted) {
          setShow(true)
        }
      })
      .catch(() => {/* ignore */})
    return () => { cancelled = true }
  }, [courseCode])

  if (!show) return null

  return (
    <div
      role="alert"
      aria-live="polite"
      className="flex items-center justify-between gap-3 bg-indigo-600 px-4 py-2 text-sm text-white dark:bg-indigo-700"
    >
      <span>
        Your course evaluation is open. Your feedback is anonymous and helps improve this course.
      </span>
      <Link
        to={`/courses/${courseCode}/evaluation`}
        className="shrink-0 rounded-lg bg-white/20 px-3 py-1 text-xs font-semibold hover:bg-white/30"
      >
        Submit evaluation
      </Link>
    </div>
  )
}

/**
 * Wraps all routes under `/courses/:courseCode` so syllabus acceptance applies on first visit
 * to any course page, not only the overview.
 */
export default function CourseLayout() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const { aiTutorEnabled } = useCourseNavFeatures()
  const { ffCourseEvaluations } = usePlatformFeatures()
  const structureRevision = useCourseStructureRevision(courseCode)
  const liveValue = useMemo(
    () => ({ structureRevision }),
    [structureRevision],
  )

  return (
    <CourseLiveContext.Provider value={liveValue}>
      {courseCode ? <CourseSyllabusAcceptanceOverlay courseCode={courseCode} /> : null}
      {courseCode && ffCourseEvaluations ? (
        <EvaluationReminderBanner courseCode={courseCode} />
      ) : null}
      <Outlet />
      {courseCode && aiTutorEnabled ? <TutorPanel courseCode={courseCode} /> : null}
    </CourseLiveContext.Provider>
  )
}
