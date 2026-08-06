import { ClipboardCheck } from 'lucide-react'
import { Link } from 'react-router-dom'
import type { ChecklistItem, ChecklistSummary } from '../../lib/course-checklist-api-schemas'
import { isOutstandingStatus } from '../../lib/course-checklist-api-schemas'
import { courseChecklistI18n } from '../../lib/course-checklist-i18n'

type ChecklistDashboardCardProps = {
  courseCode: string
  summary: ChecklistSummary | null
  topItems: ChecklistItem[]
  loading?: boolean
}

export function ChecklistDashboardCard({
  courseCode,
  summary,
  topItems,
  loading,
}: ChecklistDashboardCardProps) {
  const href = `/courses/${encodeURIComponent(courseCode)}/checklist`
  const outstanding = summary?.outstandingEssential ?? 0
  const outstandingItems = topItems.filter((i) => isOutstandingStatus(i.status)).slice(0, 3)

  return (
    <div className="rounded-2xl border border-border-default bg-surface-raised p-5 shadow-sm dark:border-border-default dark:bg-surface-raised sm:col-span-2 lg:col-span-1">
      <div className="flex items-center gap-2 text-warning-fg dark:text-amber-400">
        <ClipboardCheck className="h-5 w-5 shrink-0" aria-hidden />
        <span className="text-sm font-semibold text-fg-default">
          {courseChecklistI18n.dashboardTitle}
        </span>
      </div>

      {loading && !summary ? (
        <div className="mt-3 space-y-2" aria-busy="true">
          <div className="h-4 w-2/3 motion-safe:animate-pulse rounded bg-slate-200 dark:bg-neutral-700" />
          <div className="h-3 w-full motion-safe:animate-pulse rounded bg-surface-sunken" />
        </div>
      ) : outstanding <= 0 ? (
        <p className="mt-2 text-sm text-fg-muted">
          {courseChecklistI18n.dashboardComplete}
        </p>
      ) : (
        <>
          <p className="mt-2 text-sm text-fg-muted">
            {courseChecklistI18n.dashboardProgress(summary?.done ?? 0, summary?.total ?? 0)}
          </p>
          {outstandingItems.length > 0 ? (
            <ul className="mt-3 space-y-1.5 text-sm text-fg-default">
              {outstandingItems.map((item) => (
                <li key={item.id} className="truncate">
                  · {item.title}
                </li>
              ))}
            </ul>
          ) : null}
        </>
      )}

      <Link
        to={href}
        className="mt-4 inline-flex text-xs font-semibold text-amber-800 hover:text-warning-fg dark:hover:text-amber-200"
      >
        {courseChecklistI18n.dashboardViewAll} →
      </Link>
    </div>
  )
}
