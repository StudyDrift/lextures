import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import {
  fetchEvaluationResults,
  type EvaluationResults,
  type QuestionResult,
} from '../../lib/course-evaluations-api'
import { LmsPage } from './lms-page'

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString(undefined, { dateStyle: 'medium' })
}

function RatingResultCard({ question }: { question: QuestionResult }) {
  const dist = question.distribution ?? {}
  const max = Math.max(...Object.values(dist), 1)
  const ratings = ['1', '2', '3', '4', '5']
  return (
    <div className="rounded-xl border border-border-default bg-surface-raised p-4 dark:border-border-default dark:bg-surface-raised">
      <p className="mb-3 text-sm font-medium text-fg-default">
        {question.index + 1}. {question.text}
      </p>
      {question.average != null && (
        <p className="mb-2 text-2xl font-bold text-accent-fg">
          {question.average.toFixed(1)}{' '}
          <span className="text-sm font-normal text-fg-muted">/ 5</span>
        </p>
      )}
      <div className="space-y-1.5">
        {ratings.map((r) => {
          const count = dist[r] ?? 0
          const pct = max > 0 ? (count / max) * 100 : 0
          return (
            <div key={r} className="flex items-center gap-2 text-xs text-fg-muted">
              <span className="w-4 shrink-0 text-right">{r}</span>
              <div className="h-4 flex-1 overflow-hidden rounded bg-surface-sunken">
                <div
                  className="h-full rounded bg-indigo-500 dark:bg-indigo-400"
                  style={{ width: `${pct}%` }}
                />
              </div>
              <span className="w-6 shrink-0">{count}</span>
            </div>
          )
        })}
      </div>
    </div>
  )
}

function MultipleChoiceResultCard({ question }: { question: QuestionResult }) {
  const dist = question.distribution ?? {}
  const total = Object.values(dist).reduce((s, c) => s + c, 0)
  return (
    <div className="rounded-xl border border-border-default bg-surface-raised p-4 dark:border-border-default dark:bg-surface-raised">
      <p className="mb-3 text-sm font-medium text-fg-default">
        {question.index + 1}. {question.text}
      </p>
      <div className="space-y-1.5">
        {Object.entries(dist).map(([opt, count]) => {
          const pct = total > 0 ? Math.round((count / total) * 100) : 0
          return (
            <div key={opt} className="flex items-center gap-2 text-xs text-fg-muted">
              <span className="min-w-[6rem] truncate">{opt}</span>
              <div className="h-4 flex-1 overflow-hidden rounded bg-surface-sunken">
                <div
                  className="h-full rounded bg-violet-500 dark:bg-violet-400"
                  style={{ width: `${pct}%` }}
                />
              </div>
              <span className="w-12 shrink-0 text-right">{count} ({pct}%)</span>
            </div>
          )
        })}
      </div>
    </div>
  )
}

function OpenTextResultCard({ question }: { question: QuestionResult }) {
  const texts = question.openTexts ?? []
  return (
    <div className="rounded-xl border border-border-default bg-surface-raised p-4 dark:border-border-default dark:bg-surface-raised">
      <p className="mb-3 text-sm font-medium text-fg-default">
        {question.index + 1}. {question.text}
      </p>
      <p className="mb-3 text-xs text-fg-muted">
        {texts.length} {texts.length === 1 ? 'response' : 'responses'}
      </p>
      {texts.length === 0 ? (
        <p className="text-xs italic text-fg-subtle">No responses.</p>
      ) : (
        <ul className="space-y-2">
          {texts.map((t, i) => (
            <li
              key={i}
              className="rounded-lg bg-surface-base px-3 py-2 text-xs text-fg-muted dark:bg-surface-overlay dark:text-fg-muted"
            >
              {t}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export default function CourseEvaluationResults() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const [results, setResults] = useState<EvaluationResults | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!courseCode) return
    setLoading(true)
    try {
      const r = await fetchEvaluationResults(courseCode)
      setResults(r)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load results.')
    } finally {
      setLoading(false)
    }
  }, [courseCode])

  useEffect(() => {
    load()
  }, [load])

  if (loading) {
    return (
      <LmsPage title="Evaluation Results">
        <div className="flex items-center justify-center py-20">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-indigo-500 border-t-transparent" />
        </div>
      </LmsPage>
    )
  }

  if (error) {
    return (
      <LmsPage title="Evaluation Results">
        <div className="mx-auto max-w-xl py-16 text-center">
          <p className="text-danger-fg">{error}</p>
        </div>
      </LmsPage>
    )
  }

  if (!results) {
    return (
      <LmsPage title="Evaluation Results">
        <div className="mx-auto max-w-xl py-16 text-center">
          <p className="text-fg-muted">No evaluation results found.</p>
        </div>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Evaluation Results">
      <div className="mx-auto max-w-3xl px-4 py-8">
        {/* Summary header */}
        <div className="mb-6 grid grid-cols-3 gap-4">
          <div className="rounded-xl border border-border-default bg-surface-raised p-4 text-center dark:border-border-default dark:bg-surface-raised">
            <p className="text-2xl font-bold text-fg-default">
              {results.responseCount}
            </p>
            <p className="text-xs text-fg-muted">Responses</p>
          </div>
          <div className="rounded-xl border border-border-default bg-surface-raised p-4 text-center dark:border-border-default dark:bg-surface-raised">
            <p className="text-2xl font-bold text-fg-default">
              {results.enrolledCount}
            </p>
            <p className="text-xs text-fg-muted">Enrolled students</p>
          </div>
          <div className="rounded-xl border border-border-default bg-surface-raised p-4 text-center dark:border-border-default dark:bg-surface-raised">
            <p className="text-2xl font-bold text-fg-default">
              {results.completionPct.toFixed(0)}%
            </p>
            <p className="text-xs text-fg-muted">Completion rate</p>
          </div>
        </div>

        <p className="mb-6 text-xs text-fg-muted">
          Window: {formatDate(results.opensAt)} – {formatDate(results.closesAt)}
        </p>

        {!results.meetsThreshold ? (
          <div className="rounded-xl border border-amber-200 bg-amber-50 p-6 text-center dark:border-amber-800 dark:bg-amber-950/30">
            <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
              Not enough responses to display results
            </p>
            <p className="mt-1 text-xs text-warning-fg dark:text-amber-400">
              At least 5 responses are required to protect anonymity.
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {results.questions.map((q) => {
              if (q.type === 'rating') {
                return <RatingResultCard key={q.index} question={q} />
              }
              if (q.type === 'multiple_choice') {
                return <MultipleChoiceResultCard key={q.index} question={q} />
              }
              return <OpenTextResultCard key={q.index} question={q} />
            })}
          </div>
        )}
      </div>
    </LmsPage>
  )
}
