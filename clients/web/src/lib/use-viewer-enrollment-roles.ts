import { useCallback, useEffect, useState } from 'react'
import { fetchCourse } from './courses-api'
import { COURSE_VIEWER_ENROLLMENTS_CHANGED } from './course-view-as'

export type ViewerEnrollmentRolesState = {
  /** `null` until the course loads (or when loading failed). */
  roles: string[] | null
  /** Set when the course request failed or timed out; cleared when a retry starts. */
  error: string | null
  /** Refetches the course. */
  retry: () => void
}

/**
 * Loads `viewerEnrollmentRoles` for the course and refetches when enrollment
 * changes for the signed-in user (e.g. self-enroll as student) without a full page reload.
 * Also reports load failures so pages can show an error with Retry instead of a blank screen.
 */
export function useViewerEnrollmentRolesState(
  courseCode: string | null | undefined,
): ViewerEnrollmentRolesState {
  const [viewerRoles, setViewerRoles] = useState<string[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [retryNonce, setRetryNonce] = useState(0)

  useEffect(() => {
    if (!courseCode || courseCode === 'create') {
      setViewerRoles(null)
      setError(null)
      return
    }
    let cancelled = false
    let gen = 0
    const run = () => {
      const id = ++gen
      void (async () => {
        try {
          const c = await fetchCourse(courseCode)
          if (cancelled || id !== gen) return
          setViewerRoles(c.viewerEnrollmentRoles ?? [])
          setError(null)
        } catch (e) {
          if (cancelled || id !== gen) return
          setViewerRoles(null)
          setError(e instanceof Error && e.message ? e.message : 'Could not load this course.')
        }
      })()
    }
    run()
    function onEnrollmentChanged(e: Event) {
      const ce = e as CustomEvent<{ courseCode?: string }>
      if (ce.detail?.courseCode === courseCode) run()
    }
    window.addEventListener(COURSE_VIEWER_ENROLLMENTS_CHANGED, onEnrollmentChanged)
    return () => {
      cancelled = true
      window.removeEventListener(COURSE_VIEWER_ENROLLMENTS_CHANGED, onEnrollmentChanged)
    }
  }, [courseCode, retryNonce])

  const retry = useCallback(() => {
    setError(null)
    setRetryNonce((n) => n + 1)
  }, [])

  return { roles: viewerRoles, error, retry }
}

/** Roles only; see {@link useViewerEnrollmentRolesState} for load errors and retry. */
export function useViewerEnrollmentRoles(courseCode: string | null | undefined): string[] | null {
  return useViewerEnrollmentRolesState(courseCode).roles
}
