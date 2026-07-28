import type { ReactNode } from 'react'
import type { ToolSyncStatus } from './use-tool-state'

export type ToolFrameProps = {
  label: string
  status: string
  syncStatus: ToolSyncStatus
  savedLabel: string
  savingLabel: string
  unsyncedLabel: string
  children: ReactNode
  onBlurCapture?: () => void
  busy?: boolean
  /** CT.4 instructor Responses affordance. */
  responsesLabel?: string
  onResponsesClick?: () => void
  /** CT.7 instructor Insights affordance. */
  insightsLabel?: string
  onInsightsClick?: () => void
  /** CT.7 graded badge for students when grade link enabled. */
  gradedBadgeLabel?: string
}

function statusChipClass(status: string): string {
  switch (status) {
    case 'completed':
      return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-200'
    case 'submitted':
      return 'bg-indigo-100 text-indigo-800 dark:bg-indigo-950/50 dark:text-indigo-200'
    case 'in_progress':
      return 'bg-amber-100 text-amber-900 dark:bg-amber-950/40 dark:text-amber-200'
    default:
      return 'bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-300'
  }
}

const actionBtnClass =
  'rounded-md px-2 py-1 text-xs font-medium text-indigo-700 hover:bg-indigo-50 dark:text-indigo-300 dark:hover:bg-indigo-950/40'

export function ToolFrame({
  label,
  status,
  syncStatus,
  savedLabel,
  savingLabel,
  unsyncedLabel,
  children,
  onBlurCapture,
  busy = false,
  responsesLabel,
  onResponsesClick,
  insightsLabel,
  onInsightsClick,
  gradedBadgeLabel,
}: ToolFrameProps) {
  const syncLabel =
    syncStatus === 'saving'
      ? savingLabel
      : syncStatus === 'unsynced'
        ? unsyncedLabel
        : syncStatus === 'saved'
          ? savedLabel
          : null

  return (
    <div
      role="group"
      aria-label={label}
      aria-busy={busy || syncStatus === 'saving' ? true : undefined}
      data-content-tool-frame=""
      onBlurCapture={onBlurCapture}
      className="not-prose my-4 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-neutral-700 dark:bg-neutral-900/80"
    >
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200 bg-slate-50/80 px-3.5 py-2.5 dark:border-neutral-700 dark:bg-neutral-900">
        <p className="truncate text-sm font-semibold text-slate-900 dark:text-neutral-100">{label}</p>
        <div className="flex flex-wrap items-center gap-1">
          {insightsLabel && onInsightsClick ? (
            <button
              type="button"
              data-testid="tool-insights-button"
              className={actionBtnClass}
              onClick={onInsightsClick}
            >
              {insightsLabel}
            </button>
          ) : null}
          {responsesLabel && onResponsesClick ? (
            <button
              type="button"
              data-testid="tool-responses-button"
              className={actionBtnClass}
              onClick={onResponsesClick}
            >
              {responsesLabel}
            </button>
          ) : null}
          {gradedBadgeLabel ? (
            <span
              data-testid="tool-graded-badge"
              className="rounded-md bg-amber-100 px-1.5 py-0.5 text-[11px] font-medium text-amber-900 dark:bg-amber-950/40 dark:text-amber-200"
            >
              {gradedBadgeLabel}
            </span>
          ) : null}
          <span
            className={`rounded-md px-1.5 py-0.5 text-[11px] font-medium uppercase tracking-wide ${statusChipClass(status)}`}
          >
            {status.replace(/_/g, ' ')}
          </span>
          {syncLabel ? (
            <span
              data-sync-status={syncStatus}
              className={
                syncStatus === 'unsynced'
                  ? 'rounded-md px-1.5 py-0.5 text-[11px] font-medium text-amber-800 dark:text-amber-200'
                  : 'rounded-md px-1.5 py-0.5 text-[11px] font-medium text-slate-500 dark:text-neutral-400'
              }
            >
              {syncLabel}
            </span>
          ) : null}
        </div>
      </div>
      <div className="px-3.5 py-3.5">{children}</div>
    </div>
  )
}
