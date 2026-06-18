import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { AlertCircle, Loader2 } from 'lucide-react'
import { fetchQuizAttemptsList } from '../../lib/courses-api'

type Props = {
  courseCode: string
  itemId: string
}

export function QuizSubmissionsPanel({ courseCode, itemId }: Props) {
  const [attempts, setAttempts] = useState<
    Awaited<ReturnType<typeof fetchQuizAttemptsList>>['attempts']
  >([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchQuizAttemptsList(courseCode, itemId)
      setAttempts(data.attempts)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load submissions.')
      setAttempts([])
    } finally {
      setLoading(false)
    }
  }, [courseCode, itemId])

  useEffect(() => {
    void load()
  }, [load])

  const ungradedCount = attempts.filter((a) => a.needsManualGrading).length
  const gradebookHref = `/courses/${encodeURIComponent(courseCode)}/gradebook?item=${encodeURIComponent(itemId)}`

  return (
    <div className="mt-6 rounded-2xl border border-slate-200/90 bg-white p-4 dark:border-neutral-600 dark:bg-neutral-950">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Submissions</h2>
          <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
            Imported and in-app quiz attempts for this item. Grade essay and other manual questions in the
            gradebook.
          </p>
        </div>
        {ungradedCount > 0 ? (
          <Link
            to={gradebookHref}
            className="shrink-0 rounded-full bg-amber-100 px-2.5 py-1 text-xs font-semibold text-amber-950 transition hover:bg-amber-200 dark:bg-amber-900/50 dark:text-amber-50 dark:hover:bg-amber-900/70"
          >
            {ungradedCount} need{ungradedCount === 1 ? 's' : ''} grading
          </Link>
        ) : null}
      </div>

      {loading ? (
        <p className="mt-4 flex items-center gap-2 text-sm text-slate-500 dark:text-neutral-400">
          <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
          Loading submissions…
        </p>
      ) : null}
      {error ? (
        <p role="alert" className="mt-4 text-sm text-rose-700 dark:text-rose-300">
          {error}
        </p>
      ) : null}
      {!loading && !error && attempts.length === 0 ? (
        <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">
          No submitted attempts yet. Re-import from Canvas with <strong>Grades</strong> enabled to pull in
          existing Canvas quiz submissions.
        </p>
      ) : null}
      {!loading && !error && attempts.length > 0 ? (
        <ul className="mt-4 divide-y divide-slate-100 dark:divide-neutral-800">
          {attempts.map((attempt) => (
            <li key={attempt.id} className="flex flex-wrap items-center justify-between gap-2 py-2.5 text-sm">
              <span className="min-w-0">
                <span className="font-medium text-slate-900 dark:text-neutral-100">
                  {attempt.studentName ?? 'Student'}
                </span>
                <span className="ms-2 text-xs text-slate-500 dark:text-neutral-400">
                  Attempt {attempt.attemptNumber}
                </span>
                {attempt.needsManualGrading ? (
                  <span className="ms-2 inline-flex items-center gap-1 rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-900 dark:bg-amber-900/40 dark:text-amber-100">
                    <AlertCircle className="h-3 w-3" aria-hidden />
                    Needs grading
                  </span>
                ) : null}
              </span>
              <span className="shrink-0 text-slate-600 dark:text-neutral-300">
                {attempt.pointsPossible > 0
                  ? `${attempt.pointsEarned}/${attempt.pointsPossible} pts`
                  : '—'}
              </span>
            </li>
          ))}
        </ul>
      ) : null}
      {attempts.length > 0 ? (
        <p className="mt-3 text-xs text-slate-500 dark:text-neutral-400">
          <Link to={gradebookHref} className="font-medium text-indigo-600 hover:underline dark:text-indigo-400">
            Open gradebook
          </Link>{' '}
          to score manual quiz questions.
        </p>
      ) : null}
    </div>
  )
}