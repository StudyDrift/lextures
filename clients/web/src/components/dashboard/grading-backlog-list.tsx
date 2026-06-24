import { Link } from 'react-router-dom'
import { ClipboardCheck } from 'lucide-react'

export type GradingBacklogItemType = 'assignment' | 'quiz'

export type GradingBacklogItem = {
  itemId: string
  itemType: GradingBacklogItemType
  /** @deprecated Use itemId — kept for API backward compatibility. */
  assignmentId: string
  assignmentTitle: string
  ungradedCount: number
  courseCode?: string
  courseTitle?: string
}

function hrefForGradingItem(courseCode: string, item: Pick<GradingBacklogItem, 'itemId' | 'itemType'>): string {
  const code = encodeURIComponent(courseCode)
  const id = encodeURIComponent(item.itemId)
  if (item.itemType === 'quiz') {
    return `/courses/${code}/gradebook?item=${id}`
  }
  return `/courses/${code}/modules/assignment/${id}?preview=submissions`
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
        const itemKey = `${courseCode}-${item.itemType}-${item.itemId}`
        return (
          <li key={itemKey}>
            <Link
              to={hrefForGradingItem(courseCode, item)}
              className="flex items-start justify-between gap-3 rounded-xl border border-amber-100 bg-amber-50/60 px-3 py-2.5 text-sm transition-[background-color,color,border-color] hover:border-amber-200 hover:bg-amber-50 dark:border-amber-900/40 dark:bg-amber-950/30 dark:hover:border-amber-800 dark:hover:bg-amber-950/50"
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
                  {item.itemType === 'quiz' ? (
                    <span className="shrink-0 rounded bg-amber-200/70 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-950 dark:bg-amber-900/50 dark:text-amber-100">
                      Quiz
                    </span>
                  ) : null}
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