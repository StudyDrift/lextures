/* eslint-disable react-refresh/only-export-components -- context module exports provider + hooks */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { matchPath, useLocation } from 'react-router-dom'
import { fetchCourseChecklistSummary } from '../lib/course-checklist-api'
import type { ChecklistSummary } from '../lib/course-checklist-api-schemas'
import { subscribeChecklistInvalidation } from '../lib/course-checklist-invalidate'
import { courseItemCreatePermission } from '../lib/courses-api'
import { useCourseViewAs } from '../lib/course-view-as'
import { usePermissions } from './use-permissions'

const MEMO_MS = 60_000

type CourseChecklistSummaryContextValue = {
  summary: ChecklistSummary | null
  loading: boolean
  refresh: () => Promise<void>
  canManageCourse: boolean
}

const CourseChecklistSummaryContext = createContext<CourseChecklistSummaryContextValue>({
  summary: null,
  loading: false,
  refresh: async () => {},
  canManageCourse: false,
})

type MemoEntry = {
  summary: ChecklistSummary
  fetchedAt: number
  catalogVersion: string
}

const memoByCourse = new Map<string, MemoEntry>()

function activeCourseCodeFromPath(pathname: string): string | null {
  const m = matchPath({ path: '/courses/:courseCode/*', end: false }, pathname)
  const raw = m?.params.courseCode
  if (!raw || raw === 'create') return null
  return raw
}

export function CourseChecklistSummaryProvider({ children }: { children: ReactNode }) {
  const location = useLocation()
  const courseCode = activeCourseCodeFromPath(location.pathname)
  const { allows, loading: permLoading } = usePermissions()
  const courseViewPreview = useCourseViewAs(courseCode ?? undefined)
  const canManageCourse =
    !!courseCode && !permLoading && allows(courseItemCreatePermission(courseCode))
  const showChecklist = canManageCourse && courseViewPreview !== 'student'

  const [summary, setSummary] = useState<ChecklistSummary | null>(null)
  const [loading, setLoading] = useState(false)
  const lastFocusIdleAt = useRef(Date.now())
  const inFlight = useRef<Promise<void> | null>(null)

  const load = useCallback(
    async (code: string, opts?: { force?: boolean }) => {
      if (!opts?.force) {
        const cached = memoByCourse.get(code)
        if (cached && Date.now() - cached.fetchedAt < MEMO_MS) {
          setSummary(cached.summary)
          setLoading(false)
          return
        }
      }

      if (inFlight.current) {
        await inFlight.current
        return
      }

      setLoading(true)
      const run = (async () => {
        try {
          const next = await fetchCourseChecklistSummary(code)
          memoByCourse.set(code, {
            summary: next,
            fetchedAt: Date.now(),
            catalogVersion: `${next.outstandingEssential}:${next.done}:${next.total}:${next.computedAt}`,
          })
          setSummary(next)
        } catch (err) {
          if (import.meta.env.DEV) {
            console.warn('[course-checklist] summary fetch failed', err)
          }
          // Silent failure: no badge, no toast (FR-11).
          setSummary(null)
        } finally {
          setLoading(false)
          inFlight.current = null
        }
      })()
      inFlight.current = run
      await run
    },
    [],
  )

  const refresh = useCallback(async () => {
    if (!courseCode || !showChecklist) return
    memoByCourse.delete(courseCode)
    await load(courseCode, { force: true })
  }, [courseCode, load, showChecklist])

  useEffect(() => {
    if (!courseCode || !showChecklist) {
      setSummary(null)
      setLoading(false)
      return
    }
    void load(courseCode)
  }, [courseCode, showChecklist, load])

  useEffect(() => {
    function onFocus() {
      if (!courseCode || !showChecklist) return
      const idle = Date.now() - lastFocusIdleAt.current
      if (idle > MEMO_MS) {
        void load(courseCode, { force: true })
      }
    }
    function onBlur() {
      lastFocusIdleAt.current = Date.now()
    }
    window.addEventListener('focus', onFocus)
    window.addEventListener('blur', onBlur)
    return () => {
      window.removeEventListener('focus', onFocus)
      window.removeEventListener('blur', onBlur)
    }
  }, [courseCode, showChecklist, load])

  useEffect(() => {
    return subscribeChecklistInvalidation((code) => {
      memoByCourse.delete(code)
      if (code === courseCode && showChecklist) {
        void load(code, { force: true })
      }
    })
  }, [courseCode, showChecklist, load])

  const value = useMemo(
    () => ({
      summary: showChecklist ? summary : null,
      loading: showChecklist && (permLoading || loading),
      refresh,
      canManageCourse: showChecklist,
    }),
    [summary, loading, permLoading, refresh, showChecklist],
  )

  return (
    <CourseChecklistSummaryContext.Provider value={value}>
      {children}
    </CourseChecklistSummaryContext.Provider>
  )
}

export function useCourseChecklistSummary(): CourseChecklistSummaryContextValue {
  return useContext(CourseChecklistSummaryContext)
}

/** Bust the in-memory memo and refresh when the viewer can manage the course. */
export function useInvalidateChecklist(): (courseCode?: string) => void {
  const { refresh } = useCourseChecklistSummary()
  const location = useLocation()
  return useCallback(
    (courseCode?: string) => {
      const code = courseCode ?? activeCourseCodeFromPath(location.pathname)
      if (!code) return
      memoByCourse.delete(code)
      void refresh()
    },
    [location.pathname, refresh],
  )
}

export function clearChecklistSummaryMemoForTests(): void {
  memoByCourse.clear()
}
