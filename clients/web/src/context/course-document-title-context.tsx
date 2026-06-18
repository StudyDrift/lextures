/* eslint-disable react-refresh/only-export-components -- context module exports provider + hooks */
import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

type CourseDocumentTitleContextValue = {
  setPageTitle: (title: string | null) => void
}

const CourseDocumentTitleContext = createContext<CourseDocumentTitleContextValue | null>(null)

function formatCourseDocumentTitle(courseTitle: string, pageTitle: string): string {
  return `${courseTitle} | ${pageTitle}`
}

export function CourseDocumentTitleProvider({
  courseTitle,
  defaultPageTitle,
  children,
}: {
  courseTitle: string | null
  defaultPageTitle: string | null
  children: ReactNode
}) {
  const [pageTitleOverride, setPageTitleOverride] = useState<string | null>(null)

  useEffect(() => {
    setPageTitleOverride(null)
  }, [defaultPageTitle])

  const pageTitle = pageTitleOverride ?? defaultPageTitle

  useEffect(() => {
    if (courseTitle && pageTitle) {
      document.title = formatCourseDocumentTitle(courseTitle, pageTitle)
      return
    }
    if (courseTitle) {
      document.title = courseTitle
    }
  }, [courseTitle, pageTitle])

  const value = useMemo(
    () => ({
      setPageTitle: setPageTitleOverride,
    }),
    [],
  )

  return (
    <CourseDocumentTitleContext.Provider value={value}>
      {children}
    </CourseDocumentTitleContext.Provider>
  )
}

/** Override the route-derived page title for dynamic course pages (e.g. module items). */
export function useCoursePageTitle(pageTitle: string | null | undefined) {
  const ctx = useContext(CourseDocumentTitleContext)
  if (!ctx) {
    throw new Error('useCoursePageTitle must be used within CourseDocumentTitleProvider')
  }

  useEffect(() => {
    const next = pageTitle?.trim() ? pageTitle.trim() : null
    ctx.setPageTitle(next)
    return () => ctx.setPageTitle(null)
  }, [ctx, pageTitle])
}