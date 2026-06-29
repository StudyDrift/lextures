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
import { getAccessToken } from '../lib/auth'
import {
  fetchPinnedCourses,
  putCourseCatalogPin,
  putCourseCatalogPinLayout,
  type PinnedCourseSummary,
} from '../lib/course-catalog-settings-api'
import {
  flatPinnedRows,
  normalizePinRows,
  rowsFromFlatCourses,
  rowsToCourseIds,
} from '../lib/pinned-courses-layout'
import { useCoursesRevision } from './use-inbox-unread'

const PINNED_TOOLTIP_FLASH_MS = 2600

type CoursePinnedContextValue = {
  pinnedCourses: PinnedCourseSummary[]
  pinnedRows: PinnedCourseSummary[][]
  pinnedCourseIds: Set<string>
  /** Course id that should show an instant sidebar title tooltip after pinning. */
  flashPinnedCourseId: string | null
  loading: boolean
  togglingCourseId: string | null
  refreshPinned: () => Promise<void>
  togglePin: (
    courseId: string,
    pinned: boolean,
    optimisticCourse?: PinnedCourseSummary,
  ) => Promise<void>
  reorderPinnedRows: (rows: PinnedCourseSummary[][]) => Promise<void>
}

const CoursePinnedContext = createContext<CoursePinnedContextValue | null>(null)

function toPinnedSummary(course: {
  id: string
  courseCode: string
  title: string
  heroImageUrl: string | null
  heroImageObjectPosition: string | null
  catalogNickname?: string | null
}): PinnedCourseSummary {
  return {
    id: course.id,
    courseCode: course.courseCode,
    title: course.title,
    heroImageUrl: course.heroImageUrl,
    heroImageObjectPosition: course.heroImageObjectPosition,
    catalogNickname: course.catalogNickname ?? null,
  }
}

function resolvePinnedRows(
  courses: PinnedCourseSummary[],
  rows: PinnedCourseSummary[][],
): PinnedCourseSummary[][] {
  if (rows.length > 0) return rows
  return rowsFromFlatCourses(courses)
}

