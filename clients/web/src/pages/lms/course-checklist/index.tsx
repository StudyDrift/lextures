import { useEffect, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import { useLocation, useParams } from 'react-router-dom'
import { ChecklistProgressBar } from '../../../components/checklist/checklist-progress'
import { courseChecklistI18n } from '../../../lib/course-checklist-i18n'
import { formatTimeAgoFromIso } from '../../../lib/format-time-ago'
import { LmsPage } from '../lms-page'
import { ChecklistCategorySection } from './checklist-category-section'
import { ChecklistDismissDialog } from './checklist-dismiss-dialog'
import { ChecklistDismissedSection } from './checklist-dismissed-section'
import { useChecklistPage } from './use-checklist-page'

function ChecklistSkeleton() {
  return (
    <div className="space-y-4" aria-busy="true" aria-label="Loading checklist">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="space-y-2 rounded-lg border border-slate-200 p-4 dark:border-neutral-700">
          <div className="h-5 w-1/3 animate-pulse rounded bg-slate-200 dark:bg-neutral-700" />
          <div className="h-4 w-full animate-pulse rounded bg-slate-100 dark:bg-neutral-800" />
          <div className="h-4 w-5/6 animate-pulse rounded bg-slate-100 dark:bg-neutral-800" />
        </div>
      ))}
    </div>
  )
}

export default function CourseChecklistPage() {
  const { courseCode: raw } = useParams<{ courseCode: string }>()
  const courseCode = raw ? decodeURIComponent(raw) : undefined
  const location = useLocation()
  const [highlightItemId, setHighlightItemId] = useState<string | null>(null)
  const page = useChecklistPage(courseCode)

  useEffect(() => {
    const hash = location.hash.replace(/^#/, '')
    if (!hash.startsWith('item-') || !page.data) return
    const itemId = hash.slice('item-'.length)
    for (const cat of page.data.categories) {
      if (cat.items.some((i) => i.id === itemId)) {
        page.ensureCategoryExpanded(cat.id)
        setHighlightItemId(itemId)
        break
      }
    }
    const t = window.setTimeout(() => {
      document.getElementById(`item-${itemId}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }, 50)
    return () => window.clearTimeout(t)
    // intentionally only when data/hash change
    // eslint-disable-next-line react-hooks/exhaustive-deps -- ensureCategoryExpanded identity changes often
  }, [location.hash, page.data])

  if (page.loadState === 'forbidden') {
    return (
      <LmsPage title={courseChecklistI18n.pageTitle}>
        <p className="text-sm text-slate-600 dark:text-neutral-400">{courseChecklistI18n.noAccess}</p>
      </LmsPage>
    )
  }

  const summary = page.data?.summary
  const checkedLabel = summary
    ? courseChecklistI18n.checkedAgo(formatTimeAgoFromIso(summary.computedAt).toLowerCase())
    : undefined

  return (
    <LmsPage
      title={courseChecklistI18n.pageTitle}
      actions={
        <button
          type="button"
          disabled={page.refreshing || page.loadState === 'loading'}
          onClick={() => void page.onRefresh()}
          className="inline-flex min-h-11 items-center gap-2 rounded-lg bg-amber-700 px-4 text-sm font-semibold text-white hover:bg-amber-600 disabled:opacity-60"
        >
          <RefreshCw className={`h-4 w-4 ${page.refreshing ? 'animate-spin' : ''}`} aria-hidden />
          {page.refreshing ? courseChecklistI18n.rechecking : courseChecklistI18n.recheck}
        </button>
      }
    >
      <div className="sr-only" aria-live="polite">
        {page.liveMessage}
      </div>

      {page.loadState === 'loading' || page.loadState === 'idle' ? <ChecklistSkeleton /> : null}

      {page.loadState === 'error' ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-900 dark:bg-red-950/40">
          <p className="text-sm text-red-800 dark:text-red-200">{page.error ?? courseChecklistI18n.loadError}</p>
          <button
            type="button"
            className="mt-3 inline-flex min-h-11 items-center rounded-lg border border-red-300 px-3 text-sm font-semibold text-red-800 dark:border-red-800 dark:text-red-200"
            onClick={() => void page.load()}
          >
            {courseChecklistI18n.retry}
          </button>
        </div>
      ) : null}

      {page.loadState === 'ready' && page.data && summary ? (
        <>
          <ChecklistProgressBar
            done={summary.done}
            total={summary.total}
            outstandingTotal={summary.outstandingTotal}
            checkedLabel={checkedLabel}
            className="mb-6"
          />

          {page.catalogEmpty ? (
            <p className="text-sm text-slate-600 dark:text-neutral-400">{courseChecklistI18n.catalogEmpty}</p>
          ) : null}

          {page.allDone ? (
            <div className="mb-6 rounded-xl border border-emerald-200 bg-emerald-50/80 p-5 dark:border-emerald-900 dark:bg-emerald-950/30">
              <p className="text-base font-semibold text-emerald-900 dark:text-emerald-100">
                {courseChecklistI18n.allDoneTitle}
              </p>
              <p className="mt-1 text-sm text-emerald-800 dark:text-emerald-200">
                {courseChecklistI18n.allDoneBody}
              </p>
              <button
                type="button"
                className="mt-3 inline-flex min-h-11 items-center text-sm font-semibold text-emerald-900 underline-offset-2 hover:underline dark:text-emerald-200"
                onClick={() => page.setShowCompletedWhenAllDone((v) => !v)}
              >
                {page.showCompletedWhenAllDone
                  ? courseChecklistI18n.hideCompleted
                  : courseChecklistI18n.showCompleted}
              </button>
            </div>
          ) : null}

          {(!page.allDone || page.showCompletedWhenAllDone) &&
            page.data.categories.map((cat) => (
              <ChecklistCategorySection
                key={cat.id}
                category={cat}
                expanded={!page.collapsed[cat.id]}
                onToggle={() => page.toggleCategory(cat.id)}
                itemErrors={page.itemErrors}
                busyItemId={page.busyItemId}
                highlightItemId={highlightItemId}
                onDismiss={(item) => page.setDismissTarget(item)}
                onRecheck={(item) => void page.onRecheck(item)}
              />
            ))}

          <ChecklistDismissedSection
            items={page.data.dismissed}
            expanded={page.dismissedOpen}
            onToggle={() => page.setDismissedOpen((v) => !v)}
            busyItemId={page.busyItemId}
            itemErrors={page.itemErrors}
            onRestore={(item) => void page.onRestore(item)}
          />
        </>
      ) : null}

      <ChecklistDismissDialog
        open={!!page.dismissTarget}
        itemTitle={page.dismissTarget?.title ?? ''}
        busy={page.dismissBusy}
        error={page.dismissError}
        onClose={() => page.setDismissTarget(null)}
        onConfirm={(body) => void page.onDismissConfirm(body)}
      />
    </LmsPage>
  )
}
