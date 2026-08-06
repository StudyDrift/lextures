import { ChevronDown, ChevronRight } from 'lucide-react'
import type { ChecklistItem } from '../../../lib/course-checklist-api-schemas'
import { courseChecklistI18n, dismissReasonLabel } from '../../../lib/course-checklist-i18n'
import { formatTimeAgoFromIso } from '../../../lib/format-time-ago'

type ChecklistDismissedSectionProps = {
  items: ChecklistItem[]
  expanded: boolean
  onToggle: () => void
  busyItemId: string | null
  itemErrors: Record<string, string>
  onRestore: (item: ChecklistItem) => void
}

export function ChecklistDismissedSection({
  items,
  expanded,
  onToggle,
  busyItemId,
  itemErrors,
  onRestore,
}: ChecklistDismissedSectionProps) {
  if (items.length === 0) return null
  const panelId = 'checklist-dismissed'

  return (
    <section aria-labelledby={`${panelId}-heading`} className="mt-6 border-t border-border-default pt-4 dark:border-border-subtle">
      <h2 id={`${panelId}-heading`}>
        <button
          type="button"
          className="flex w-full min-h-11 items-center justify-between gap-3 text-start"
          aria-expanded={expanded}
          aria-controls={panelId}
          onClick={onToggle}
        >
          <span className="inline-flex items-center gap-2 text-base font-semibold text-fg-muted">
            {expanded ? (
              <ChevronDown className="h-4 w-4 shrink-0" aria-hidden />
            ) : (
              <ChevronRight className="h-4 w-4 shrink-0" aria-hidden />
            )}
            {courseChecklistI18n.dismissedSection(items.length)}
          </span>
        </button>
      </h2>
      {expanded ? (
        <ul id={panelId} className="mt-2 space-y-3">
          {items.map((item) => {
            const d = item.dismissal
            return (
              <li
                key={item.id}
                className="rounded-lg border border-border-default bg-slate-50/80 p-3 dark:border-border-default/60"
              >
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-fg-default">
                      {item.title}
                    </p>
                    {d ? (
                      <p className="mt-1 text-xs text-fg-muted">
                        {courseChecklistI18n.dismissedBy(
                          d.byDisplayName || 'Someone',
                          formatTimeAgoFromIso(d.dismissedAt),
                        )}
                        {' · '}
                        {dismissReasonLabel(d.reason)}
                        {d.note ? ` — ${d.note}` : ''}
                      </p>
                    ) : null}
                    {itemErrors[item.id] ? (
                      <p className="mt-1 text-sm text-danger-fg" role="alert">
                        {itemErrors[item.id]}
                      </p>
                    ) : null}
                  </div>
                  <button
                    type="button"
                    disabled={busyItemId === item.id}
                    onClick={() => onRestore(item)}
                    className="inline-flex min-h-11 items-center rounded-lg border border-border-strong px-3 text-xs font-semibold text-fg-default hover:border-amber-300 dark:border-border-default dark:text-fg-default"
                  >
                    {busyItemId === item.id
                      ? courseChecklistI18n.restoring
                      : courseChecklistI18n.restore}
                  </button>
                </div>
              </li>
            )
          })}
        </ul>
      ) : (
        <div id={panelId} hidden />
      )}
    </section>
  )
}
