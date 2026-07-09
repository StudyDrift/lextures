/* eslint-disable react-refresh/only-export-components -- context module exports provider + hooks */
import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { putCourseCatalogHidden } from '../lib/course-catalog-settings-api'
import { useCoursePins } from './course-pinned-context'

type CourseHiddenContextValue = {
  togglingCourseId: string | null
  toggleHidden: (courseId: string, hidden: boolean, wasPinned?: boolean) => Promise<void>
}

const CourseHiddenContext = createContext<CourseHiddenContextValue | null>(null)

export function CourseHiddenProvider({ children }: { children: ReactNode }) {
  const { togglePin } = useCoursePins()
  const [togglingCourseId, setTogglingCourseId] = useState<string | null>(null)

  const toggleHidden = useCallback(
    async (courseId: string, hidden: boolean, wasPinned?: boolean) => {
      setTogglingCourseId(courseId)
      try {
        await putCourseCatalogHidden(courseId, hidden)
        if (hidden && wasPinned) {
          await togglePin(courseId, false)
        }
      } finally {
        setTogglingCourseId(null)
      }
    },
    [togglePin],
  )

  const value = useMemo(
    () => ({
      togglingCourseId,
      toggleHidden,
    }),
    [togglingCourseId, toggleHidden],
  )

  return <CourseHiddenContext.Provider value={value}>{children}</CourseHiddenContext.Provider>
}

export function useCourseHidden() {
  const ctx = useContext(CourseHiddenContext)
  if (!ctx) {
    throw new Error('useCourseHidden must be used within CourseHiddenProvider')
  }
  return ctx
}