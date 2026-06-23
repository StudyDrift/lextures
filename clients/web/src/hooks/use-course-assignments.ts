import { useEffect, useState } from 'react'
import { fetchCourseStructure } from '../lib/courses-api'
import {
  sortAssignmentsByTitle,
  type CourseAssignmentOption,
} from '../components/annotation/grader-agent/activity-node-data'

export function useCourseAssignments(courseCode: string, enabled: boolean) {
  const [assignments, setAssignments] = useState<CourseAssignmentOption[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!enabled || !courseCode.trim()) {
      setAssignments([])
      setLoading(false)
      setError(null)
      return
    }

    let cancelled = false
    setLoading(true)
    setError(null)

    void fetchCourseStructure(courseCode)
      .then((items) => {
        if (cancelled) return
        const next = sortAssignmentsByTitle(
          items
            .filter((item) => item.kind === 'assignment')
            .map((item) => ({ id: item.id, title: item.title.trim() || 'Untitled assignment' })),
        )
        setAssignments(next)
      })
      .catch((e: unknown) => {
        if (cancelled) return
        setAssignments([])
        setError(e instanceof Error ? e.message : 'Could not load assignments.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [courseCode, enabled])

  return { assignments, loading, error }
}