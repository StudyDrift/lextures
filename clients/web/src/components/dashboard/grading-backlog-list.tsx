import { Link } from 'react-router-dom'
import { ClipboardCheck } from 'lucide-react'

export type GradingBacklogItem = {
  assignmentId: string
  assignmentTitle: string
  ungradedCount: number
  courseCode?: string
  courseTitle?: string
}

function hrefForAssignmentGrading(courseCode: string, assignmentId: string): string {
  return `/courses/${encodeURIComponent(courseCode)}/modules/assignment/${encodeURIComponent(assignmentId)}?preview=submissions`
}

function formatUngradedCount(count: number): string {
  return `${count} ungraded submission${count === 1 ? '' : 's'}`
}

type GradingBacklogListProps = {
  items: GradingBacklogItem[]
  /** Show course title above each assignment (global dashboard). */
  showCourse?: boolean
  emptyMessage?: string
}

export function GradingBacklogList({ items, showCourse = false, emptyMessage }: GradingBacklogListProps) {
  if (items.length === 0) {
    if (!emptyMessage) return null
    return <p className="text-sm text-slate-500 dark:text-neutral-400">{emptyMessage}</p>
  }

  return (
    <ul className="space-y-2">
      {items.map((item) => {
        const courseCode = item.courseCode
        if (!courseCode) return null
        return (
          <li key={`${courseCode}-${item.assignmentId}`}>
            <Link
              to={hrefForAssignmentGrading(courseCode, item.assignmentId)}
              className="flex items-start justify-between gap-3 rounded-xl border border-amber-100 bg-amber-50/60 px-3 py-2.5 text-sm transition hover:border-amber-200 hover:bg-amber-50 dark:border-amber-900/40 dark:bg-amber-950/30 dark:hover:border-amber-800 dark:hover:bg-amber-950/50"
            >
              <span className="min-w-0">
                {showCourse && item.courseTitle ? (
                  <span className="block text-xs font-medium text-slate-500 dark:text-neutral-400">
                    {item.courseTitle}
                  </span>
                ) : null}
                <span className="flex items-center gap-1.5 font-semibold text-slate-900 dark:text-neutral-100">
                  <ClipboardCheck className="h-3.5 w-3.5 shrink-0 text-amber-700 dark:text-amber-300" aria-hidden />
                  <span className="truncate">{item.assignmentTitle}</span>
                </span>
              </span>
              <span className="shrink-0 rounded-full bg-amber-200/80 px-2 py-0.5 text-xs font-semibold text-amber-950 dark:bg-amber-900/60 dark:text-amber-50">
                {formatUngradedCount(item.ungradedCount)}
              </span>
            </Link>
          </li>
        )
      })}
    </ul>
  )
}