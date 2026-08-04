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
    <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-neutral-700 dark:bg-neutral-900 sm:col-span-2 lg:col-span-1">
      <div className="flex items-center gap-2 text-amber-700 dark:text-amber-400">
        <ClipboardCheck className="h-5 w-5 shrink-0" aria-hidden />
        <span className="text-sm font-semibold text-slate-900 dark:text-neutral-50">
          {courseChecklistI18n.dashboardTitle}
        </span>
      </div>

      {loading && !summary ? (
        <div className="mt-3 space-y-2" aria-busy="true">
          <div className="h-4 w-2/3 animate-pulse rounded bg-slate-200 dark:bg-neutral-700" />
          <div className="h-3 w-full animate-pulse rounded bg-slate-100 dark:bg-neutral-800" />
        </div>
      ) : outstanding <= 0 ? (
        <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
          {courseChecklistI18n.dashboardComplete}
        </p>
      ) : (
        <>
          <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
            {courseChecklistI18n.dashboardProgress(summary?.done ?? 0, summary?.total ?? 0)}
          </p>
          {outstandingItems.length > 0 ? (
            <ul className="mt-3 space-y-1.5 text-sm text-slate-800 dark:text-neutral-200">
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
        className="mt-4 inline-flex text-xs font-semibold text-amber-800 hover:text-amber-700 dark:text-amber-300 dark:hover:text-amber-200"
      >
        {courseChecklistI18n.dashboardViewAll} →
      </Link>
    </div>
  )
}
