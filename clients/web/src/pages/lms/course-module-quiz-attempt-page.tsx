import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { QuizStudentTakePanel } from '../../components/quiz/quiz-student-take-panel'
import {
  fetchModuleQuiz,
  learnerCourseItemHref,
  quizAdvancedSettingsFromPayload,
  type ModuleQuizPayload,
  type QuizAdvancedSettings,
} from '../../lib/courses-api'
import { useCoursePageTitle } from '../../context/course-document-title-context'
import { recordLastVisitedModuleItem } from '../../lib/last-visited-module-item'

export default function CourseModuleQuizAttemptPage() {
  const { courseCode, itemId } = useParams<{ courseCode: string; itemId: string }>()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [quiz, setQuiz] = useState<ModuleQuizPayload | null>(null)
  const [advanced, setAdvanced] = useState<QuizAdvancedSettings | null>(null)
  const [oneQuestionAtATime, setOneQuestionAtATime] = useState(false)

  const quizHref =
    courseCode && itemId
      ? learnerCourseItemHref(courseCode, { kind: 'quiz', id: itemId })
      : '/courses'

  const load = useCallback(async () => {
    if (!courseCode || !itemId) return
    setLoading(true)
    setLoadError(null)
    try {
      const data = await fetchModuleQuiz(courseCode, itemId)
      setQuiz(data)
      setAdvanced(quizAdvancedSettingsFromPayload(data))
      setOneQuestionAtATime(Boolean(data.oneQuestionAtATime))
      recordLastVisitedModuleItem(courseCode, {
        itemId,
        kind: 'quiz',
        title: data.title,
      })
    } catch (e) {
      setQuiz(null)
      setAdvanced(null)
      setLoadError(e instanceof Error ? e.message : 'Could not load this quiz.')
    } finally {
      setLoading(false)
    }
  }, [courseCode, itemId])

  useEffect(() => {
    void load()
  }, [load])

  useCoursePageTitle(!loading && quiz?.title ? quiz.title : null)

  if (!courseCode || !itemId) {
    return (
      <div className="px-4 py-8 text-sm text-slate-600 dark:text-neutral-400">
        Missing course or quiz.
      </div>
    )
  }

  if (loading) {
    return (
      <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 px-4 py-16 text-sm text-slate-600 dark:text-neutral-400">
        <Loader2 className="h-6 w-6 animate-spin text-indigo-600 dark:text-indigo-400" aria-hidden />
        Loading quiz…
      </div>
    )
  }

  if (loadError || !quiz || !advanced) {
    return (
      <div className="px-4 py-8 sm:px-6 md:px-8">
        <p className="text-sm text-red-700 dark:text-red-400">{loadError ?? 'Could not load this quiz.'}</p>
        <Link
          to={quizHref}
          className="mt-4 inline-block text-sm font-medium text-indigo-700 hover:text-indigo-900 dark:text-indigo-300 dark:hover:text-indigo-200"
        >
          Back to quiz
        </Link>
      </div>
    )
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <QuizStudentTakePanel
        layout="page"
        open
        onClose={() => navigate(quizHref)}
        courseCode={courseCode}
        itemId={itemId}
        quiz={quiz}
        advanced={advanced}
        oneQuestionAtATime={oneQuestionAtATime}
        allowBackNavigation={advanced.allowBackNavigation}
      />
    </div>
  )
}
