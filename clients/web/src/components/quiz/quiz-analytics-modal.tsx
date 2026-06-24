import { useCallback, useEffect, useId, useState } from 'react'
import { RefreshCw, X } from 'lucide-react'
import {
  fetchQuizAnalytics,
  type QuizAnalyticsReport,
  type QuizFocusAttemptStat,
  type QuizQuestionStat,
  type QuizScoreBucket,
} from '../../lib/quiz-analytics-api'
import { QuizItemAnalysisPanel } from './quiz-item-analysis-panel'

type Props = {
  open: boolean
  onClose: () => void
  courseCode: string
  itemId: string
  quizTitle: string
}

export function QuizAnalyticsModal({ open, onClose, courseCode, itemId, quizTitle }: Props) {
  const titleId = useId()
  const [report, setReport] = useState<QuizAnalyticsReport | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchQuizAnalytics(courseCode, itemId)
      setReport(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load analytics.')
      setReport(null)
    } finally {
      setLoading(false)
    }
  }, [courseCode, itemId])

  useEffect(() => {
    if (!open) return
    void load()
  }, [load, open])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose, open])

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/40 p-4 sm:items-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="flex max-h-[min(90vh,56rem)] w-full max-w-3xl flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-600 dark:bg-neutral-950">
        <div className="flex shrink-0 items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
          <div>
            <h2 id={titleId} className="text-base font-semibold text-slate-900 dark:text-neutral-100">
              Quiz analytics
            </h2>
            <p className="mt-0.5 text-sm text-slate-500 dark:text-neutral-400">{quizTitle}</p>
          </div>
          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => void load()}
              disabled={loading}
              aria-label="Refresh analytics"
              className="rounded-lg p-1.5 text-slate-500 transition-[background-color,color,border-color] hover:bg-slate-100 hover:text-slate-800 disabled:opacity-50 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
            >
              <RefreshCw className={`h-5 w-5 ${loading ? 'animate-spin' : ''}`} aria-hidden />
            </button>
            <button
              type="button"
              onClick={onClose}
              aria-label="Close analytics"
              className="rounded-lg p-1.5 text-slate-500 transition-[background-color,color,border-color] hover:bg-slate-100 hover:text-slate-800 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
            >
              <X className="h-5 w-5" aria-hidden />
            </button>
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4 sm:px-6">
          {error ? (
            <p role="alert" className="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-300">
              {error}
            </p>
          ) : null}

          {loading && !report ? (
            <p className="text-sm text-slate-500 dark:text-neutral-400">Loading analytics…</p>
          ) : null}

          {report ? (
            <div className="space-y-8">
              <AnalyticsSummary report={report} />
              <ScoreDistributionChart buckets={report.scoreBuckets} nAttempts={report.nAttempts} />
              <FocusLossSection attempts={report.focusAttempts} />
              <QuestionPerformanceSection questions={report.questionStats} />
              <QuizItemAnalysisPanel courseCode={courseCode} itemId={itemId} />
            </div>
          ) : null}
        </div>
      </div>
    </div>
  )
}

function AnalyticsSummary({ report }: { report: QuizAnalyticsReport }) {
  return (
    <dl className="grid grid-cols-2 gap-3 sm:grid-cols-3">
      <SummaryCard label="Submitted attempts" value={String(report.nAttempts)} />
      <SummaryCard
        label="Mean score"
        value={report.meanScore != null ? `${report.meanScore.toFixed(1)}%` : '—'}
      />
      <SummaryCard
        label="Questions"
        value={String(report.questionStats.length)}
        className="col-span-2 sm:col-span-1"
      />
    </dl>
  )
}

function SummaryCard({
  label,
  value,
  className = '',
}: {
  label: string
  value: string
  className?: string
}) {
  return (
    <div
      className={`rounded-xl border border-slate-100 bg-slate-50 px-3 py-2.5 dark:border-neutral-700 dark:bg-neutral-900 ${className}`}
    >
      <dt className="text-[11px] uppercase tracking-wide text-slate-500 dark:text-neutral-400">
        {label}
      </dt>
      <dd className="mt-0.5 text-lg font-semibold tabular-nums text-slate-900 dark:text-neutral-100">
        {value}
      </dd>
    </div>
  )
}

