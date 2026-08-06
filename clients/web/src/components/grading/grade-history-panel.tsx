import { useMemo } from 'react'
import type { GradeHistoryEvent } from '../../lib/courses-api'
import { LocaleTime } from '../ui/locale-time'

function actionLabel(a: string): string {
  switch (a) {
    case 'created':
      return 'Score entered'
    case 'updated':
      return 'Score or rubric changed'
    case 'posted':
      return 'Grade released to students'
    case 'retracted':
      return 'Grade taken back (hidden)'
    case 'deleted':
      return 'Score cleared'
    case 'excused':
      return 'Excused'
    case 'unexcused':
      return 'Unexcused'
    case 'revision_requested':
      return 'Revision requested'
    case 'resubmission_received':
      return 'Resubmission received'
    default:
      return a.replaceAll('_', ' ')
  }
}

/**
 * Read-only list of grade audit events (plan 3.10). Parent supplies modal chrome if needed.
 */
export function GradeHistoryPanel({
  title,
  events,
  loading,
  error,
}: {
  title: string
  events: GradeHistoryEvent[] | null
  loading: boolean
  error: string | null
}) {
  const id = useMemo(
    () => `grade-hist-${title.replace(/\s+/g, '-').slice(0, 40)}`,
    [title],
  )
  if (loading) {
    return <p className="text-sm text-fg-muted">Loading history…</p>
  }
  if (error) {
    return (
      <p className="text-sm text-danger-fg" role="alert">
        {error}
      </p>
    )
  }
  if (!events || events.length === 0) {
    return (
      <p className="text-sm text-fg-muted">
        No changes recorded for this cell yet.
      </p>
    )
  }
  return (
    <div>
      <h3 className="text-base font-semibold text-fg-default" id={id}>
        {title}
      </h3>
      <p className="mt-0.5 text-xs text-fg-subtle">
        Who changed the score, when, and (if your instructor provided one) why. Older backfill rows
        may not show a grader.
      </p>
      <ul
        className="mt-4 max-h-72 list-none space-y-3 overflow-y-auto ps-0"
        role="list"
        aria-labelledby={id}
      >
        {events.map((e) => (
          <li
            key={e.id}
            role="listitem"
            className="border-s-2 border-indigo-200 ps-3 dark:border-indigo-800/80"
          >
            <div className="text-xs text-fg-subtle">
              <LocaleTime date={e.changedAt} dateOnly={false} />
            </div>
            <div className="text-sm font-medium text-fg-default">
              {actionLabel(e.action)}
            </div>
            {e.previousScore != null || e.newScore != null ? (
              <div className="text-sm text-fg-muted">
                {e.previousScore != null ? e.previousScore : '—'}
                {' → '}
                {e.newScore != null ? e.newScore : '—'}
              </div>
            ) : null}
            {e.previousStatus && e.newStatus && e.action !== 'deleted' ? (
              <div className="text-xs text-fg-muted">
                Visibility: {e.previousStatus} → {e.newStatus}
              </div>
            ) : null}
            {e.reason ? (
              <p className="mt-0.5 text-sm text-fg-muted">{e.reason}</p>
            ) : null}
            {e.changedBy ? (
              <div className="mt-0.5 text-xs text-fg-subtle">
                Changed by <span className="font-mono text-[0.7rem]">{e.changedBy}</span>
              </div>
            ) : null}
          </li>
        ))}
      </ul>
    </div>
  )
}
