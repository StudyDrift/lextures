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
  type PinnedCourseSummary,
} from '../lib/course-catalog-settings-api'
import { useCoursesRevision } from './use-inbox-unread'

const PINNED_TOOLTIP_FLASH_MS = 2600

type CoursePinnedContextValue = {
  pinnedCourses: PinnedCourseSummary[]
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

export function CoursePinnedProvider({ children }: { children: ReactNode }) {
  const coursesRevision = useCoursesRevision()
  const [pinnedCourses, setPinnedCourses] = useState<PinnedCourseSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [togglingCourseId, setTogglingCourseId] = useState<string | null>(null)
  const [flashPinnedCourseId, setFlashPinnedCourseId] = useState<string | null>(null)
  const flashTimeoutRef = useRef<number | null>(null)
  const flashRafRef = useRef<number | null>(null)

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

  const refreshPinned = useCallback(async () => {
    if (!getAccessToken()) {
      setPinnedCourses([])
      setLoading(false)
      return
    }
    try {
      const courses = await fetchPinnedCourses()
      setPinnedCourses(courses)
    } catch {
      /* keep previous list */
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    let cancelled = false
    void (async () => {
      if (!getAccessToken()) {
        if (!cancelled) {
          setPinnedCourses([])
          setLoading(false)
        }
        return
      }
      try {
        const courses = await fetchPinnedCourses()
        if (!cancelled) setPinnedCourses(courses)
      } catch {
        if (!cancelled) setPinnedCourses([])
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [coursesRevision, refreshPinned])

  const togglePin = useCallback(
    async (courseId: string, pinned: boolean, optimisticCourse?: PinnedCourseSummary) => {
      const previous = pinnedCourses
      setTogglingCourseId(courseId)
      setPinnedCourses((current) => {
        if (!pinned) return current.filter((course) => course.id !== courseId)
        if (current.some((course) => course.id === courseId)) return current
        if (!optimisticCourse) return current
        return [...current, optimisticCourse]
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
        setPinnedCourses(previous)
        if (pinned) clearFlash()
        throw err
      } finally {
        setTogglingCourseId(null)
      }
    },
    [pinnedCourses, refreshPinned, flashPinnedCourseId, flashPinnedTooltip],
  )

  const pinnedCourseIds = useMemo(
    () => new Set(pinnedCourses.map((course) => course.id)),
    [pinnedCourses],
  )

  const value = useMemo(
    () => ({
      pinnedCourses,
      pinnedCourseIds,
      flashPinnedCourseId,
      loading,
      togglingCourseId,
      refreshPinned,
      togglePin,
    }),
    [pinnedCourses, pinnedCourseIds, flashPinnedCourseId, loading, togglingCourseId, refreshPinned, togglePin],
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