function ScoreDistributionChart({
  buckets,
  nAttempts,
}: {
  buckets: QuizScoreBucket[]
  nAttempts: number
}) {
  const captionId = useId()
  const maxCount = Math.max(1, ...buckets.map((b) => b.count))
  const chartBarMaxHeight = 112

  if (nAttempts === 0) {
    return (
      <section aria-labelledby={captionId}>
        <h3 id={captionId} className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          Overall score distribution
        </h3>
        <p className="mt-3 text-sm text-slate-500 dark:text-neutral-400">
          No submitted attempts yet.
        </p>
      </section>
    )
  }

  return (
    <section aria-labelledby={captionId}>
      <h3 id={captionId} className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
        Overall score distribution
      </h3>
      <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
        Percent of total score across {nAttempts} submitted attempt{nAttempts === 1 ? '' : 's'}.
      </p>
      <figure className="mt-4">
        <div
          role="img"
          aria-label={`Score histogram across ${nAttempts} attempts`}
          className="flex h-40 items-end gap-1.5 sm:gap-2"
        >
          {buckets.map((bucket) => {
            return (
              <div key={bucket.label} className="flex min-w-0 flex-1 flex-col items-center justify-end gap-1">
                <span className="text-[10px] tabular-nums text-slate-500 dark:text-neutral-400">
                  {bucket.count > 0 ? bucket.count : ''}
                </span>
                <div
                  title={`${bucket.label}: ${bucket.count} attempt${bucket.count === 1 ? '' : 's'}`}
                  className="w-full rounded-t-md bg-indigo-500 dark:bg-indigo-400"
                  style={{
                    height:
                      bucket.count > 0
                        ? `${Math.max((bucket.count / maxCount) * chartBarMaxHeight, 6)}px`
                        : 0,
                  }}
                />
                <span className="max-w-full truncate text-[10px] text-slate-600 dark:text-neutral-400">
                  {bucket.min}
                </span>
              </div>
            )
          })}
        </div>
        <figcaption className="sr-only">
          Histogram of quiz scores in 10-point buckets from 0% to 100%
        </figcaption>
      </figure>
      <table className="mt-4 w-full text-sm">
        <caption className="text-start text-xs font-medium text-slate-600 dark:text-neutral-400">
          Score distribution data
        </caption>
        <thead>
          <tr className="border-b border-slate-200 dark:border-neutral-700">
            <th scope="col" className="py-1 pe-3 text-start font-medium">
              Score range
            </th>
            <th scope="col" className="py-1 text-end font-medium">
              Attempts
            </th>
          </tr>
        </thead>
        <tbody>
          {buckets.map((bucket) => (
            <tr key={bucket.label} className="border-b border-slate-100 dark:border-neutral-800">
              <td className="py-1 pe-3">{bucket.label}</td>
              <td className="py-1 text-end tabular-nums">{bucket.count}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  )
}

function FocusLossSection({ attempts }: { attempts: QuizFocusAttemptStat[] }) {
  const headingId = useId()

  if (attempts.length === 0) {
    return (
      <section aria-labelledby={headingId}>
        <h3 id={headingId} className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          Focus-loss events
        </h3>
        <p className="mt-3 text-sm text-slate-500 dark:text-neutral-400">
          No focus-loss events recorded on submitted attempts.
        </p>
      </section>
    )
  }

  return (
    <section aria-labelledby={headingId}>
      <h3 id={headingId} className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
        Focus-loss events
      </h3>
      <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
        Tab switches and window blur events logged while learners had a quiz in progress.
      </p>
      <table className="mt-4 w-full text-sm">
        <caption className="sr-only">Focus-loss events by attempt</caption>
        <thead>
          <tr className="border-b border-slate-200 dark:border-neutral-700">
            <th scope="col" className="py-1 pe-3 text-start font-medium">
              Attempt
            </th>
            <th scope="col" className="py-1 pe-3 text-end font-medium">
              Events
            </th>
            <th scope="col" className="py-1 text-end font-medium">
              Flagged
            </th>
          </tr>
        </thead>
        <tbody>
          {attempts.map((a) => (
            <tr key={a.attemptId} className="border-b border-slate-100 dark:border-neutral-800">
              <td className="py-1.5 pe-3 tabular-nums">#{a.attemptNumber}</td>
              <td className="py-1.5 pe-3 text-end tabular-nums">{a.eventCount}</td>
              <td className="py-1.5 text-end">{a.academicIntegrityFlag ? 'Yes' : '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  )
}

function QuestionPerformanceSection({ questions }: { questions: QuizQuestionStat[] }) {
  const headingId = useId()

  if (questions.length === 0) {
    return (
      <section aria-labelledby={headingId}>
        <h3 id={headingId} className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          Question performance
        </h3>
        <p className="mt-3 text-sm text-slate-500 dark:text-neutral-400">
          No graded responses yet.
        </p>
      </section>
    )
  }

  return (
    <section aria-labelledby={headingId}>
      <h3 id={headingId} className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
        Question performance
      </h3>
      <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
        Average score earned on each question, shown as percent correct.
      </p>
      <ul className="mt-4 space-y-4">
        {questions.map((q) => (
          <li key={q.questionIndex}>
            <QuestionPerformanceRow question={q} />
          </li>
        ))}
      </ul>
    </section>
  )
}

function QuestionPerformanceRow({ question }: { question: QuizQuestionStat }) {
  const truncated =
    question.questionText.length > 120
      ? `${question.questionText.slice(0, 120)}…`
      : question.questionText
  const pct = Math.round(question.pctCorrect)
  const barColor =
    pct < 40 ? 'bg-rose-500' : pct < 70 ? 'bg-amber-400' : 'bg-emerald-500'

  return (
    <div>
      <div className="flex items-start justify-between gap-3">
        <p className="text-sm text-slate-800 dark:text-neutral-200">
          <span className="font-medium text-slate-500 dark:text-neutral-400">
            Q{question.questionIndex + 1}.
          </span>{' '}
          {truncated || <span className="italic text-slate-400">No question text</span>}
        </p>
        <span className="shrink-0 text-sm font-semibold tabular-nums text-slate-900 dark:text-neutral-100">
          {question.pctCorrect.toFixed(1)}%
        </span>
      </div>
      <div
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={pct}
        aria-label={`Question ${question.questionIndex + 1}: ${question.pctCorrect.toFixed(1)}% correct from ${question.nResponses} responses`}
        className="mt-2 h-2.5 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-neutral-800"
      >
        <div className={`h-full rounded-full ${barColor}`} style={{ width: `${pct}%` }} />
      </div>
      <p className="mt-1 text-[11px] text-slate-500 dark:text-neutral-400">
        {question.nResponses} response{question.nResponses === 1 ? '' : 's'}
      </p>
    </div>
  )
}
