import { useEffect, useState } from 'react'
import { LinkButton } from '../ui'
import {
  fetchStudentProgress,
  type StudentProgressSummary,
} from '../../lib/student-progress-api'

export type LearnerProgressCardModel = {
  enrollmentId: string
  name: string
  sectionLabel?: string | null
}

function formatPercent(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return '—'
  return `${Math.round(value)}%`
}

export function LearnerProgressCards({
  courseCode,
  learners,
  loadSummaries,
}: {
  courseCode: string
  learners: LearnerProgressCardModel[]
  loadSummaries: boolean
}) {
  const [summaries, setSummaries] = useState<Record<string, StudentProgressSummary | null>>({})
  const enrollmentKey = learners.map((learner) => learner.enrollmentId).join('\n')

  useEffect(() => {
    if (!loadSummaries || enrollmentKey === '') {
      setSummaries({})
      return
    }
    const ids = enrollmentKey.split('\n')
    let cancelled = false
    void Promise.all(
      ids.map(async (enrollmentId) => {
        try {
          const progress = await fetchStudentProgress(courseCode, enrollmentId)
          return [enrollmentId, progress.summary] as const
        } catch {
          return [enrollmentId, null] as const
        }
      }),
    ).then((rows) => {
      if (cancelled) return
      setSummaries(Object.fromEntries(rows))
    })
    return () => {
      cancelled = true
    }
  }, [courseCode, enrollmentKey, loadSummaries])

  return (
    <ul className="mt-6 grid gap-4 sm:grid-cols-2">
      {learners.map((learner) => {
        const summary = summaries[learner.enrollmentId]
        const progressPath = `/courses/${encodeURIComponent(courseCode)}/students/${encodeURIComponent(learner.enrollmentId)}/progress`
        return (
          <li
            key={learner.enrollmentId}
            className="flex flex-col gap-3 rounded-2xl border border-border-default bg-surface-raised p-4 shadow-sm"
          >
            <div>
              <h2 className="text-base font-semibold text-fg-default">{learner.name}</h2>
              {learner.sectionLabel ? (
                <p className="mt-1 text-xs text-fg-muted">{learner.sectionLabel}</p>
              ) : null}
            </div>
            {loadSummaries ? (
              <dl className="grid grid-cols-3 gap-2 text-sm">
                <div>
                  <dt className="text-xs text-fg-muted">Average</dt>
                  <dd className="font-medium tabular-nums text-fg-default">
                    {formatPercent(summary?.avgGradePercent)}
                  </dd>
                </div>
                <div>
                  <dt className="text-xs text-fg-muted">Turned in</dt>
                  <dd className="font-medium tabular-nums text-fg-default">
                    {summary ? `${Math.round(summary.assignmentsSubmittedPct)}%` : '—'}
                  </dd>
                </div>
                <div>
                  <dt className="text-xs text-fg-muted">Missing</dt>
                  <dd className="font-medium tabular-nums text-fg-default">
                    {summary ? summary.missingCount : '—'}
                  </dd>
                </div>
              </dl>
            ) : (
              <p className="text-sm text-fg-muted">Open progress to see scores and missing work.</p>
            )}
            <div className="mt-auto">
              <LinkButton to={progressPath} variant="secondary" size="sm">
                See progress
              </LinkButton>
            </div>
          </li>
        )
      })}
    </ul>
  )
}
