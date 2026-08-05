import { ChevronDown, ChevronRight } from 'lucide-react'
import type { ChecklistCategory, ChecklistItem } from '../../../lib/course-checklist-api-schemas'
import {
  isOutstandingStatus,
  visibleChecklistItems,
} from '../../../lib/course-checklist-api-schemas'
import { courseChecklistI18n } from '../../../lib/course-checklist-i18n'
import { ChecklistItemRow } from './checklist-item-row'

type ChecklistCategorySectionProps = {
  category: ChecklistCategory
  expanded: boolean
  onToggle: () => void
  showCompleted: boolean
  itemErrors: Record<string, string>
  busyItemId: string | null
  highlightItemId: string | null
  onDismiss: (item: ChecklistItem) => void
  onRecheck: (item: ChecklistItem) => void
  onAssist?: (item: ChecklistItem) => void
  hideAiActions?: boolean
}

export function ChecklistCategorySection({
  category,
  expanded,
  onToggle,
  showCompleted,
  itemErrors,
  busyItemId,
  highlightItemId,
  onDismiss,
  onRecheck,
  onAssist,
  hideAiActions,
}: ChecklistCategorySectionProps) {
  const outstanding = category.items.filter((i) => isOutstandingStatus(i.status)).length
  const visibleItems = visibleChecklistItems(category.items, showCompleted)
  const panelId = `checklist-cat-${category.id}`

  // Nothing to show in this category while completed are hidden.
  if (visibleItems.length === 0) return null

  return (
    <section aria-labelledby={`${panelId}-heading`} className="border-b border-slate-200 py-4 last:border-0 dark:border-neutral-800">
      <h2
        id={`${panelId}-heading`}
        className="sticky top-0 z-10 -mx-1 bg-[var(--lx-page-bg,theme(colors.slate.50))] px-1 py-1 dark:bg-[var(--lx-page-bg,theme(colors.neutral.950))] md:static md:bg-transparent"
      >
        <button
          type="button"
          className="flex w-full min-h-11 items-center justify-between gap-3 text-start"
          aria-expanded={expanded}
          aria-controls={panelId}
          onClick={onToggle}
        >
          <span className="inline-flex items-center gap-2 text-base font-semibold text-slate-900 dark:text-neutral-50">
            {expanded ? (
              <ChevronDown className="h-4 w-4 shrink-0" aria-hidden />
            ) : (
              <ChevronRight className="h-4 w-4 shrink-0" aria-hidden />
            )}
            {category.title}
          </span>
          <span className="shrink-0 text-xs font-medium text-slate-500 dark:text-neutral-400">
            {courseChecklistI18n.outstandingCount(outstanding)}
          </span>
        </button>
      </h2>
      {expanded ? (
        <ul id={panelId} className="mt-2 divide-y divide-slate-100 dark:divide-neutral-800">
          {visibleItems.map((item) => (
            <ChecklistItemRow
              key={item.id}
              item={item}
              busy={busyItemId === item.id}
              error={itemErrors[item.id] ?? null}
              highlighted={highlightItemId === item.id}
              onDismiss={onDismiss}
              onRecheck={onRecheck}
              onAssist={onAssist}
              hideAiActions={hideAiActions}
            />
          ))}
        </ul>
      ) : (
        <div id={panelId} hidden />
      )}
    </section>
  )
}