export function CoursePinnedProvider({ children }: { children: ReactNode }) {
  const coursesRevision = useCoursesRevision()
  const [pinnedRows, setPinnedRows] = useState<PinnedCourseSummary[][]>([])
  const [loading, setLoading] = useState(true)
  const [togglingCourseId, setTogglingCourseId] = useState<string | null>(null)
  const [flashPinnedCourseId, setFlashPinnedCourseId] = useState<string | null>(null)
  const flashTimeoutRef = useRef<number | null>(null)
  const flashRafRef = useRef<number | null>(null)

  const pinnedCourses = useMemo(() => flatPinnedRows(pinnedRows), [pinnedRows])

  const flashPinnedTooltip = useCallback((courseId: string) => {
    if (flashTimeoutRef.current !== null) {
      window.clearTimeout(flashTimeoutRef.current)
    }
    setFlashPinnedCourseId(courseId)
    flashTimeoutRef.current = window.setTimeout(() => {
      setFlashPinnedCourseId(null)
      flashTimeoutRef.current = null
    }, PINNED_TOOLTIP_FLASH_MS)
  }, [])

  useEffect(() => {
    return () => {
      if (flashTimeoutRef.current !== null) {
        window.clearTimeout(flashTimeoutRef.current)
      }
      if (flashRafRef.current !== null) {
        cancelAnimationFrame(flashRafRef.current)
      }
    }
  }, [])

  const applyPinnedPayload = useCallback((courses: PinnedCourseSummary[], rows: PinnedCourseSummary[][]) => {
    setPinnedRows(resolvePinnedRows(courses, rows))
  }, [])

  const refreshPinned = useCallback(async () => {
    if (!getAccessToken()) {
      setPinnedRows([])
      setLoading(false)
      return
    }
    try {
      const payload = await fetchPinnedCourses()
      applyPinnedPayload(payload.courses, payload.rows)
    } catch {
      /* keep previous list */
    } finally {
      setLoading(false)
    }
  }, [applyPinnedPayload])

  useEffect(() => {
    let cancelled = false
    void (async () => {
      if (!getAccessToken()) {
        if (!cancelled) {
          setPinnedRows([])
          setLoading(false)
        }
        return
      }
      try {
        const payload = await fetchPinnedCourses()
        if (!cancelled) applyPinnedPayload(payload.courses, payload.rows)
      } catch {
        if (!cancelled) setPinnedRows([])
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [coursesRevision, refreshPinned, applyPinnedPayload])

  const reorderPinnedRows = useCallback(
    async (rows: PinnedCourseSummary[][]) => {
      const normalized = normalizePinRows(rows.filter((row) => row.length > 0))
      const previous = pinnedRows
      setPinnedRows(normalized)
      try {
        await putCourseCatalogPinLayout(rowsToCourseIds(normalized))
      } catch (err) {
        setPinnedRows(previous)
        throw err
      }
    },
    [pinnedRows],
  )

  const togglePin = useCallback(
    async (courseId: string, pinned: boolean, optimisticCourse?: PinnedCourseSummary) => {
      const previous = pinnedRows
      setTogglingCourseId(courseId)
      setPinnedRows((current) => {
        if (!pinned) {
          return normalizePinRows(current.map((row) => row.filter((course) => course.id !== courseId)))
        }
        if (current.some((row) => row.some((course) => course.id === courseId))) return current
        if (!optimisticCourse) return current
        const next = current.map((row) => [...row])
        const lastRow = next[next.length - 1]
        if (lastRow && lastRow.length < 4) {
          lastRow.push(optimisticCourse)
        } else {
          next.push([optimisticCourse])
        }
        return next
      })
      const clearFlash = () => {
        setFlashPinnedCourseId(null)
        if (flashTimeoutRef.current !== null) {
          window.clearTimeout(flashTimeoutRef.current)
          flashTimeoutRef.current = null
        }
        if (flashRafRef.current !== null) {
          cancelAnimationFrame(flashRafRef.current)
          flashRafRef.current = null
        }
      }
      if (pinned) {
        // Defer so the optimistic tile mounts before we measure the tooltip anchor.
        if (flashRafRef.current !== null) cancelAnimationFrame(flashRafRef.current)
        flashRafRef.current = requestAnimationFrame(() => {
          flashRafRef.current = null
          flashPinnedTooltip(courseId)
        })
      } else if (flashPinnedCourseId === courseId) {
        clearFlash()
      }
      try {
        await putCourseCatalogPin(courseId, pinned)
        await refreshPinned()
      } catch (err) {
        setPinnedRows(previous)
        if (pinned) clearFlash()
        throw err
      } finally {
        setTogglingCourseId(null)
      }
    },
    [pinnedRows, refreshPinned, flashPinnedCourseId, flashPinnedTooltip],
  )

  const pinnedCourseIds = useMemo(
    () => new Set(pinnedCourses.map((course) => course.id)),
    [pinnedCourses],
  )

  const value = useMemo(
    () => ({
      pinnedCourses,
      pinnedRows,
      pinnedCourseIds,
      flashPinnedCourseId,
      loading,
      togglingCourseId,
      refreshPinned,
      togglePin,
      reorderPinnedRows,
    }),
    [
      pinnedCourses,
      pinnedRows,
      pinnedCourseIds,
      flashPinnedCourseId,
      loading,
      togglingCourseId,
      refreshPinned,
      togglePin,
      reorderPinnedRows,
    ],
  )

  return <CoursePinnedContext.Provider value={value}>{children}</CoursePinnedContext.Provider>
}

export function useCoursePins() {
  const ctx = useContext(CoursePinnedContext)
  if (!ctx) {
    throw new Error('useCoursePins must be used within CoursePinnedProvider')
  }
  return ctx
}

export { toPinnedSummary }