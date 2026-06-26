import { useEffect, useState } from 'react'
import { fetchCourseStructure } from '../lib/courses-api'
import {
  quizOptionsFromStructure,
  type CourseQuizOption,
} from '../components/annotation/grader-agent/course-quiz-options'

export function useCourseQuizzes(courseCode: string, enabled: boolean) {
  const [quizzes, setQuizzes] = useState<CourseQuizOption[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!enabled || !courseCode.trim()) {
      setQuizzes([])
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
        setQuizzes(quizOptionsFromStructure(items))
      })
      .catch((e: unknown) => {
        if (cancelled) return
        setQuizzes([])
        setError(e instanceof Error ? e.message : 'Could not load quizzes.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [courseCode, enabled])

  return { quizzes, loading, error }
}