import { useEffect, useState } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { LinkButton } from '../ui'
import {
  adjacentCourseActivities,
  type CourseActivityNeighbors,
} from '../../lib/course-activity-neighbors'
import { fetchCourseStructure } from '../../lib/courses-api'

type CourseActivityPagerProps = {
  courseCode: string
  itemId: string
}

/**
 * Previous and next activities in outline order, labeled with each activity's name.
 */
export function CourseActivityPager({ courseCode, itemId }: CourseActivityPagerProps) {
  const [neighbors, setNeighbors] = useState<CourseActivityNeighbors | null>(null)

  useEffect(() => {
    let cancelled = false
    setNeighbors(null)
    void (async () => {
      try {
        const items = await fetchCourseStructure(courseCode)
        if (cancelled) return
        setNeighbors(adjacentCourseActivities(items, courseCode, itemId))
      } catch {
        if (!cancelled) setNeighbors({ previous: null, next: null })
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode, itemId])

  if (!neighbors?.previous && !neighbors?.next) return null

  const linkClass = 'w-full min-w-0 overflow-hidden sm:max-w-md'

  return (
    <nav
      aria-label="Activity"
      className="mt-10 flex flex-col gap-3 border-t border-border-default pt-6 sm:flex-row sm:items-center sm:justify-between"
    >
      {neighbors.previous ? (
        <LinkButton
          to={neighbors.previous.href}
          variant="secondary"
          className={linkClass}
        >
          <ChevronLeft className="h-4 w-4 shrink-0" aria-hidden />
          <span className="min-w-0 truncate">Previous · {neighbors.previous.title}</span>
        </LinkButton>
      ) : null}
      {neighbors.next ? (
        <LinkButton
          to={neighbors.next.href}
          variant="primary"
          className={`${linkClass}${neighbors.previous ? '' : ' sm:ms-auto'}`}
        >
          <span className="min-w-0 truncate">Next · {neighbors.next.title}</span>
          <ChevronRight className="h-4 w-4 shrink-0" aria-hidden />
        </LinkButton>
      ) : null}
    </nav>
  )
}
