import { useEffect, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'
import { ChecklistProgressBar } from '../../../components/checklist/checklist-progress'
import type { ChecklistItem } from '../../../lib/course-checklist-api-schemas'
import { isDoneStatus } from '../../../lib/course-checklist-api-schemas'
import { courseChecklistI18n } from '../../../lib/course-checklist-i18n'
import { draftWelcomeAnnouncement } from '../../../lib/course-checklist-api'
import { emitChecklistTelemetry } from '../../../lib/checklist-telemetry'
import { formatTimeAgoFromIso } from '../../../lib/format-time-ago'
import { hrefForTarget } from '../../../lib/use-focus-anchor'
import { LmsPage } from '../lms-page'
import { ChecklistCategorySection } from './checklist-category-section'
import { ChecklistDismissDialog } from './checklist-dismiss-dialog'
import { ChecklistDismissedSection } from './checklist-dismissed-section'
import { ChecklistMappingAssistDialog } from './checklist-mapping-assist-dialog'
import { useChecklistPage } from './use-checklist-page'

function ChecklistSkeleton() {
  return (
    <div className="space-y-4" aria-busy="true" aria-label="Loading checklist">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="space-y-2 rounded-lg border border-border-default p-4 dark:border-border-default">
          <div className="h-5 w-1/3 motion-safe:animate-pulse rounded bg-slate-200 dark:bg-neutral-700" />
          <div className="h-4 w-full motion-safe:animate-pulse rounded bg-surface-sunken" />
          <div className="h-4 w-5/6 motion-safe:animate-pulse rounded bg-surface-sunken" />
        </div>
      ))}
    </div>
  )
}

export default function CourseChecklistPage() {
  const { courseCode: raw } = useParams<{ courseCode: string }>()
  const courseCode = raw ? decodeURIComponent(raw) : undefined
  const location = useLocation()
  const navigate = useNavigate()
  const [highlightItemId, setHighlightItemId] = useState<string | null>(null)
  const [assistError, setAssistError] = useState<string | null>(null)
  const page = useChecklistPage(courseCode)

  const handleAssist = async (item: ChecklistItem) => {
    if (!courseCode || !item.action) return
    setAssistError(null)
    const kind = item.action.kind
    emitChecklistTelemetry('checklist_assist_started', { itemId: item.id, actionKind: kind })

    const manualHref = hrefForTarget(
      {
        route: item.target?.route,
        anchor: item.target?.anchor,
        entityKey: item.target?.entityKey,
      },
      { courseCode },
    )

    try {
      switch (kind) {
        case 'suggest_outcome_mappings':
          page.setMappingAssistItem(item)
          return
        case 'build_rubric_ai': {
          // Route to first high-stakes evidence row or the item target (existing generate-rubric flow).
          const row = item.evidence?.rows?.[0]
          const href =
            hrefForTarget(
              {
                route: row?.target?.route ?? item.target?.route,
                anchor: row?.target?.anchor ?? item.target?.anchor ?? 'assignment.rubric',
                entityKey: row?.target?.entityKey ?? item.target?.entityKey,
              },
              { courseCode },
            ) ?? manualHref
          if (href) navigate(href)
          else setAssistError(courseChecklistI18n.assistFailed)
          return
        }
        case 'draft_welcome': {
          const draft = await draftWelcomeAnnouncement(courseCode)
          const q = new URLSearchParams({
            draftSubject: draft.subject,
            draftBody: draft.body,
            channel: 'announcements',
          })
          navigate(`/courses/${encodeURIComponent(courseCode)}/feed?${q.toString()}`)
          return
        }
        case 'suggest_alt_text': {
          const href =
            hrefForTarget(
              {
                route: item.target?.route,
                anchor: item.target?.anchor,
                entityKey: item.target?.entityKey,
              },
              { courseCode },
            ) ?? manualHref
          if (href) navigate(href)
          else setAssistError(courseChecklistI18n.assistFailed)
          return
        }
        default:
          // Unknown action kind: render nothing / degrade to manual (FR-12).
          if (manualHref) navigate(manualHref)
          return
      }
    } catch (e) {
      setAssistError(e instanceof Error ? e.message : courseChecklistI18n.assistFailed)
      if (manualHref) {
        // degrade to manual path
      }
    }
  }

  useEffect(() => {
    const hash = location.hash.replace(/^#/, '')
    if (!hash.startsWith('item-') || !page.data) return
    const itemId = hash.slice('item-'.length)
    for (const cat of page.data.categories) {
      const match = cat.items.find((i) => i.id === itemId)
      if (match) {
        // Done items are hidden by default; reveal them so the deep link can land.
        if (isDoneStatus(match.status) && !page.showCompleted) {
          page.setShowCompleted(true)
        }
        page.ensureCategoryExpanded(cat.id)
        setHighlightItemId(itemId)
        break
      }
    }
    // intentionally only when data/hash change
    // eslint-disable-next-line react-hooks/exhaustive-deps -- ensureCategoryExpanded identity changes often
  }, [location.hash, page.data])

  // Scroll after layout so revealing completed items has time to mount the target.
  useEffect(() => {
    if (!highlightItemId) return
    const t = window.setTimeout(() => {
      document.getElementById(`item-${highlightItemId}`)?.scrollIntoView({
        behavior: 'smooth',
        block: 'center',
      })
    }, 50)
    return () => window.clearTimeout(t)
  }, [highlightItemId, page.showCompleted, page.data])

  if (page.loadState === 'forbidden') {
    return (
      <LmsPage title={courseChecklistI18n.pageTitle}>
        <p className="text-sm text-fg-muted">{courseChecklistI18n.noAccess}</p>
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
          <RefreshCw className={`h-4 w-4 ${page.refreshing ? 'motion-safe:animate-spin' : ''}`} aria-hidden />
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
            <p className="text-sm text-fg-muted">{courseChecklistI18n.catalogEmpty}</p>
          ) : null}

          {page.allDone ? (
            <div className="mb-6 rounded-xl border border-emerald-200 bg-emerald-50/80 p-5 dark:border-emerald-900 dark:bg-emerald-950/30">
              <p className="text-base font-semibold text-emerald-900 dark:text-emerald-100">
                {courseChecklistI18n.allDoneTitle}
              </p>
              <p className="mt-1 text-sm text-emerald-800 dark:text-emerald-200">
                {courseChecklistI18n.allDoneBody}
              </p>
              {summary.done > 0 ? (
                <button
                  type="button"
                  className="mt-3 inline-flex min-h-11 items-center text-sm font-semibold text-emerald-900 underline-offset-2 hover:underline dark:text-emerald-200"
                  onClick={() => page.setShowCompleted((v) => !v)}
                >
                  {page.showCompleted
                    ? courseChecklistI18n.hideCompleted
                    : courseChecklistI18n.showCompleted}
                </button>
              ) : null}
            </div>
          ) : null}

          {/* When work remains, still offer a way to review crossed-off items. */}
          {!page.allDone && summary.done > 0 ? (
            <div className="mb-4 flex justify-end">
              <button
                type="button"
                className="inline-flex min-h-11 items-center text-sm font-semibold text-fg-muted underline-offset-2 hover:underline dark:text-fg-muted"
                onClick={() => page.setShowCompleted((v) => !v)}
                aria-pressed={page.showCompleted}
              >
                {page.showCompleted
                  ? courseChecklistI18n.hideCompleted
                  : courseChecklistI18n.showCompleted}
              </button>
            </div>
          ) : null}

          {assistError ? (
            <p className="mb-4 text-sm text-danger-fg" role="alert">
              {assistError}
            </p>
          ) : null}

          {(!page.allDone || page.showCompleted) &&
            page.data.categories.map((cat) => (
              <ChecklistCategorySection
                key={cat.id}
                category={cat}
                expanded={!page.collapsed[cat.id]}
                onToggle={() => page.toggleCategory(cat.id)}
                showCompleted={page.showCompleted}
                itemErrors={page.itemErrors}
                busyItemId={page.busyItemId}
                highlightItemId={highlightItemId}
                onDismiss={(item) => page.setDismissTarget(item)}
                onRecheck={(item) => void page.onRecheck(item)}
                onAssist={(item) => void handleAssist(item)}
                hideAiActions={page.hideAiActions}
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

      {courseCode && page.mappingAssistItem ? (
        <ChecklistMappingAssistDialog
          courseCode={courseCode}
          itemId={page.mappingAssistItem.id}
          open={!!page.mappingAssistItem}
          onClose={() => page.setMappingAssistItem(null)}
          onApplied={() => {
            void page.onRefresh()
          }}
          manualHref={
            hrefForTarget(
              {
                route: page.mappingAssistItem.target?.route,
                anchor: page.mappingAssistItem.target?.anchor,
                entityKey: page.mappingAssistItem.target?.entityKey,
              },
              { courseCode },
            ) ?? null
          }
        />
      ) : null}
    </LmsPage>
  )
}
